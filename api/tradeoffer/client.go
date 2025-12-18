package tradeoffer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/escrow-tf/steam/api"
	"github.com/escrow-tf/steam/api/transport"
	"github.com/escrow-tf/steam/steamid"
	"github.com/escrow-tf/steam/steamlang"
	"github.com/google/go-querystring/query"
	"github.com/rotisserie/eris"
)

var CommunityBase = url.URL{
	Scheme: "https",
	Host:   "steamcommunity.com",
}

type SessionIdFunc func(transport api.Transport) (string, error)

type AcceptBody struct {
	SessionID string `url:"sessionid"`
}

type AcceptResponse struct {
	TradeOfferId uint64 `json:"tradeofferid,string"`
}

func Accept(id uint64, sessionId string) (transport.PrivateTransportRequest[*AcceptBody, *AcceptResponse], error) {
	request := transport.PrivateTransportRequest[*AcceptBody, *AcceptResponse]{
		BaseUrl: CommunityBase,
		Path:    fmt.Sprintf("/tradeoffer/%d/accept", id),
		Method:  http.MethodPost,
		Body: &AcceptBody{
			SessionID: sessionId,
		},
		Transformer: transport.UrlEncodeTransformer[*AcceptBody](),
		Decoder:     transport.JsonDecoder[*AcceptResponse](),
	}

	return request, nil
}

type DeclineBody struct {
	SessionID string `url:"sessionid"`
}

type DeclineResponse struct {
	TradeOfferId uint64 `json:"tradeofferid,string"`
}

func Decline(id uint64, sessionId string) (transport.PrivateTransportRequest[*DeclineBody, *DeclineResponse], error) {
	request := transport.PrivateTransportRequest[*DeclineBody, *DeclineResponse]{
		BaseUrl: CommunityBase,
		Path:    fmt.Sprintf("/tradeoffer/%d/decline", id),
		Method:  http.MethodPost,
		Body: &DeclineBody{
			SessionID: sessionId,
		},
		Transformer: transport.UrlEncodeTransformer[*DeclineBody](),
		Decoder:     transport.JsonDecoder[*DeclineResponse](),
	}

	return request, nil
}

type CancelBody struct {
	SessionID string `url:"sessionid"`
}

type CancelResponse struct {
	TradeOfferId uint64 `json:"tradeofferid,string"`
}

func Cancel(id uint64, sessionId string) (transport.PrivateTransportRequest[*CancelBody, *CancelResponse], error) {
	request := transport.PrivateTransportRequest[*CancelBody, *CancelResponse]{
		BaseUrl: CommunityBase,
		Path:    fmt.Sprintf("/tradeoffer/%d/cancel", id),
		Method:  http.MethodPost,
		Body: &CancelBody{
			SessionID: sessionId,
		},
		Transformer: transport.UrlEncodeTransformer[*CancelBody](),
		Decoder:     transport.JsonDecoder[*CancelResponse](),
	}

	return request, nil
}

type CreateOfferRequest struct {
	SessionId        string `url:"sessionid"`
	ServerId         string `url:"serverid"`
	Partner          string `url:"partner"`
	Message          string `url:"tradeoffermessage"`
	OfferJson        string `url:"json_tradeoffer"`
	CreateParamsJson string `url:"trade_offer_create_params"`
}

type CreateOfferResponse struct {
	Error        string `json:"strError"`
	TradeOfferId uint64 `json:"tradeOfferId,string"`
}

type VerifiedCreateResponse CreateOfferResponse

func (c CreateOfferResponse) Verify() (*VerifiedCreateResponse, error) {
	// There are a couple of error formats we're likely to receive back:
	// A generic error message with an error number at the end:
	//  {"strError":"There was an error sending your trade offer.  Please try again later. (ERROR NUMBER)"}
	//
	// A specific error message:
	//  {"strError":"You have sent too many trade offers, or have too many outstanding trade offers with
	//  <bot display name>. Please cancel some before sending more."}
	//
	// In both of these cases, steam returns a 500 error code despite these clearly being 4xx errors, and doesn't
	// give us an EResult header in the response.

	if strings.HasPrefix(c.Error, "There was an error sending your trade offer.  Please try again later. (") {
		leftParenIdx := strings.Index(c.Error, "(")
		rightParenIdx := strings.Index(c.Error, ")")
		eResultString := c.Error[leftParenIdx:rightParenIdx]
		eResult, err := strconv.ParseInt(eResultString, 10, 32)
		if err != nil {
			return nil, eris.Errorf("error sending offer: %v", c.Error)
		}

		switch steamlang.EResult(eResult) {
		case steamlang.InvalidStateResult:
			return nil, InvalidStateError
		case steamlang.AccessDeniedResult:
			return nil, AccessDeniedError
		case steamlang.TimeoutResult:
			return nil, TimeoutError
		case steamlang.ServiceUnavailableResult:
			return nil, ServiceUnavailableError
		case steamlang.LimitExceededResult:
			return nil, TooManyTradeOffersError
		case steamlang.RevokedResult:
			return nil, ItemsDontExistError
		case steamlang.AlreadyRedeemedResult:
			return nil, ChangedPersonaNameRecentlyError
		}

		return nil, steamlang.EResultError(steamlang.EResult(eResult))
	}

	if strings.HasPrefix(
		c.Error,
		"You have sent too many trade offers, or have too many outstanding trade offers with",
	) {
		return nil, TooManyTradeOffersError
	}

	if c.Error != "" {
		return nil, eris.Errorf("error sending offer: %v", c.Error)
	}

	if c.TradeOfferId == 0 {
		return nil, eris.Errorf("error creating offer: steam returned tradeofferid 0")
	}

	verified := VerifiedCreateResponse(c)
	return &verified, nil
}

type createParams struct {
	AccessToken string `json:"trade_offer_access_token"`
}

type offer struct {
	NewVersion bool  `json:"newversion"`
	Version    int   `json:"version"`
	Me         party `json:"me"`
	Them       party `json:"them"`
}

type party struct {
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

func Create(
	other steamid.SteamID,
	partnerToken string,
	myItems, theirItems []Item,
	message string,
	sessionId string,
) (*transport.PrivateTransportRequest[*CreateOfferRequest, *CreateOfferResponse], error) {
	offerJson, offerJsonErr := json.Marshal(offer{
		NewVersion: true,
		Version:    3,
		Me: party{
			Assets:   myItems,
			Currency: []struct{}{},
			Ready:    false,
		},
		Them: party{
			Assets:   theirItems,
			Currency: []struct{}{},
			Ready:    false,
		},
	})
	if offerJsonErr != nil {
		return nil, eris.Wrap(offerJsonErr, "error marshalling offer")
	}

	createParamsJson, createParamsJsonErr := json.Marshal(createParams{
		AccessToken: partnerToken,
	})
	if createParamsJsonErr != nil {
		return nil, eris.Wrap(createParamsJsonErr, "error marshalling CreateParams")
	}

	encodedPartnerAccountId := strconv.FormatUint(other.ID(), 10)
	encodedPartnerToken := url.QueryEscape(partnerToken)
	referer := fmt.Sprintf(
		"https://steamcommunity.com/tradeoffer/new/?partner=%s&token=%s",
		encodedPartnerAccountId,
		encodedPartnerToken,
	)

	transportRequest := &transport.PrivateTransportRequest[*CreateOfferRequest, *CreateOfferResponse]{
		CanRetry: false,
		BaseUrl:  CommunityBase,
		Path:     "/tradeoffer/new/send",
		Headers: http.Header{
			"Referer": []string{referer},
		},
		Method: http.MethodPost,
		Body: &CreateOfferRequest{
			SessionId:        sessionId,
			ServerId:         "1",
			Partner:          other.String(),
			Message:          message,
			OfferJson:        string(offerJson),
			CreateParamsJson: string(createParamsJson),
		},
		Transformer: transport.UrlEncodeTransformer[*CreateOfferRequest](),
		Decoder:     transport.JsonDecoder[*CreateOfferResponse](),
	}

	return transportRequest, nil
}

type PartnerInventoryRequest struct {
	SessionId string `url:"sessionid"`
	AppId     uint64 `url:"appid"`
	ContextId string `url:"contextid"`
	Partner   string `url:"partner"`
}

func PartnerInventory(
	partnerId steamid.SteamID,
	partnerToken string,
	appId uint64,
	contextId string,
	sessionId string,
) (*transport.PrivateTransportRequest[*PartnerInventoryRequest, *PartnerInventoryResponse], error) {
	referer := fmt.Sprintf(
		"https://steamcommunity.com/tradeoffer/new/?partner=%d&token=%s",
		partnerId.AccountId(),
		url.QueryEscape(partnerToken),
	)

	params, err := query.Values(PartnerInventoryRequest{
		SessionId: sessionId,
		AppId:     appId,
		ContextId: contextId,
		Partner:   partnerId.String(),
	})

	if err != nil {
		return nil, eris.Wrap(err, "error creating params for request")
	}

	request := &transport.PrivateTransportRequest[*PartnerInventoryRequest, *PartnerInventoryResponse]{
		CanRetry:    true,
		BaseUrl:     CommunityBase,
		Path:        "/tradeoffer/new/partnerinventory/",
		Headers:     http.Header{"Referer": []string{referer}},
		Params:      params,
		Method:      http.MethodGet,
		Body:        nil,
		Transformer: transport.EmptyTransformer[*PartnerInventoryRequest](),
		Decoder:     partnerInventoryResponseDecoder{},
	}

	return request, nil
}

type partnerInventoryResponseDecoder struct{}

func (p partnerInventoryResponseDecoder) Decode(response io.ReadCloser, result *PartnerInventoryResponse) error {
	body, err := io.ReadAll(response)
	if err != nil {
		return eris.Wrap(err, "error reading response body")
	}

	if err = json.Unmarshal(body, result); err != nil {
		return eris.Wrap(err, "error unmarshalling json response")
	}

	for key, description := range result.Descriptions {
		if err = json.Unmarshal(description.JsonDescriptionLines, &description.DescriptionLines); err != nil {
			return eris.Wrapf(err, "error unmarshalling json description line, key: %d", key)
		}
		if err = json.Unmarshal(description.JsonTags, &description.Tags); err != nil {
			return eris.Wrapf(err, "error unmarshalling json tag, key: %d", key)
		}
	}

	return nil
}

type PartnerItem struct {
	Id          string `json:"id"`
	ClassId     string `json:"classid"`
	InstanceId  string `json:"instanceid"`
	Amount      string `json:"amount"`
	HideInChina int    `json:"hide_in_china"`
	Position    int    `json:"pos"`
}

type PartnerDescription struct {
	AppId                       string          `json:"appid"`
	ClassId                     string          `json:"classid"`
	InstanceId                  string          `json:"instanceid"`
	IconUrl                     string          `json:"icon_url"`
	IconDragUrl                 string          `json:"icon_drag_url"`
	Name                        string          `json:"name"`
	MarketHashName              string          `json:"market_hash_name"`
	MarketName                  string          `json:"market_name"`
	NameColor                   string          `json:"name_color"`
	BackgroundColor             string          `json:"background_color"`
	Type                        string          `json:"type"`
	Tradable                    int             `json:"tradable"`
	Marketable                  int             `json:"marketable"`
	Commodity                   int             `json:"commodity"`
	MarketTradableRestriction   string          `json:"market_tradable_restriction"`
	MarketMarketableRestriction string          `json:"market_marketable_restriction"`
	JsonDescriptionLines        json.RawMessage `json:"descriptions"`
	JsonTags                    json.RawMessage `json:"tags"`

	DescriptionLines []PartnerDescriptionLine `json:"-"`
	Tags             []PartnerDescriptionTag  `json:"-"`
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
