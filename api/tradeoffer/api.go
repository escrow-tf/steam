package tradeoffer

import (
	"context"
	"github.com/escrow-tf/steam/steamid"
)

type Api interface {
	Accept(ctx context.Context, id uint64) (*ActionResponse, error)
	Decline(ctx context.Context, id uint64) (*ActionResponse, error)
	Cancel(ctx context.Context, id uint64) (*ActionResponse, error)
	Create(
		ctx context.Context,
		other steamid.SteamID,
		partnerToken string,
		myItems, theirItems []Item,
		message string,
	) (CreateResponse, error)
}
