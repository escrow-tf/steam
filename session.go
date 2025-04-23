package steam

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/escrow-tf/steam/api"
	"github.com/escrow-tf/steam/api/auth"
	"github.com/escrow-tf/steam/api/community"
	"github.com/escrow-tf/steam/api/mobileconf"
	"github.com/escrow-tf/steam/api/tradeoffer"
	"github.com/escrow-tf/steam/api/twofactor"
	"github.com/escrow-tf/steam/steamid"
	"github.com/escrow-tf/steam/totp"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type AccountState struct {
	accountName string
	password    string
	totpState   *totp.State
}

func NewAccountState(accountName string, password string, sharedSecret string, identitySecret string) (*AccountState, error) {
	state, err := totp.NewState(sharedSecret, identitySecret)
	if err != nil {
		return nil, fmt.Errorf("NewAccountState failed %v", err)
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
	transport        *api.Transport
	authClient       *auth.Client
	mobileConfClient *mobileconf.Client
	tradeOfferClient *tradeoffer.Client
	twoFactorClient  *twofactor.Client
	communityClient  *community.Client

	clientId        string
	requestId       string
	steamId         steamid.SteamID
	jwt             *jwt.Token
	refreshToken    string
	accessToken     string
	refreshInterval int
}

func (w *WebSession) Transport() *api.Transport {
	return w.transport
}

func Authenticate(accountState *AccountState, webApiKey string) (*WebSession, error) {
	deviceHostName, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("os.Hostname() failed: %v", err)
	}

	webTransport := api.NewTransport(webApiKey)
	authClient := &auth.Client{
		Transport: webTransport,
	}
	twoFactorClient := &twofactor.Client{
		Transport: webTransport,
	}

	alignErr := twoFactorClient.AlignTime()
	if alignErr != nil {
		return nil, fmt.Errorf("twoFactorClient.AlignTime() failed: %v", alignErr)
	}

	encryptedPassword, err := authClient.EncryptAccountPassword(accountState.accountName, accountState.password)
	if err != nil {
		return nil, fmt.Errorf("EncryptPassword failed %v", err)
	}

	deviceDetails := auth.DeviceDetails{
		FriendlyName:     fmt.Sprintf("%s (steamguard-cli)", deviceHostName),
		PlatformType:     auth.MobileAppPlatformType,
		OsType:           auth.AndroidUnknownOsType,
		GamingDeviceType: auth.DefaultGamingDeviceType,
	}

	sessionResponse, err := authClient.StartSessionWithCredentials(accountState.accountName, encryptedPassword, deviceDetails)
	if err != nil {
		return nil, fmt.Errorf("StartSessionWithCredentials failed %v", err)
	}

	hasDeviceCodeType := false
	for _, allowedConfirmation := range sessionResponse.Response.AllowedConfirmations {
		if allowedConfirmation.ConfirmationType == auth.DeviceCodeGuardType {
			hasDeviceCodeType = true
		}
	}

	if !hasDeviceCodeType {
		return nil, fmt.Errorf("DeviceCode auth not in list of allowed confirmations")
	}

	weakToken, _, err := jwt.NewParser().ParseUnverified(sessionResponse.Response.WeakToken, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("weak token was invalid JWT, credentials probably incorrect: %v", err)
	}

	weakTokenSubject, err := weakToken.Claims.GetSubject()
	if err != nil {
		return nil, fmt.Errorf("weak token was missing subject claim, credentials probably incorrect: %v", err)
	}

	steamID, err := steamid.ParseSteamID64(weakTokenSubject)
	if err != nil {
		return nil, fmt.Errorf("weak token Sub returned invalid steamid64: %v", err)
	}

	code, err := accountState.totpState.GenerateTotpCode("conf", totp.Time(0))
	if err != nil {
		return nil, fmt.Errorf("error generating totp code failed: %v", err)
	}

	err = authClient.SubmitSteamGuardCode(sessionResponse.Response.ClientId, steamID, code)
	if err != nil {
		return nil, fmt.Errorf("error submitting totp code: %v", err)
	}

	mobileConfClient, err := mobileconf.NewClient(accountState.totpState, steamID, twoFactorClient, webTransport)
	if err != nil {
		return nil, fmt.Errorf("mobileconf.NewTransport failed: %v", err)
	}

	webSession := &WebSession{
		state:            accountState,
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

	err = webSession.pollSession()
	if err != nil {
		return nil, err
	}

	// N.B. we need a refresh token in order to get an access token, which we need in order to create the
	// steamLoginSecure web cookie
	if len(webSession.refreshToken) == 0 {
		return nil, fmt.Errorf("no refresh token found in poll response")
	}

	return webSession, nil
}

func (w *WebSession) pollSession() error {
	pollResponse, err := w.authClient.PollSessionStatus(w.clientId, w.requestId)
	if err != nil {
		return fmt.Errorf("PollSessionStatus failed: %v", err)
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
		accessTokenResponse, accessTokenErr := w.authClient.GenerateAccessTokenForApp(w.refreshToken, false)
		if accessTokenErr != nil {
			return fmt.Errorf("GenerateAccessTokenForApp failed: %v", accessTokenErr)
		}

		w.accessToken = accessTokenResponse.Response.AccessToken
		w.refreshToken = accessTokenResponse.Response.RefreshToken
	}

	refreshTokenJwt, _, err := jwt.NewParser().ParseUnverified(w.refreshToken, jwt.MapClaims{})
	if err != nil {
		return fmt.Errorf("refresh token was invalid JWT: %v", err)
	}

	if _, err := refreshTokenJwt.Claims.GetExpirationTime(); err != nil {
		return fmt.Errorf("refresh token was missing expiration claim: %v", err)
	}

	w.jwt = refreshTokenJwt

	if oldRefreshToken != w.refreshToken {
		err = w.finalizeLogin()
		if err != nil {
			return fmt.Errorf("finalizeLogin failed: %v", err)
		}
	}

	return nil
}

func (w *WebSession) finalizeLogin() error {
	sessionIdBuffer := [12]byte{}
	_, err := rand.Read(sessionIdBuffer[:])
	if err != nil {
		return fmt.Errorf("error creating sessionid bytes: %v", err)
	}

	sessionIdBytes := make([]byte, hex.EncodedLen(len(sessionIdBuffer)))
	_ = hex.Encode(sessionIdBytes, sessionIdBuffer[:])

	steamLoginSecure := fmt.Sprintf("%s||%s", w.steamId.String(), w.accessToken)
	cookieUrl := &url.URL{Scheme: "https", Host: "steamcommunity.com", Path: "/"}
	w.transport.CookieJar().SetCookies(cookieUrl, []*http.Cookie{
		&http.Cookie{
			Name:  "sessionid",
			Value: string(sessionIdBytes),
		},
		&http.Cookie{
			Name:  "steamLoginSecure",
			Value: url.QueryEscape(steamLoginSecure),
		},
	})

	// TODO: do i need to query finalizelogin? https://github.com/DoctorMcKay/node-steam-session/blob/811dadd2bfcc11de7861fff7442cb4a44ab61955/src/LoginSession.ts#L819-L835

	return nil
}

func (w *WebSession) BeginPolling() {
	// TODO: how do we cancel polling?
	go func() {
		time.Sleep(time.Duration(w.refreshInterval) * time.Second)

		err := w.pollSession()
		if err != nil {
			log.Printf("Error polling session: %v", err)
		}
	}()
}

func (w *WebSession) SteamId() steamid.SteamID {
	return w.steamId
}

func GetSessionId(transport *api.Transport) (string, error) {
	steamUrl := &url.URL{Scheme: "https", Host: "steamcommunity.com", Path: "/"}
	steamCookies := transport.CookieJar().Cookies(steamUrl)
	for _, cookie := range steamCookies {
		if strings.ToLower(cookie.Name) == "sessionid" {
			return cookie.Value, nil
		}
	}

	return "", fmt.Errorf("could not find sessionid cookie")
}

func (w *WebSession) SessionId() (string, error) {
	return GetSessionId(w.transport)
}

func (w *WebSession) MobileConfClient() *mobileconf.Client {
	return w.mobileConfClient
}

func (w *WebSession) TradeOfferClient() *tradeoffer.Client {
	return w.tradeOfferClient
}

func (w *WebSession) CommunityClient() *community.Client {
	return w.communityClient
}
