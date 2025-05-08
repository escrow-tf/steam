package econ

import (
	"fmt"
	"github.com/escrow-tf/steam/api"
	"github.com/escrow-tf/steam/api/community"
	"net/http"
	"net/url"
	"strconv"
)

type OfferState uint

const (
	InvalidOfferState                  OfferState = 1  // Invalid
	ActiveOfferState                              = 2  // This trade offer has been sent, neither party has acted on it yet.
	AcceptedOfferState                            = 3  // The trade offer was accepted by the recipient and items were exchanged.
	CounteredOfferState                           = 4  // The recipient made a counter-offer
	ExpiredOfferState                             = 5  // The trade offer was not accepted before the expiration date
	CanceledOfferState                            = 6  // The sender cancelled the offer
	DeclinedOfferState                            = 7  // The recipient declined the offer
	InvalidItemsOfferState                        = 8  // Some of the items in the offer are no longer available (indicated by the missing flag in the output)
	CreatedNeedsConfirmationOfferState            = 9  // The offer hasn't been sent yet and is awaiting email/mobile confirmation. The offer is only visible to the sender.
	CanceledBySecondFactorOfferState              = 10 // Either party canceled the offer via email/mobile. The offer is visible to both parties, even if the sender canceled it before it was sent.
	InEscrowOfferState                            = 11 // The trade has been placed on hold. The items involved in the trade have all been removed from both parties' inventories and will be automatically delivered in the future.
)

type OfferConfirmationMethod uint

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
	Transport *api.Transport
}

type GetTradeOfferRequest struct {
	id       uint64
	language string
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

func (g GetTradeOfferRequest) Values() (url.Values, error) {
	values := make(url.Values)
	values.Add("tradeofferid", strconv.FormatUint(g.id, 10))
	values.Add("language", g.language)
	return values, nil
}

type GetTradeOfferResponse struct {
	Offer        *TradeOffer              `json:"offer"`
	Descriptions []*community.Description `json:"descriptions"`
}

func (c *Client) GetTradeOffer(id uint64) (*GetTradeOfferResponse, error) {
	request := GetTradeOfferRequest{
		id:       id,
		language: "en_us",
	}
	var response GetTradeOfferResponse
	sendErr := c.Transport.Send(request, &response)
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

func (g GetTradeOffersRequest) Values() (url.Values, error) {
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

func (c *Client) GetTradeOffers(getSent, getReceived, getDescriptions, activeOnly, historicalOnly bool, historicalCutoff uint32) (*GetTradeOffersResponse, error) {
	request := GetTradeOffersRequest{
		getSent:          getSent,
		getReceived:      getReceived,
		getDescriptions:  getDescriptions,
		activeOnly:       activeOnly,
		historicalOnly:   historicalOnly,
		historicalCutoff: historicalCutoff,
	}
	var response GetTradeOffersResponse
	sendErr := c.Transport.Send(request, &response)
	if sendErr != nil {
		return nil, sendErr
	}

	return &response, nil
}
