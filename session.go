package steam

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/escrow-tf/steam/api"
	"github.com/escrow-tf/steam/api/auth"
	"github.com/escrow-tf/steam/api/community"
	"github.com/escrow-tf/steam/api/econ"
	"github.com/escrow-tf/steam/api/mobileconf"
	"github.com/escrow-tf/steam/api/tf2econ"
	"github.com/escrow-tf/steam/api/tradeoffer"
	"github.com/escrow-tf/steam/api/twofactor"
	"github.com/escrow-tf/steam/steamid"
	"github.com/escrow-tf/steam/totp"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rotisserie/eris"
)

type AccountState struct {
	accountName string
	password    string
	totpState   *totp.State
}

func NewAccountState(
	accountName string,
	password string,
	sharedSecret string,
	identitySecret string,
) (*AccountState, error) {
	state, err := totp.NewState(sharedSecret, identitySecret)
	if err != nil {
		return nil, eris.Errorf("NewAccountState failed %v", err)
	}

	return &AccountState{
		accountName: accountName,
		password:    password,
		totpState:   state,
	}, nil
}

func (accountState *AccountState) TotpState() *totp.State {
	return accountState.totpState
}

type WebSession struct {
	state            *AccountState
	transport        *api.HttpTransport
	authClient       *auth.Client
	communityClient  *community.Client
	econClient       *econ.Client
	mobileConfClient *mobileconf.Client
	tf2EconClient    *tf2econ.Client
	tradeOfferClient *tradeoffer.Client
	twoFactorClient  *twofactor.Client

	clientId        string
	requestId       string
	steamId         steamid.SteamID
	jwt             *jwt.Token
	refreshToken    string
	accessToken     string
	refreshInterval int
}

func (w *WebSession) Transport() api.Transport {
	return w.transport
}

type Options struct {
	AccountState *AccountState
	api.HttpTransportOptions
}

func Authenticate(ctx context.Context, options Options) (*WebSession, error) {
	if options.AccountState == nil {
		return nil, errors.New("AccountState is required")
	}

	deviceHostName, err := os.Hostname()
	if err != nil {
		return nil, eris.Errorf("os.Hostname() failed: %v", err)
	}

	webTransport := api.NewTransport(options.HttpTransportOptions)
	authClient := &auth.Client{
		Transport: webTransport,
	}
	twoFactorClient := &twofactor.Client{
		Transport: webTransport,
	}

	alignErr := twoFactorClient.AlignTime(ctx)
	if alignErr != nil {
		return nil, eris.Errorf("twoFactorClient.AlignTime() failed: %v", alignErr)
	}

	encryptedPassword, err := authClient.EncryptAccountPassword(
		ctx,
		options.AccountState.accountName,
		options.AccountState.password,
	)
	if err != nil {
		return nil, eris.Errorf("EncryptPassword failed %v", err)
	}

	deviceDetails := auth.DeviceDetails{
		FriendlyName:     fmt.Sprintf("%s (steamguard-cli)", deviceHostName),
		PlatformType:     auth.MobileAppPlatformType,
		OsType:           auth.AndroidUnknownOsType,
		GamingDeviceType: auth.DefaultGamingDeviceType,
	}

	sessionResponse, err := authClient.StartSessionWithCredentials(
		ctx,
		options.AccountState.accountName,
		encryptedPassword,
		deviceDetails,
	)
	if err != nil {
		return nil, eris.Errorf("StartSessionWithCredentials failed %v", err)
	}

	hasDeviceCodeType := false
	for _, allowedConfirmation := range sessionResponse.Response.AllowedConfirmations {
		if allowedConfirmation.ConfirmationType == auth.DeviceCodeGuardType {
			hasDeviceCodeType = true
		}
	}

	if !hasDeviceCodeType {
		return nil, eris.Errorf("DeviceCode auth not in list of allowed confirmations")
	}

	weakToken, _, err := jwt.NewParser().ParseUnverified(sessionResponse.Response.WeakToken, jwt.MapClaims{})
	if err != nil {
		return nil, eris.Errorf("weak token was invalid JWT, credentials probably incorrect: %v", err)
	}

	weakTokenSubject, err := weakToken.Claims.GetSubject()
	if err != nil {
		return nil, eris.Errorf("weak token was missing subject claim, credentials probably incorrect: %v", err)
	}

	steamID, err := steamid.ParseSteamID64(weakTokenSubject)
	if err != nil {
		return nil, eris.Errorf("weak token Sub returned invalid steamid64: %v", err)
	}

	code, err := options.AccountState.totpState.GenerateTotpCode("conf", totp.Time(0))
	if err != nil {
		return nil, eris.Errorf("error generating totp code failed: %v", err)
	}

	err = authClient.SubmitSteamGuardCode(ctx, sessionResponse.Response.ClientId, steamID, code)
	if err != nil {
		return nil, eris.Errorf("error submitting totp code: %v", err)
	}

	mobileConfClient, err := mobileconf.NewClient(
		options.AccountState.totpState,
		steamID,
		twoFactorClient,
		webTransport,
	)
	if err != nil {
		return nil, eris.Errorf("mobileconf.NewTransport failed: %v", err)
	}

	webSession := &WebSession{
		state:            options.AccountState,
		transport:        webTransport,
		authClient:       authClient,
		mobileConfClient: mobileConfClient,
		tradeOfferClient: &tradeoffer.Client{
			Transport:     webTransport,
			SessionIdFunc: GetSessionId,
		},
		twoFactorClient: twoFactorClient,
		communityClient: &community.Client{
			Transport: webTransport,
		},
		clientId:        sessionResponse.Response.ClientId,
		requestId:       sessionResponse.Response.RequestId,
		steamId:         steamID,
		refreshInterval: sessionResponse.Response.Interval,
	}

	err = webSession.pollSession(ctx)
	if err != nil {
		return nil, err
	}

	// N.B. we need a refresh token in order to get an access token, which we need in order to create the
	// steamLoginSecure web cookie
	if len(webSession.refreshToken) == 0 {
		return nil, eris.Errorf("no refresh token found in poll response")
	}

	return webSession, nil
}

func (w *WebSession) pollSession(ctx context.Context) error {
	pollResponse, err := w.authClient.PollSessionStatus(ctx, w.clientId, w.requestId)
	if err != nil {
		return eris.Errorf("PollSessionStatus failed: %v", err)
	}

	if len(pollResponse.Response.NewClientID) > 0 {
		w.clientId = pollResponse.Response.NewClientID
	}

	// only attempt to refresh if a refresh token was given to us
	if len(pollResponse.Response.RefreshToken) == 0 {
		return nil
	}

	oldRefreshToken := w.refreshToken

	// TODO: do we need to update state.accountName with pollResponse.Response.AccountName?
	w.accessToken = pollResponse.Response.AccessToken
	w.refreshToken = pollResponse.Response.RefreshToken
	if len(w.accessToken) == 0 {
		// under some circumstances, the access token may not be issued by steam when polling login. We may need to
		// establish the access token ourselves.
		accessTokenResponse, accessTokenErr := w.authClient.GenerateAccessTokenForApp(ctx, w.refreshToken, false)
		if accessTokenErr != nil {
			return eris.Errorf("GenerateAccessTokenForApp failed: %v", accessTokenErr)
		}

		w.accessToken = accessTokenResponse.Response.AccessToken
		w.refreshToken = accessTokenResponse.Response.RefreshToken
	}

	refreshTokenJwt, _, err := jwt.NewParser().ParseUnverified(w.refreshToken, jwt.MapClaims{})
	if err != nil {
		return eris.Errorf("refresh token was invalid JWT: %v", err)
	}

	if _, err := refreshTokenJwt.Claims.GetExpirationTime(); err != nil {
		return eris.Errorf("refresh token was missing expiration claim: %v", err)
	}

	w.jwt = refreshTokenJwt

	if oldRefreshToken != w.refreshToken {
		err = w.finalizeLogin()
		if err != nil {
			return eris.Errorf("finalizeLogin failed: %v", err)
		}
	}

	return nil
}

func (w *WebSession) finalizeLogin() error {
	sessionIdBuffer := [12]byte{}
	_, err := rand.Read(sessionIdBuffer[:])
	if err != nil {
		return eris.Errorf("error creating sessionid bytes: %v", err)
	}

	sessionIdBytes := make([]byte, hex.EncodedLen(len(sessionIdBuffer)))
	_ = hex.Encode(sessionIdBytes, sessionIdBuffer[:])

	steamLoginSecure := fmt.Sprintf("%s||%s", w.steamId.String(), w.accessToken)
	cookieUrl := &url.URL{Scheme: "https", Host: "steamcommunity.com", Path: "/"}
	w.transport.CookieJar().SetCookies(cookieUrl, []*http.Cookie{
		{
			Name:  "sessionid",
			Value: string(sessionIdBytes),
		},
		{
			Name:  "steamLoginSecure",
			Value: url.QueryEscape(steamLoginSecure),
		},
	})

	// TODO: do i need to query finalizelogin? https://github.com/DoctorMcKay/node-steam-session/blob/811dadd2bfcc11de7861fff7442cb4a44ab61955/src/LoginSession.ts#L819-L835

	return nil
}

func (w *WebSession) BeginPolling(ctx context.Context) {
	go func() {
		ticker := time.Tick(time.Duration(w.refreshInterval) * time.Second)
		for range ticker {
			select {
			case <-ctx.Done():
				return
			default:
			}

			err := w.pollSession(ctx)
			if err != nil {
				log.Printf("Error polling session: %v", err)
			}
		}
	}()
}

func (w *WebSession) SteamId() steamid.SteamID {
	return w.steamId
}

func GetSessionId(transport api.Transport) (string, error) {
	steamUrl := &url.URL{Scheme: "https", Host: "steamcommunity.com", Path: "/"}
	steamCookies := transport.CookieJar().Cookies(steamUrl)
	for _, cookie := range steamCookies {
		if strings.ToLower(cookie.Name) == "sessionid" {
			return cookie.Value, nil
		}
	}

	return "", eris.Errorf("could not find sessionid cookie")
}

func (w *WebSession) SessionId() (string, error) {
	return GetSessionId(w.transport)
}

func (w *WebSession) CommunityClient() community.Api {
	return w.communityClient
}

func (w *WebSession) EconClient() econ.Api {
	return w.econClient
}

func (w *WebSession) MobileConfClient() mobileconf.Api {
	return w.mobileConfClient
}

func (w *WebSession) Tf2EconClient() tf2econ.Api {
	return w.tf2EconClient
}

func (w *WebSession) TradeOfferClient() tradeoffer.Api {
	return w.tradeOfferClient
}
