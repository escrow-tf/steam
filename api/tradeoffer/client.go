package tradeoffer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/escrow-tf/steam/api"
	"github.com/escrow-tf/steam/steamid"
	"github.com/escrow-tf/steam/steamlang"
)

type SessionIdFunc func(transport api.Transport) (string, error)

type Client struct {
	Transport     api.Transport
	SessionIdFunc SessionIdFunc
}

type ActionResponse struct {
	TradeOfferId uint64 `json:"tradeofferid,string"`
}

type ActionRequest struct {
	id        uint64
	verb      string
	sessionId string
}

func (t ActionRequest) EnsureResponseSuccess(httpResponse *http.Response) error {
	return steamlang.EnsureSuccessResponse(httpResponse)
}

func (t ActionRequest) Retryable() bool {
	return false
}

func (t ActionRequest) RequiresApiKey() bool {
	return false
}

func (t ActionRequest) Method() string {
	return http.MethodPost
}

func (t ActionRequest) Url() string {
	return fmt.Sprintf("https://steamcommunity.com/tradeoffer/%d/%s", t.id, t.verb)
}

func (t ActionRequest) Values() (url.Values, error) {
	return url.Values{
		"sessionid": []string{t.sessionId},
	}, nil
}

func (t ActionRequest) Headers() (http.Header, error) {
	// TODO: do we need referer when acting on a request?
	return nil, nil
}

func (c *Client) act(ctx context.Context, id uint64, verb string) (*ActionResponse, error) {
	sessionId, sessionIdErr := c.SessionIdFunc(c.Transport)
	if sessionIdErr != nil {
		return nil, fmt.Errorf("error retrieving sessionId from transport: %v", sessionIdErr)
	}

	request := ActionRequest{
		id:        id,
		verb:      verb,
		sessionId: sessionId,
	}
	var response ActionResponse
	sendErr := c.Transport.Send(ctx, request, &response)
	if sendErr != nil {
		return nil, sendErr
	}
	return &response, nil
}

func (c *Client) Accept(ctx context.Context, id uint64) (*ActionResponse, error) {
	return c.act(ctx, id, "accept")
}

func (c *Client) Decline(ctx context.Context, id uint64) (*ActionResponse, error) {
	return c.act(ctx, id, "decline")
}

func (c *Client) Cancel(ctx context.Context, id uint64) (*ActionResponse, error) {
	return c.act(ctx, id, "cancel")
}

type CreateParams struct {
	AccessToken string `json:"trade_offer_access_token"`
}

type Offer struct {
	NewVersion bool  `json:"newversion"`
	Version    int   `json:"version"`
	Me         Party `json:"me"`
	Them       Party `json:"them"`
}

type Party struct {
	Assets   []Item     `json:"assets"`
	Currency []struct{} `json:"currency"`
	Ready    bool       `json:"ready"`
}

type Item struct {
	AppId      uint64 `json:"appid"`
	ContextId  string `json:"contextid"`
	Amount     uint64 `json:"amount"`
	AssetId    string `json:"assetid,omitempty"`
	CurrencyId string `json:"currencyid,omitempty"`
}

type CreateRequest struct {
	SessionId        string
	ServerId         string
	Partner          string
	Message          string
	OfferJson        string
	CreateParamsJson string
	PartnerAccountId uint32
	PartnerToken     string
}

func (c CreateRequest) EnsureResponseSuccess(httpResponse *http.Response) error {
	// Create will check strError in this case
	if httpResponse.StatusCode == 500 {
		return nil
	}
	return steamlang.EnsureSuccessResponse(httpResponse)
}

func (c CreateRequest) Retryable() bool {
	return false
}

func (c CreateRequest) RequiresApiKey() bool {
	return false
}

func (c CreateRequest) Method() string {
	return http.MethodPost
}

func (c CreateRequest) Url() string {
	return "https://steamcommunity.com/tradeoffer/new/send"
}

func (c CreateRequest) Values() (url.Values, error) {
	values := make(url.Values)
	values.Add("sessionid", c.SessionId)
	values.Add("serverid", c.ServerId)
	values.Add("partner", c.Partner)
	values.Add("tradeoffermessage", c.Message)
	values.Add("json_tradeoffer", c.OfferJson)
	values.Add("trade_offer_create_params", c.CreateParamsJson)
	return values, nil
}

func (c CreateRequest) Headers() (http.Header, error) {
	encodedPartnerAccountId := strconv.FormatUint(uint64(c.PartnerAccountId), 10)
	encodedPartnerToken := url.QueryEscape(c.PartnerToken)
	referer := fmt.Sprintf(
		"https://steamcommunity.com/tradeoffer/new/?partner=%s&token=%s",
		encodedPartnerAccountId,
		encodedPartnerToken,
	)
	return http.Header{
		"Referer": []string{referer},
	}, nil
}

type CreateResponse struct {
	Error        string `json:"strError"`
	TradeOfferId uint64 `json:"tradeOfferId,string"`
}

func (c *Client) Create(
	ctx context.Context,
	other steamid.SteamID,
	partnerToken string,
	myItems, theirItems []Item,
	message string,
) (CreateResponse, error) {
	sessionId, sessionIdErr := c.SessionIdFunc(c.Transport)
	if sessionIdErr != nil {
		return CreateResponse{}, fmt.Errorf("error retrieving sessionId from transport: %v", sessionIdErr)
	}

	offer := Offer{
		NewVersion: true,
		Version:    3,
		Me: Party{
			Assets:   myItems,
			Currency: []struct{}{},
			Ready:    false,
		},
		Them: Party{
			Assets:   theirItems,
			Currency: []struct{}{},
			Ready:    false,
		},
	}

	offerJson, offerJsonErr := json.Marshal(offer)
	if offerJsonErr != nil {
		return CreateResponse{}, fmt.Errorf("error marshalling Offer: %v", offerJsonErr)
	}

	createParams := CreateParams{
		AccessToken: partnerToken,
	}

	createParamsJson, createParamsJsonErr := json.Marshal(createParams)
	if createParamsJsonErr != nil {
		return CreateResponse{}, fmt.Errorf("error marshalling CreateParams: %v", createParamsJsonErr)
	}

	request := CreateRequest{
		SessionId:        sessionId,
		ServerId:         "1",
		Partner:          other.String(),
		Message:          message,
		OfferJson:        string(offerJson),
		CreateParamsJson: string(createParamsJson),
		PartnerAccountId: other.AccountId(),
		PartnerToken:     partnerToken,
	}
	var response CreateResponse
	sendErr := c.Transport.Send(ctx, request, &response)
	if sendErr != nil {
		return CreateResponse{}, fmt.Errorf("error creating new Offer: %v", sendErr)
	}

	// There are a couple of error formats we're likely to receive back:
	// A generic error message with an error number at the end:
	//  {"strError":"There was an error sending your trade offer.  Please try again later. (ERROR NUMBER)"}
	//
	// A specific error message:
	//  {"strError":"You have sent too many trade offers, or have too many outstanding trade offers with
	//  snuppy. Please cancel some before sending more."}
	//
	// In both of these cases, steam returns a 500 error code despite these clearly being 4xx errors, and doesn't
	// give us an EResult header in the response.

	if strings.HasPrefix(response.Error, "There was an error sending your trade offer.  Please try again later. (") {
		leftParenIdx := strings.Index(response.Error, "(")
		rightParenIdx := strings.Index(response.Error, ")")
		eResultString := response.Error[leftParenIdx:rightParenIdx]
		eResult, err := strconv.ParseInt(eResultString, 10, 32)
		if err != nil {
			return CreateResponse{}, fmt.Errorf("error sending offer: %v", response.Error)
		}

		switch steamlang.EResult(eResult) {
		case steamlang.InvalidStateResult:
			return CreateResponse{}, InvalidStateError
		case steamlang.AccessDeniedResult:
			return CreateResponse{}, AccessDeniedError
		case steamlang.TimeoutResult:
			return CreateResponse{}, TimeoutError
		case steamlang.ServiceUnavailableResult:
			return CreateResponse{}, ServiceUnavailableError
		case steamlang.LimitExceededResult:
			return CreateResponse{}, TooManyTradeOffersError
		case steamlang.RevokedResult:
			return CreateResponse{}, ItemsDontExistError
		case steamlang.AlreadyRedeemedResult:
			return CreateResponse{}, ChangedPersonaNameRecentlyError
		}

		return CreateResponse{}, steamlang.EResultError(steamlang.EResult(eResult))
	}

	if strings.HasPrefix(
		response.Error,
		"You have sent too many trade offers, or have too many outstanding trade offers with",
	) {
		return CreateResponse{}, TooManyTradeOffersError
	}

	if response.Error != "" {
		return CreateResponse{}, fmt.Errorf("error sending offer: %v", response.Error)
	}

	if response.TradeOfferId == 0 {
		return CreateResponse{}, fmt.Errorf("error creating offer: steam returned tradeofferid 0")
	}

	return response, nil
}
