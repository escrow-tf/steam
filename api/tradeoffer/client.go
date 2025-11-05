package tradeoffer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/escrow-tf/steam/api"
	"github.com/escrow-tf/steam/steamid"
	"github.com/escrow-tf/steam/steamlang"
	"github.com/rotisserie/eris"
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

func (t ActionRequest) CacheTTL() time.Duration {
	return 0
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

func (t ActionRequest) OldValues() (url.Values, error) {
	return url.Values{
		"sessionid": []string{t.sessionId},
	}, nil
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
		return nil, eris.Errorf("error retrieving sessionId from transport: %v", sessionIdErr)
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

func (c CreateRequest) Values() (url.Values, error) {
	return c.OldValues()
}

func (c CreateRequest) CacheTTL() time.Duration {
	return 0
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

func (c CreateRequest) OldValues() (url.Values, error) {
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
		return CreateResponse{}, eris.Errorf("error retrieving sessionId from transport: %v", sessionIdErr)
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
		return CreateResponse{}, eris.Errorf("error marshalling Offer: %v", offerJsonErr)
	}

	createParams := CreateParams{
		AccessToken: partnerToken,
	}

	createParamsJson, createParamsJsonErr := json.Marshal(createParams)
	if createParamsJsonErr != nil {
		return CreateResponse{}, eris.Errorf("error marshalling CreateParams: %v", createParamsJsonErr)
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
		return CreateResponse{}, eris.Errorf("error creating new Offer: %v", sendErr)
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
			return CreateResponse{}, eris.Errorf("error sending offer: %v", response.Error)
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
		return CreateResponse{}, eris.Errorf("error sending offer: %v", response.Error)
	}

	if response.TradeOfferId == 0 {
		return CreateResponse{}, eris.Errorf("error creating offer: steam returned tradeofferid 0")
	}

	return response, nil
}

type PartnerInventoryRequest struct {
	SessionId string
	AppId     uint64
	ContextId string

	PartnerSteamId steamid.SteamID
	PartnerToken   string
}

func (p PartnerInventoryRequest) Values() (url.Values, error) {
	return p.OldValues()
}

func (p PartnerInventoryRequest) CacheTTL() time.Duration {
	return 0
}

func (p PartnerInventoryRequest) Retryable() bool {
	return true
}

func (p PartnerInventoryRequest) RequiresApiKey() bool {
	return false
}

func (p PartnerInventoryRequest) Method() string {
	return http.MethodGet
}

func (p PartnerInventoryRequest) Url() string {
	return "https://steamcommunity.com/tradeoffer/new/partnerinventory/"
}

func (p PartnerInventoryRequest) OldValues() (url.Values, error) {
	values := make(url.Values)
	values.Add("sessionid", p.SessionId)
	values.Add("partner", p.PartnerSteamId.String())
	values.Add("appid", url.QueryEscape(strconv.FormatUint(p.AppId, 10)))
	values.Add("contextid", url.QueryEscape(p.ContextId))
	return values, nil
}

func (p PartnerInventoryRequest) Headers() (http.Header, error) {
	referer := fmt.Sprintf(
		"https://steamcommunity.com/tradeoffer/new/?partner=%d&token=%s",
		p.PartnerSteamId.AccountId(),
		url.QueryEscape(p.PartnerToken),
	)

	return http.Header{
		"Referer": []string{referer},
	}, nil
}

func (p PartnerInventoryRequest) EnsureResponseSuccess(httpResponse *http.Response) error {
	return steamlang.EnsureSuccessResponse(httpResponse)
}

type PartnerItem struct {
	Id          string `json:"id"`
	ClassId     string `json:"classid"`
	InstanceId  string `json:"instanceid"`
	Amount      string `json:"amount"`
	HideInChina bool   `json:"hide_in_china"`
	Position    int    `json:"pos"`
}

type PartnerDescription struct {
	AppId                       string                   `json:"appid"`
	ClassId                     string                   `json:"classid"`
	InstanceId                  string                   `json:"instanceid"`
	IconUrl                     string                   `json:"icon_url"`
	IconDragUrl                 string                   `json:"icon_drag_url"`
	Name                        string                   `json:"name"`
	MarketHashName              string                   `json:"market_hash_name"`
	MarketName                  string                   `json:"market_name"`
	NameColor                   string                   `json:"name_color"`
	BackgroundColor             string                   `json:"background_color"`
	Type                        string                   `json:"type"`
	Tradable                    int                      `json:"tradable"`
	Marketable                  int                      `json:"marketable"`
	Commodity                   int                      `json:"commodity"`
	MarketTradableRestriction   string                   `json:"market_tradable_restriction"`
	MarketMarketableRestriction string                   `json:"market_marketable_restriction"`
	DescriptionLines            []PartnerDescriptionLine `json:"descriptions"`
	Tags                        []PartnerDescriptionTag  `json:"tags"`
}

type PartnerDescriptionLine struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

type PartnerDescriptionTag struct {
	InternalName string `json:"internal_name"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	CategoryName string `json:"category_name"`
}

type PartnerInventoryResponse struct {
	Success      bool                          `json:"success"`
	Inventory    map[string]PartnerItem        `json:"rgInventory"`
	Descriptions map[string]PartnerDescription `json:"rgDescriptions"`
	More         bool                          `json:"more"`
	MoreStart    json.RawMessage               `json:"more_start"`
}

func (c *Client) GetPartnerInventory(
	ctx context.Context,
	partnerId steamid.SteamID,
	partnerToken string,
	appId uint64,
	contextId string,
) (*PartnerInventoryResponse, error) {
	sessionId, sessionIdErr := c.SessionIdFunc(c.Transport)
	if sessionIdErr != nil {
		return nil, eris.Errorf("error retrieving sessionId from transport: %v", sessionIdErr)
	}

	request := &PartnerInventoryRequest{
		SessionId:      sessionId,
		AppId:          appId,
		ContextId:      contextId,
		PartnerSteamId: partnerId,
		PartnerToken:   partnerToken,
	}
	var response PartnerInventoryResponse
	sendErr := c.Transport.Send(ctx, request, &response)
	if sendErr != nil {
		return nil, sendErr
	}

	return &response, nil
}
