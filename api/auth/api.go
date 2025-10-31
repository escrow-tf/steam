package auth

import (
	"context"

	"github.com/escrow-tf/steam/steamid"
)

type Api interface {
	GetPublicRsaKey(ctx context.Context, accountName string) (PublicRsaKey, error)
	EncryptAccountPassword(ctx context.Context, accountName string, password string) (EncryptedPassword, error)
	StartSessionWithCredentials(
		ctx context.Context,
		accountName string,
		password EncryptedPassword,
		deviceDetails DeviceDetails,
	) (StartSessionResponse, error)
	SubmitSteamGuardCode(ctx context.Context, clientID string, steamID steamid.SteamID, code string) error
	PollSessionStatus(ctx context.Context, clientID string, requestID string) (PollSessionStatusResponse, error)
	GenerateAccessTokenForApp(ctx context.Context, refreshToken string, renew bool) (GenerateAccessTokenResponse, error)
}
