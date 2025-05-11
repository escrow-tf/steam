package twofactor

import (
	"context"
	"time"
)

type Api interface {
	SteamTime() (time.Time, error)
	AlignTime(ctx context.Context) error
	QueryTime(ctx context.Context) (*QueryTimeResponse, error)
}
