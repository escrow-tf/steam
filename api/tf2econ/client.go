package tf2econ

import (
	"context"
	"fmt"
	"github.com/escrow-tf/steam/api"
	"github.com/escrow-tf/steam/steamid"
	"net/http"
	"net/url"
)

const WebApiBaseUrl = "https://api.steampowered.com"

type Client struct {
	Transport *api.HttpTransport
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

func (client *Client) GetPlayerItems(ctx context.Context, steamId steamid.SteamID) (*PlayerItemsResponse, error) {
	request := PlayerItemsRequest{
		steamId: steamId,
	}
	var response PlayerItemsResponse
	sendErr := client.Transport.Send(ctx, request, &response)
	if sendErr != nil {
		return nil, sendErr
	}
	return &response, nil
}
