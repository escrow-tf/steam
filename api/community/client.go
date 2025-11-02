package community

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/escrow-tf/steam/api"
	"github.com/escrow-tf/steam/steamid"
	"github.com/escrow-tf/steam/steamlang"
)

const BaseURL = "https://www.steamcommunity.com"

type Client struct {
	Transport api.Transport
}

type PlayerInventoryRequest struct {
	steamId   steamid.SteamID
	appId     string
	contextId string
	language  string
	count     uint
	start     uint
}

func (p PlayerInventoryRequest) CacheTTL() time.Duration {
	return 0
}

func (p PlayerInventoryRequest) EnsureResponseSuccess(httpResponse *http.Response) error {
	return steamlang.EnsureSuccessResponse(httpResponse)
}

func (p PlayerInventoryRequest) Headers() (http.Header, error) {
	return nil, nil
}

func (p PlayerInventoryRequest) Retryable() bool {
	return true
}

func (p PlayerInventoryRequest) RequiresApiKey() bool {
	return false
}

func (p PlayerInventoryRequest) Method() string {
	return http.MethodGet
}

func (p PlayerInventoryRequest) OldValues() (url.Values, error) {
	values := make(url.Values)
	values.Add("l", p.language)
	values.Add("count", strconv.FormatUint(uint64(p.count), 10))
	values.Add("start", strconv.FormatUint(uint64(p.start), 10))
	return values, nil
}

func (p PlayerInventoryRequest) Values() (interface{}, error) {
	values := make(url.Values)
	values.Add("l", p.language)
	values.Add("count", strconv.FormatUint(uint64(p.count), 10))
	values.Add("start", strconv.FormatUint(uint64(p.start), 10))
	return values, nil
}

func (p PlayerInventoryRequest) Url() string {
	steamIdEncoded := url.QueryEscape(p.steamId.String())
	appIdEncoded := url.QueryEscape(p.appId)
	contextIdEncoded := url.QueryEscape(p.contextId)
	return fmt.Sprintf("%s/inventory/%s/%s/%s", BaseURL, appIdEncoded, steamIdEncoded, contextIdEncoded)
}

type PlayerInventory struct {
	Assets              []Asset       `json:"assets"`
	Descriptions        []Description `json:"descriptions"`
	MoreItems           int           `json:"more_items,omitempty"`
	LastAssetId         int           `json:"last_assetid,omitempty"`
	TotalInventoryCount int           `json:"total_inventory_count"`
	Success             int           `json:"success"`
	Rwgrsn              int           `json:"rwgrsn"`
}

type Asset struct {
	AppId      uint   `json:"appid"`
	ContextId  string `json:"contextid"`
	AssetId    string `json:"assetid"`
	ClassId    string `json:"classid"`
	InstanceId string `json:"instanceid"`
	Amount     string `json:"amount"`
}

type Description struct {
	AppId                       uint     `json:"appid"`
	ClassId                     string   `json:"classid"`
	InstanceId                  string   `json:"instanceid"`
	Currency                    int      `json:"currency"`
	BackgroundColor             string   `json:"background_color"`
	IconUrl                     string   `json:"icon_url"`
	IconUrlLarge                string   `json:"icon_url_large"`
	Tradable                    int      `json:"tradable"`
	Name                        string   `json:"name"`
	NameColor                   string   `json:"name_color"`
	Type                        string   `json:"type"`
	MarketName                  string   `json:"market_name"`
	MarketHashName              string   `json:"market_hash_name"`
	Commodity                   int      `json:"commodity"`
	MarketTradableRestriction   string   `json:"market_tradable_restriction"`
	MarketMarketableRestriction string   `json:"market_marketable_restriction"`
	Marketable                  string   `json:"marketable"`
	FraudWarnings               []string `json:"fraudwarnings,omitempty"`
	Tags                        []Tag    `json:"tags"`
	Lines                       []Line   `json:"descriptions,omitempty"`
	Actions                     []Action `json:"actions,omitempty"`
	MarketActions               []Action `json:"market_actions,omitempty"`
}

type Tag struct {
	Category              string `json:"category"`
	InternalName          string `json:"internal_name"`
	LocalizedCategoryName string `json:"localized_category_name"`
	LocalizedTagName      string `json:"localized_tag_name"`
	Color                 string `json:"color,omitempty"`
}

type Line struct {
	Value string `json:"value"`
	Color string `json:"color,omitempty"`
	Type  string `json:"type,omitempty"`
	Name  string `json:"name"`
}

type Action struct {
	Link string `json:"link"`
	Name string `json:"name"`
}

func (c *Client) GetPlayerInventory(
	ctx context.Context,
	steamID steamid.SteamID,
	appID, contextID, language string,
	count uint,
	start uint,
) (*PlayerInventory, error) {
	request := PlayerInventoryRequest{
		steamId:   steamID,
		appId:     appID,
		contextId: contextID,
		language:  language,
		count:     count,
		start:     start,
	}
	response := &PlayerInventory{}
	sendErr := c.Transport.Send(ctx, request, response)
	if sendErr != nil {
		return nil, sendErr
	}
	return response, nil
}
