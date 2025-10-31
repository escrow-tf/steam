package community

import (
	"context"

	"github.com/escrow-tf/steam/steamid"
)

type Api interface {
	GetPlayerInventory(
		ctx context.Context,
		steamID steamid.SteamID,
		appID, contextID, language string,
		count uint,
		start uint,
	) (*PlayerInventory, error)
}
