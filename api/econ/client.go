package econ

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/escrow-tf/steam/api"
	"github.com/escrow-tf/steam/api/community"
	"github.com/escrow-tf/steam/steamlang"
)

type OfferState uint

//goland:noinspection GoUnusedConst
const (
	// InvalidOfferState - Invalid
	InvalidOfferState OfferState = 1
	// ActiveOfferState - This trade offer has been sent, neither party has acted on it yet.
	ActiveOfferState = 2
	// AcceptedOfferState - The trade offer was accepted by the recipient and items were exchanged.
	AcceptedOfferState = 3
	// CounteredOfferState - The recipient made a counter-offer
	CounteredOfferState = 4
	// ExpiredOfferState - The trade offer was not accepted before the expiration date
	ExpiredOfferState = 5
	// CanceledOfferState - The sender cancelled the offer
	CanceledOfferState = 6
	// DeclinedOfferState - The recipient declined the offer
	DeclinedOfferState = 7
	// InvalidItemsOfferState - Some of the items in the offer are no longer available (indicated by the
	// missing flag in the output)
	InvalidItemsOfferState = 8
	// CreatedNeedsConfirmationOfferState - The offer hasn't been sent yet and is awaiting email/mobile
	// confirmation. The offer is only visible to the sender.
	CreatedNeedsConfirmationOfferState = 9
	// CanceledBySecondFactorOfferState - Either party canceled the offer via email/mobile. The offer is
	// visible to both parties, even if the sender canceled it before it was sent.
	CanceledBySecondFactorOfferState = 10
	// InEscrowOfferState - The trade has been placed on hold. The items involved in the trade have all
	// been removed from both parties' inventories and will be automatically delivered in the future.
	InEscrowOfferState = 11
)

type OfferConfirmationMethod uint

//goland:noinspection GoUnusedConst
const (
	InvalidOfferConfirmationMethod   OfferConfirmationMethod = 0
	EmailOfferConfirmationMethod                             = 1
	MobileAppOfferConfirmationMethod                         = 2
)

type TradeOffer struct {
	TradeOfferId       uint64                  `json:",string"`
	TradeId            uint64                  `json:",string"`
	OtherAccountId     uint32                  `json:"accountid_other"`
	OtherSteamId       string                  `json:"other_steam_id"`
	Message            string                  `json:"message"`
	ExpirationTime     uint32                  `json:"expiraton_time"`
	State              OfferState              `json:"trade_offer_state"`
	ToGive             []*community.Asset      `json:"items_to_give"`
	ToReceive          []*community.Asset      `json:"items_to_receive"`
	IsOurOffer         bool                    `json:"is_our_offer"`
	TimeCreated        uint32                  `json:"time_created"`
	TimeUpdated        uint32                  `json:"time_updated"`
	EscrowEndDate      uint32                  `json:"escrow_end_date"`
	ConfirmationMethod OfferConfirmationMethod `json:"confirmation_method"`
}

type Client struct {
	Transport api.Transport
}

type GetTradeOfferRequest struct {
	id       uint64
	language string
}

func (g GetTradeOfferRequest) CacheTTL() time.Duration {
	return 0
}

func (g GetTradeOfferRequest) EnsureResponseSuccess(httpResponse *http.Response) error {
	return steamlang.EnsureSuccessResponse(httpResponse)
}

func (g GetTradeOfferRequest) Headers() (http.Header, error) {
	return nil, nil
}

func (g GetTradeOfferRequest) Retryable() bool {
	return true
}

func (g GetTradeOfferRequest) RequiresApiKey() bool {
	return true
}

func (g GetTradeOfferRequest) Method() string {
	return http.MethodGet
}

func (g GetTradeOfferRequest) Url() string {
	return fmt.Sprintf("%s/IEconService/GetTradeOffer/v1/", api.BaseURL)
}

func (g GetTradeOfferRequest) OldValues() (url.Values, error) {
	values := make(url.Values)
	values.Add("tradeofferid", strconv.FormatUint(g.id, 10))
	values.Add("language", g.language)
	return values, nil
}

func (g GetTradeOfferRequest) Values() (interface{}, error) {
	values := make(url.Values)
	values.Add("tradeofferid", strconv.FormatUint(g.id, 10))
	values.Add("language", g.language)
	return values, nil
}

type GetTradeOfferResponse struct {
	Offer        *TradeOffer              `json:"offer"`
	Descriptions []*community.Description `json:"descriptions"`
}

func (c *Client) GetTradeOffer(ctx context.Context, id uint64) (*GetTradeOfferResponse, error) {
	request := GetTradeOfferRequest{
		id:       id,
		language: "en_us",
	}
	var response GetTradeOfferResponse
	sendErr := c.Transport.Send(ctx, request, &response)
	if sendErr != nil {
		return nil, sendErr
	}

	return &response, nil
}

type GetTradeOffersRequest struct {
	getSent          bool
	getReceived      bool
	getDescriptions  bool
	activeOnly       bool
	historicalOnly   bool
	historicalCutoff uint32
	language         string
}

func (g GetTradeOffersRequest) Values() (interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (g GetTradeOffersRequest) CacheTTL() time.Duration {
	return 0
}

func (g GetTradeOffersRequest) EnsureResponseSuccess(httpResponse *http.Response) error {
	return steamlang.EnsureSuccessResponse(httpResponse)
}

func (g GetTradeOffersRequest) Headers() (http.Header, error) {
	return nil, nil
}

func (g GetTradeOffersRequest) Retryable() bool {
	return true
}

func (g GetTradeOffersRequest) RequiresApiKey() bool {
	return true
}

func (g GetTradeOffersRequest) Method() string {
	return http.MethodGet
}

func (g GetTradeOffersRequest) Url() string {
	return fmt.Sprintf("%s/IEconService/GetTradeOffers/v1/", api.BaseURL)
}

func (g GetTradeOffersRequest) OldValues() (url.Values, error) {
	values := make(url.Values)
	values.Add("language", g.language)
	if g.getSent {
		values.Add("get_sent_offers", "1")
	}
	if g.getReceived {
		values.Add("get_received_offers", "1")
	}
	if g.getDescriptions {
		values.Add("get_descriptions", "1")
	}
	if g.activeOnly {
		values.Add("active_only", "1")
	}
	if g.historicalOnly {
		values.Add("historical_only", "1")
	}
	if g.historicalCutoff != 0 {
		values.Add("time_historical_cutoff", strconv.FormatUint(uint64(g.historicalCutoff), 10))
	}
	return values, nil
}

type GetTradeOffersResponse struct {
	Sent         []*TradeOffer            `json:"sent"`
	Received     []*TradeOffer            `json:"received"`
	Descriptions []*community.Description `json:"descriptions"`
}

func (c *Client) GetTradeOffers(
	ctx context.Context,
	getSent, getReceived, getDescriptions, activeOnly, historicalOnly bool,
	historicalCutoff uint32,
) (*GetTradeOffersResponse, error) {
	request := GetTradeOffersRequest{
		getSent:          getSent,
		getReceived:      getReceived,
		getDescriptions:  getDescriptions,
		activeOnly:       activeOnly,
		historicalOnly:   historicalOnly,
		historicalCutoff: historicalCutoff,
	}
	var response GetTradeOffersResponse
	sendErr := c.Transport.Send(ctx, request, &response)
	if sendErr != nil {
		return nil, sendErr
	}

	return &response, nil
}
