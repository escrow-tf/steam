package econ

import "context"

type Api interface {
	GetTradeOffer(ctx context.Context, id uint64) (*GetTradeOfferResponse, error)
	GetTradeOffers(
		ctx context.Context,
		getSent, getReceived, getDescriptions, activeOnly, historicalOnly bool,
		historicalCutoff uint32,
	) (*GetTradeOffersResponse, error)
}
