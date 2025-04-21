package mobileconf

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/escrow-tf/steam/api"
	"github.com/escrow-tf/steam/steamid"
	"github.com/escrow-tf/steam/steamlang"
	"github.com/escrow-tf/steam/totp"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type ConfirmationType int

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
}

func NewClient(totpState *totp.State, steamID steamid.SteamID, transport *api.Transport) (*Client, error) {
	//jar, err := cookiejar.New(nil)
	//if err != nil {
	//	return nil, err
	//}
	//
	//baseUrl, err := url.Parse("https://steamcommunity.com/")
	//if err != nil {
	//	return nil, err
	//}
	//
	//jar.SetCookies(baseUrl, []*http.Cookie{
	//	&http.Cookie{
	//		Name:  "sessionid",
	//		Value: url.QueryEscape(sessionId),
	//	},
	//	&http.Cookie{
	//		Name:  "steamLogin",
	//		Value: steamLogin,
	//	},
	//	&http.Cookie{
	//		Name:  "steamLoginSecure",
	//		Value: steamLoginSecure,
	//	},
	//})
	//
	//client := &http.client{
	//	Jar: jar,
	//}

	return &Client{
		totpState: totpState,
		steamID:   steamID,
		client:    transport.HttpClient(),
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
}

func (c Client) SendMobileConfRequest(request Request, response any) error {
	totpTime := totp.Time(0)
	key, err := c.totpState.GenerateConfirmationKey(totpTime, []byte(request.Tag))
	if err != nil {
		return fmt.Errorf("totpState.GenerateConfirmationKey: %v", err)
	}

	parameters := make(url.Values)

	parameters.Add("k", base64.StdEncoding.EncodeToString(key))
	parameters.Add("p", totp.GetDeviceId(c.steamID.String()))
	parameters.Add("c", c.steamID.String())
	parameters.Add("t", strconv.Itoa(int(totpTime.Unix())))
	parameters.Add("tag", request.Tag)
	parameters.Add("m", "react")

	if request.Operation != nil {
		parameters.Add("op", request.Operation.Operation)
		parameters.Add("cid", request.Operation.ID)
		parameters.Add("ck", request.Operation.Nonce)
	}

	var httpResponse *http.Response
	var httpError error
	if request.Posts {
		requestUrl := fmt.Sprintf("https://steamcommunity.com/mobileconf/%s", request.Path)
		httpResponse, httpError = c.client.PostForm(requestUrl, parameters)
	} else {
		queryString := parameters.Encode()
		requestUrl := fmt.Sprintf("https://steamcommunity.com/mobileconf/%s?%s", request.Path, queryString)
		httpResponse, httpError = c.client.Get(requestUrl)
	}

	if httpError != nil {
		return fmt.Errorf("mobileconf request errored: %v", err)
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

func (c Client) GetList() (GetListResponse, error) {
	request := Request{
		Posts:     false,
		Path:      "getlist",
		Tag:       "list",
		Operation: nil,
	}

	response := GetListResponse{}
	err := c.SendMobileConfRequest(request, &response)
	if err != nil {
		return GetListResponse{}, fmt.Errorf("getlist mobile conf request failed: %v", err)
	}

	return response, nil
}

type DetailsPageResponse struct {
	// TODO: implement me
}

func (c Client) GetDetailsPage(id string) (DetailsPageResponse, error) {
	request := Request{
		Posts:     false,
		Path:      "detailspage/" + id,
		Tag:       "details",
		Operation: nil,
	}

	response := DetailsPageResponse{}
	err := c.SendMobileConfRequest(request, &response)
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

func (c Client) Accept(id, nonce string) (AcceptResponse, error) {
	request := Request{
		Posts: true,
		Path:  "multiajaxop",
		Tag:   "accept",
		Operation: &Operation{
			Operation: "accept",
			ID:        id,
			Nonce:     nonce,
		},
	}

	response := AcceptResponse{}
	err := c.SendMobileConfRequest(request, &response)
	if err != nil {
		return AcceptResponse{}, fmt.Errorf("accept mobile conf request failed: %v", err)
	}

	return response, nil
}

type DeclineResponse struct {
	Success   bool   `json:"success"`
	NeedsAuth bool   `json:"needsauth,omitempty"`
	Message   string `json:"message,omitempty"`
	Details   string `json:"details,omitempty"`
}

func (c Client) Decline(id, nonce string) (DeclineResponse, error) {
	request := Request{
		Posts: true,
		Path:  "multiajaxop",
		Tag:   "cancel",
		Operation: &Operation{
			Operation: "cancel",
			ID:        id,
			Nonce:     nonce,
		},
	}

	response := DeclineResponse{}
	err := c.SendMobileConfRequest(request, &response)
	if err != nil {
		return DeclineResponse{}, fmt.Errorf("decline mobile conf request failed: %v", err)
	}
	return response, nil
}
