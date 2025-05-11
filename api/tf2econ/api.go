package tf2econ

import (
	"context"
	"github.com/escrow-tf/steam/steamid"
)

type Api interface {
	GetPlayerItems(ctx context.Context, steamId steamid.SteamID) (*PlayerItemsResponse, error)
}
