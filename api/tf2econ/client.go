package tf2econ

import (
	"fmt"
	"github.com/escrow-tf/steam/api"
	"github.com/escrow-tf/steam/steamid"
	"github.com/hashicorp/go-retryablehttp"
	"io"
	"log"
	"net/http"
	"net/url"
)

const WebApiBaseUrl = "https://api.steampowered.com"

type Client struct {
	Transport *api.Transport
}

type PlayerItemsRequest struct {
	steamId steamid.SteamID
}

func (p PlayerItemsRequest) Headers() (http.Header, error) {
	return nil, nil
}

func (p PlayerItemsRequest) Retryable() bool {
	return true
}

func (p PlayerItemsRequest) RequiresApiKey() bool {
	return true
}

func (p PlayerItemsRequest) Method() string {
	return http.MethodGet
}

func (p PlayerItemsRequest) Url() string {
	return fmt.Sprintf("%s/IEconItems_440/GetPlayerItems/v1/", WebApiBaseUrl)
}

func (p PlayerItemsRequest) Values() (url.Values, error) {
	return url.Values{
		"steamid": []string{p.steamId.String()},
	}, nil
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
	request := PlayerItemsRequest{
		steamId: steamId,
	}
	var response PlayerItemsResponse
	sendErr := client.Transport.Send(request, &response)
	if sendErr != nil {
		return nil, sendErr
	}
	return &response, nil
}
