package tf2econ

import (
	"context"

	"github.com/escrow-tf/steam/steamid"
)

type Api interface {
	GetPlayerItems(ctx context.Context, steamId steamid.SteamID) (*PlayerItemsResponse, error)
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
