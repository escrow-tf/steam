package tf2econ

import (
	"encoding/json"
	"fmt"
	"github.com/escrow-tf/steam/steamid"
	"github.com/escrow-tf/steam/steamlang"
	"github.com/hashicorp/go-retryablehttp"
	"io"
	"log"
	"net/http"
	"net/url"
)

const WebApiBaseUrl = "https://api.steampowered.com"

type Client struct {
	webApiKey string
	client    *retryablehttp.Client
}

func NewClient(webApiKey string) *Client {
	return &Client{
		webApiKey: webApiKey,
		client:    retryablehttp.NewClient(),
	}
}

type PlayerItemsResponse struct {
	Result struct {
		Status           int    `json:"status"`
		StatusDetail     string `json:"status_detail,omitempty"`
		NumBackpackSlots int    `json:"num_backpack_slots,omitempty"`
		Items            []Item `json:"items,omitempty"`
	} `json:"result"`
}

type Item struct {
	Id                int         `json:"id"`
	OriginalId        int         `json:"original_id"`
	DefIndex          int         `json:"defindex"`
	Level             int         `json:"level"`
	Quality           int         `json:"quality"`
	Inventory         int64       `json:"inventory"`
	Quantity          int         `json:"quantity"`
	Origin            int         `json:"origin"`
	CannotTrade       bool        `json:"cannot_trade,omitempty"`
	Style             int         `json:"style,omitempty"`
	CannotCraft       bool        `json:"cannot_craft,omitempty"`
	CustomName        *string     `json:"custom_name,omitempty"`
	CustomDescription *string     `json:"custom_desc,omitempty"`
	Attributes        []Attribute `json:"attributes,omitempty"`
	Equipped          []EquipInfo `json:"equipped,omitempty"`
}

type Attribute struct {
	DefIndex int `json:"defindex"`
	// will be float64 or string
	Value      interface{} `json:"value"`
	FloatValue *float64    `json:"float_value,omitempty"`
}

type EquipInfo struct {
	Class int `json:"class"`
	Slot  int `json:"slot"`
}

func closeBody(body io.ReadCloser) {
	err := body.Close()
	if err != nil {
		log.Printf("error closing http response body: %s", err)
	}
}

func InitializeRequestHeaders(request *retryablehttp.Request) {
	request.Header.Add("Accept", "application/json")
	request.Header.Add("User-Agent", "okhttp/3.12.12")
}

func (client *Client) GetPlayerItems(steamId steamid.SteamID) (*PlayerItemsResponse, error) {
	encodedSteamId := url.QueryEscape(steamId.String())
	requestUrl := fmt.Sprintf("%s/IEconItems_440/GetPlayerItems/v1/?key=%s&steamid=%s", WebApiBaseUrl, client.webApiKey, encodedSteamId)
	request, err := retryablehttp.NewRequest(http.MethodGet, requestUrl, nil)
	if err != nil {
		return nil, err
	}

	InitializeRequestHeaders(request)

	httpResponse, err := client.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error on GET IEconItems_440/GetPlayerItems/v1/: %v", err)
	}

	if err = steamlang.EnsureSuccessResponse(httpResponse); err != nil {
		return nil, fmt.Errorf("non-success status code on GET IEconItems_440/GetPlayerItems/v1/: %v", err)
	}

	defer closeBody(httpResponse.Body)

	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body in response from IEconItems_440/GetPlayerItems/v1/: %v", err)
	}

	playerItems := &PlayerItemsResponse{}
	err = json.Unmarshal(responseBody, playerItems)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling PlayerItemsResponse: %v", err)
	}

	return playerItems, nil
}
