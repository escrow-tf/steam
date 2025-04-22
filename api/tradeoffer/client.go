package tradeoffer

import (
	"encoding/json"
	"fmt"
	"github.com/escrow-tf/steam/api"
	"github.com/escrow-tf/steam/steamid"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Client struct {
	client *http.Client
}

func NewClient(transport *api.Transport) *Client {
	return &Client{
		client: transport.HttpClient(),
	}
}

type CreateParams struct {
	AccessToken string `json:"trade_offer_access_token"`
}

type OfferData struct {
	SessionId        string `json:"sessionid"`
	ServerId         string `json:"serverid"`
	Partner          string `json:"partner"`
	Message          string `json:"tradeoffermessage"`
	OfferJson        string `json:"json_tradeoffer"`
	CreateParamsJson string `json:"trade_offer_create_params"`
}

func (o OfferData) GetValues() (url.Values, error) {
	values := make(url.Values)
	values.Add("sessionid", o.SessionId)
	values.Add("serverid", o.ServerId)
	values.Add("partner", o.Partner)
	values.Add("tradeoffermessage", o.Message)
	values.Add("json_tradeoffer", o.OfferJson)
	values.Add("trade_offer_create_params", o.CreateParamsJson)
	return values, nil
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

func InitializeFormRequestHeaders(request *http.Request) {
	request.Header.Add("Accept", "application/json")
	request.Header.Add("User-Agent", "okhttp/3.12.12")
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
}

type ActionResponse struct {
	TradeOfferId uint64 `json:"tradeofferid,string"`
}

func (c Client) act(sessionId string, id uint64, verb string) (ActionResponse, error) {
	requestUrl := fmt.Sprintf("https://steamcommunity.com/tradeoffer/%d/%s", id, verb)

	encodedSessionId := url.QueryEscape(sessionId)
	body := fmt.Sprintf("sessionid=%s", encodedSessionId)
	httpRequest, err := http.NewRequest(http.MethodPost, requestUrl, strings.NewReader(body))
	if err != nil {
		return ActionResponse{}, fmt.Errorf("error creating request: %v", err)
	}

	InitializeFormRequestHeaders(httpRequest)

	// TODO: do we need referrer?

	httpResponse, err := c.client.Do(httpRequest)
	if err != nil {
		return ActionResponse{}, fmt.Errorf("error sending request: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("error closing body: %v", err)
		}
	}(httpResponse.Body)

	response := ActionResponse{}
	if err = json.NewDecoder(httpResponse.Body).Decode(&response); err != nil {
		return ActionResponse{}, fmt.Errorf("error decoding response: %v", err)
	}

	if response.TradeOfferId == 0 {
		return ActionResponse{}, fmt.Errorf("error creating offer: steam returned tradeofferid 0")
	}

	if httpResponse.StatusCode != http.StatusOK {
		return ActionResponse{}, fmt.Errorf("error sending request: %d %s", httpResponse.StatusCode, httpResponse.Status)
	}

	return response, nil
}

func (c Client) Accept(sessionId string, id uint64) (ActionResponse, error) {
	return c.act(sessionId, id, "accept")
}

func (c Client) Decline(sessionId string, id uint64) (ActionResponse, error) {
	return c.act(sessionId, id, "decline")
}

func (c Client) Cancel(sessionId string, id uint64) (ActionResponse, error) {
	return c.act(sessionId, id, "cancel")
	//requestUrl := fmt.Sprintf("https://steamcommunity.com/tradeoffer/%d/cancel", id)
	//
	//encodedSessionId := url.QueryEscape(sessionId)
	//body := fmt.Sprintf("sessionid=%s", encodedSessionId)
	//httpRequest, err := http.NewRequest(http.MethodPost, requestUrl, strings.NewReader(body))
	//if err != nil {
	//	return ActionResponse{}, fmt.Errorf("error creating request: %v", err)
	//}
	//
	//InitializeFormRequestHeaders(httpRequest)
	//
	//// TODO: do we need referrer?
	//
	//httpResponse, err := c.client.Do(httpRequest)
	//if err != nil {
	//	return ActionResponse{}, fmt.Errorf("error sending request: %v", err)
	//}
	//defer func(Body io.ReadCloser) {
	//	err := Body.Close()
	//	if err != nil {
	//		log.Printf("error closing body: %v", err)
	//	}
	//}(httpResponse.Body)
	//
	//response := ActionResponse{}
	//if err = json.NewDecoder(httpResponse.Body).Decode(&response); err != nil {
	//	return ActionResponse{}, fmt.Errorf("error decoding response: %v", err)
	//}
	//
	//if response.TradeOfferId == 0 {
	//	return ActionResponse{}, fmt.Errorf("error creating offer: steam returned tradeofferid 0")
	//}
	//
	//if httpResponse.StatusCode != http.StatusOK {
	//	return ActionResponse{}, fmt.Errorf("error sending request: %d %s", httpResponse.StatusCode, httpResponse.Status)
	//}
	//
	//return response, nil
}

type CreateResponse struct {
	Error        string `json:"strError"`
	TradeOfferId uint64 `json:"tradeOfferId,string"`
}

func (c Client) Create(sessionId string, other steamid.SteamID, partnerToken string, myItems, theirItems []Item, message string) (CreateResponse, error) {
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

	offerJson, err := json.Marshal(offer)
	if err != nil {
		return CreateResponse{}, fmt.Errorf("error marshalling Offer: %v", err)
	}

	createParams := CreateParams{
		AccessToken: partnerToken,
	}

	createParamsJson, err := json.Marshal(createParams)
	if err != nil {
		return CreateResponse{}, fmt.Errorf("error marshalling CreateParams: %v", err)
	}

	offerData := OfferData{
		SessionId:        sessionId,
		ServerId:         "1",
		Partner:          other.String(),
		Message:          message,
		OfferJson:        string(offerJson),
		CreateParamsJson: string(createParamsJson),
		// CreateParamsJson: "{}",
	}

	values, err := offerData.GetValues()
	if err != nil {
		return CreateResponse{}, fmt.Errorf("couldn't get values from StartSessionRequest: %v", err)
	}

	body := values.Encode()
	httpRequest, err := http.NewRequest(http.MethodPost, "https://steamcommunity.com/tradeoffer/new/send", strings.NewReader(body))
	if err != nil {
		return CreateResponse{}, fmt.Errorf("error creating request: %v", err)
	}

	InitializeFormRequestHeaders(httpRequest)

	encodedPartnerAccountId := strconv.Itoa(other.AccountId())
	encodedPartnerToken := url.QueryEscape(partnerToken)
	referer := fmt.Sprintf("https://steamcommunity.com/tradeoffer/new/?partner=%s&token=%s", encodedPartnerAccountId, encodedPartnerToken)
	httpRequest.Header.Add("Referer", referer)

	httpResponse, err := c.client.Do(httpRequest)
	if err != nil {
		return CreateResponse{}, fmt.Errorf("error sending request: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("error closing body: %v", err)
		}
	}(httpResponse.Body)

	response := CreateResponse{}
	if err = json.NewDecoder(httpResponse.Body).Decode(&response); err != nil {
		return CreateResponse{}, fmt.Errorf("error decoding response: %v", err)
	}

	// Error code descriptions:
	// 15	invalid trade access token
	// 16	timeout
	// 20	wrong contextid
	// 25	can't send more offers until some is accepted/cancelled...
	// 26	object is not in our inventory
	// error code names are in steamlang/enums.go EResult_name
	if response.Error != "" {
		return CreateResponse{}, fmt.Errorf("error sending offer: %v", response.Error)
	}

	if response.TradeOfferId == 0 {
		return CreateResponse{}, fmt.Errorf("error creating offer: steam returned tradeofferid 0")
	}

	if httpResponse.StatusCode != http.StatusOK {
		return CreateResponse{}, fmt.Errorf("error sending request: %d %s", httpResponse.StatusCode, httpResponse.Status)
	}

	return response, nil
}
