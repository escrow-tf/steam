package mobileconf

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/escrow-tf/steam/api"
	"github.com/escrow-tf/steam/api/twofactor"
	"github.com/escrow-tf/steam/steamid"
	"github.com/escrow-tf/steam/steamlang"
	"github.com/escrow-tf/steam/totp"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ConfirmationType int

//goland:noinspection GoUnusedConst
const (
	InvalidConfirmationType ConfirmationType = iota
	TradeConfirmationType
	MarketListingConfirmationType
	OtherConfirmationType
)

type Client struct {
	totpState *totp.State
	steamID   steamid.SteamID
	client    *http.Client
	twoFactor *twofactor.Client
	transport api.Transport
}

func NewClient(
	totpState *totp.State,
	steamID steamid.SteamID,
	twoFactorClient *twofactor.Client,
	transport api.Transport,
) (*Client, error) {
	return &Client{
		totpState: totpState,
		steamID:   steamID,
		client:    transport.HttpClient(),
		twoFactor: twoFactorClient,
		transport: transport,
	}, nil
}

type Operation struct {
	Operation string
	ID        string
	Nonce     string
}

type Request struct {
	Operation *Operation
	Posts     bool
	Path      string
	Tag       string

	key      []byte
	steamID  steamid.SteamID
	totpTime time.Time
}

func (r Request) EnsureResponseSuccess(httpResponse *http.Response) error {
	return steamlang.EnsureSuccessResponse(httpResponse)
}

func (r Request) Headers() (http.Header, error) {
	return nil, nil
}

func (r Request) Retryable() bool {
	return !r.Posts
}

func (r Request) RequiresApiKey() bool {
	return false
}

func (r Request) Method() string {
	if r.Posts {
		return http.MethodPost
	}

	return http.MethodGet
}

func (r Request) Url() string {
	return fmt.Sprintf("https://steamcommunity.com/mobileconf/%s", r.Path)
}

func (r Request) Values() (url.Values, error) {
	parameters := make(url.Values)

	parameters.Add("k", base64.StdEncoding.EncodeToString(r.key))
	parameters.Add("p", totp.GetDeviceId(r.steamID.String()))
	parameters.Add("c", r.steamID.String())
	parameters.Add("t", strconv.Itoa(int(r.totpTime.Unix())))
	parameters.Add("tag", r.Tag)
	parameters.Add("m", "react")

	if r.Operation != nil {
		parameters.Add("op", r.Operation.Operation)
		parameters.Add("cid", r.Operation.ID)
		parameters.Add("ck", r.Operation.Nonce)
	}

	return parameters, nil
}

func (c *Client) SendMobileConfRequest(ctx context.Context, request Request, response any) error {
	// totpTime := totp.Time(0)
	totpTime, steamTimeErr := c.twoFactor.SteamTime()
	if steamTimeErr != nil {
		return steamTimeErr
	}

	key, err := c.totpState.GenerateConfirmationKey(totpTime, []byte(request.Tag))
	if err != nil {
		return fmt.Errorf("totpState.GenerateConfirmationKey: %v", err)
	}

	parameters := make(url.Values)
	parameters.Add("p", totp.GetDeviceId(c.steamID.String()))
	parameters.Add("a", c.steamID.String())
	parameters.Add("k", base64.StdEncoding.EncodeToString(key))
	// parameters.Add("c", c.steamID.String())
	parameters.Add("t", strconv.Itoa(int(totpTime.Unix()))) // TODO: unix or local?
	parameters.Add("m", "react")
	parameters.Add("tag", request.Tag)

	if request.Operation != nil {
		parameters.Add("op", request.Operation.Operation)
		parameters.Add("cid", request.Operation.ID)
		parameters.Add("ck", request.Operation.Nonce)
	}

	var httpRequest *http.Request
	var httpRequestErr error
	if request.Posts {
		requestUrl := fmt.Sprintf("https://steamcommunity.com/mobileconf/%s", request.Path)
		httpRequest, httpRequestErr = http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			requestUrl,
			strings.NewReader(parameters.Encode()),
		)
		if httpRequestErr == nil {
			httpRequest.Header.Add("Content-Type", api.FormContentType)
		}
	} else {
		queryString := parameters.Encode()
		requestUrl := fmt.Sprintf("https://steamcommunity.com/mobileconf/%s?%s", request.Path, queryString)
		httpRequest, httpRequestErr = http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			requestUrl,
			strings.NewReader(parameters.Encode()),
		)
	}

	if httpRequestErr != nil {
		return fmt.Errorf("mobileconf request errored: %v", httpRequestErr)
	}

	httpResponse, httpError := c.client.Do(httpRequest)

	if httpError != nil {
		return fmt.Errorf("mobileconf request errored: %v", httpError)
	}

	if err = steamlang.EnsureSuccessResponse(httpResponse); err != nil {
		return err
	}

	if err = steamlang.EnsureEResultResponse(httpResponse); err != nil {
		return err
	}

	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return fmt.Errorf("couldnt read mobileconf response body: %v", err)
	}

	err = json.Unmarshal(responseBody, response)
	if err != nil {
		return fmt.Errorf("couldnt unmarshal mobileconf response body: %v", err)
	}

	return nil
}

type GetListResponse struct {
	Success       bool   `json:"success"`
	NeedsAuth     bool   `json:"needsauth,omitempty"`
	Message       string `json:"message,omitempty"`
	Details       string `json:"details,omitempty"`
	Confirmations []struct {
		ID           string           `json:"id"`
		Type         ConfirmationType `json:"type"`
		CreatorID    string           `json:"creator_id"`
		Nonce        string           `json:"nonce"`
		TypeName     string           `json:"type_name"`
		Headline     string           `json:"headline"`
		Summary      []string         `json:"summary"`
		CreationTime int64            `json:"creation_time"`
		Icon         string           `json:"icon"`
	} `json:"conf"`
}

func (c *Client) GetList(ctx context.Context) (GetListResponse, error) {
	request := Request{
		Posts:     false,
		Path:      "getlist",
		Tag:       "conf",
		Operation: nil,
	}

	response := GetListResponse{}
	err := c.SendMobileConfRequest(ctx, request, &response)
	if err != nil {
		return GetListResponse{}, fmt.Errorf("getlist mobile conf request failed: %v", err)
	}

	if !response.Success {
		return GetListResponse{}, fmt.Errorf("getlist mobile conf request failed: %v", response.Message)
	}

	return response, nil
}

type DetailsPageResponse struct {
	TradeOffer *struct {
		Id string `json:"id,omitempty"`
	} `json:"tradeoffer,omitempty"`
}

func (c *Client) GetDetailsPage(ctx context.Context, id string) (DetailsPageResponse, error) {
	request := Request{
		Posts:     false,
		Path:      "detailspage/" + id,
		Tag:       "details",
		Operation: nil,
	}

	response := DetailsPageResponse{}
	err := c.SendMobileConfRequest(ctx, request, &response)
	if err != nil {
		return DetailsPageResponse{}, fmt.Errorf("detailspage mobile conf request failed: %v", err)
	}

	return response, nil
}

type AcceptResponse struct {
	Success   bool   `json:"success"`
	NeedsAuth bool   `json:"needsauth,omitempty"`
	Message   string `json:"message,omitempty"`
	Details   string `json:"details,omitempty"`
}

func (c *Client) Accept(ctx context.Context, id, nonce string) (AcceptResponse, error) {
	request := Request{
		Posts: false,
		Path:  "ajaxop",
		Tag:   "accept",
		Operation: &Operation{
			Operation: "allow",
			ID:        id,
			Nonce:     nonce,
		},
	}

	response := AcceptResponse{}
	err := c.SendMobileConfRequest(ctx, request, &response)
	if err != nil {
		return AcceptResponse{}, fmt.Errorf("accept mobile conf request failed: %v", err)
	}

	if !response.Success {
		return AcceptResponse{}, fmt.Errorf("accept mobile conf request failed: %v", response.Message)
	}

	return response, nil
}

type DeclineResponse struct {
	Success   bool   `json:"success"`
	NeedsAuth bool   `json:"needsauth,omitempty"`
	Message   string `json:"message,omitempty"`
	Details   string `json:"details,omitempty"`
}

func (c *Client) Decline(ctx context.Context, id, nonce string) (DeclineResponse, error) {
	request := Request{
		Posts: false,
		Path:  "ajaxop",
		Tag:   "reject",
		Operation: &Operation{
			Operation: "cancel",
			ID:        id,
			Nonce:     nonce,
		},
	}

	response := DeclineResponse{}
	err := c.SendMobileConfRequest(ctx, request, &response)
	if err != nil {
		return DeclineResponse{}, fmt.Errorf("decline mobile conf request failed: %v", err)
	}
	return response, nil
}
