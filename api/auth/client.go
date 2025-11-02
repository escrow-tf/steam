package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/escrow-tf/steam/api"
	"github.com/escrow-tf/steam/gorsa"
	steamproto "github.com/escrow-tf/steam/proto/steam"
	"github.com/escrow-tf/steam/steamid"
	"github.com/escrow-tf/steam/steamlang"
	"github.com/rotisserie/eris"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	Transport api.Transport
}

type GetRsaKeyRequest struct {
	accountName string
}

func (g GetRsaKeyRequest) CacheTTL() time.Duration {
	return 0
}

func (g GetRsaKeyRequest) EnsureResponseSuccess(httpResponse *http.Response) error {
	return steamlang.EnsureSuccessResponse(httpResponse)
}

func (g GetRsaKeyRequest) Headers() (http.Header, error) {
	return nil, nil
}

func (g GetRsaKeyRequest) Retryable() bool {
	return true
}

func (g GetRsaKeyRequest) RequiresApiKey() bool {
	return false
}

func (g GetRsaKeyRequest) Method() string {
	return http.MethodGet
}

func (g GetRsaKeyRequest) OldValues() (url.Values, error) {
	values := make(url.Values)
	values.Add("account_name", g.accountName)
	return values, nil
}

func (g GetRsaKeyRequest) Values() (interface{}, error) {
	values := make(url.Values)
	values.Add("account_name", g.accountName)
	return values, nil
}

func (g GetRsaKeyRequest) Url() string {
	return fmt.Sprintf("%v/IAuthenticationService/GetPasswordRSAPublicKey/v1/", api.BaseURL)
}

type PublicRsaKey struct {
	PublicKey rsa.PublicKey
	Timestamp string
}

type GetRsaKeyResponse struct {
	Response struct {
		PublicKeyMod string `json:"publickey_mod"`
		PublicKeyExp string `json:"publickey_exp"`
		Timestamp    string `json:"timestamp"`
	} `json:"response"`
}

func (r GetRsaKeyResponse) PublicKey() (PublicRsaKey, error) {
	exponent, err := strconv.ParseInt(r.Response.PublicKeyExp, 16, 64)
	if err != nil {
		return PublicRsaKey{}, eris.Errorf("error parsing public key exponent %v", err)
	}

	modulus := big.NewInt(0)
	modulus, ok := modulus.SetString(r.Response.PublicKeyMod, 16)
	if !ok {
		return PublicRsaKey{}, eris.Errorf("error parsing public key modulus")
	}

	return PublicRsaKey{
		PublicKey: rsa.PublicKey{
			E: int(exponent),
			N: modulus,
		},
		Timestamp: r.Response.Timestamp,
	}, nil
}

func (c *Client) GetPublicRsaKey(ctx context.Context, accountName string) (PublicRsaKey, error) {
	request := GetRsaKeyRequest{accountName: accountName}
	var response GetRsaKeyResponse
	sendErr := c.Transport.Send(ctx, request, &response)
	if sendErr != nil {
		return PublicRsaKey{}, sendErr
	}

	publicRsaKey, rsaKeyErr := response.PublicKey()
	if rsaKeyErr != nil {
		return PublicRsaKey{}, rsaKeyErr
	}

	return publicRsaKey, nil
}

type EncryptedPassword struct {
	Base64    string
	TimeStamp string
}

// EncryptAccountPassword
// Retrieves the RSA key for the specified accountName, and encrypted the given password using the RSA key.
func (c *Client) EncryptAccountPassword(
	ctx context.Context,
	accountName string,
	password string,
) (EncryptedPassword, error) {
	publicKey, err := c.GetPublicRsaKey(ctx, accountName)
	if err != nil {
		return EncryptedPassword{}, eris.Errorf("GetPublicRsaKey failed: %v", err)
	}

	encryptedPassword, err := gorsa.EncryptPKCS1([]byte(password), &publicKey.PublicKey)
	if err != nil {
		return EncryptedPassword{}, eris.Errorf("gorsa.EncryptPKCS1() failed: %v", err)
	}

	return EncryptedPassword{
		Base64:    base64.StdEncoding.EncodeToString(encryptedPassword),
		TimeStamp: publicKey.Timestamp,
	}, nil
}

type Persistence int

//goland:noinspection GoUnusedConst
const (
	InvalidSessionPersistence    Persistence = -1
	EphemeralSessionPersistence  Persistence = 0
	PersistentSessionPersistence Persistence = 1
)

type PlatformType int

//goland:noinspection GoUnusedConst
const (
	UnknownPlatformType PlatformType = iota
	SteamClientPlatformType
	WebBrowserPlatformType
	MobileAppPlatformType
)

type TokenRenewalType int

const (
	NoneRenewalType TokenRenewalType = iota
	AllowRenewalType
)

type GuardType int

//goland:noinspection GoUnusedConst
const (
	UnknownGuardType GuardType = iota
	NoneGuardType
	EmailCodeGuardType
	DeviceCodeGuardType
	DeviceConfirmationGuardType
	EmailConfirmationGuardType
	MachineTokenGuardType
	LegacyMachineAuthGuardType
)

const (
	AndroidUnknownOsType    int32 = -500
	DefaultGamingDeviceType       = 528
)

type DeviceDetails struct {
	FriendlyName     string                            `json:"device_friendly_name"`
	PlatformType     steamproto.EAuthTokenPlatformType `json:"platform_type"`
	OsType           int32                             `json:"os_type"`
	GamingDeviceType uint32                            `json:"gaming_device_type"`
}

type StartSessionRequest struct {
	AccountName         string
	EncryptedPassword   string
	EncryptionTimestamp string
	Persistence         steamproto.ESessionPersistence
	DeviceDetails       DeviceDetails
	Language            uint32
	QosLevel            int32
}

func (r StartSessionRequest) CacheTTL() time.Duration {
	return 0
}

func (r StartSessionRequest) EnsureResponseSuccess(httpResponse *http.Response) error {
	return steamlang.EnsureSuccessResponse(httpResponse)
}

func (r StartSessionRequest) Headers() (http.Header, error) {
	return nil, nil
}

func (r StartSessionRequest) Retryable() bool {
	return false
}

func (r StartSessionRequest) RequiresApiKey() bool {
	return false
}

func (r StartSessionRequest) Method() string {
	return http.MethodPost
}

func (r StartSessionRequest) OldValues() (url.Values, error) {
	deviceDetailsBytes, err := json.Marshal(r.DeviceDetails)
	if err != nil {
		return nil, eris.Errorf("json marshal failed %v", err)
	}

	var websiteId string
	switch r.DeviceDetails.PlatformType {
	case steamproto.EAuthTokenPlatformType_k_EAuthTokenPlatformType_WebBrowser:
		websiteId = "Community"
	case steamproto.EAuthTokenPlatformType_k_EAuthTokenPlatformType_MobileApp:
		websiteId = "Mobile"
	case steamproto.EAuthTokenPlatformType_k_EAuthTokenPlatformType_SteamClient:
		websiteId = "Unknown"
	default:
		return nil, eris.Errorf("unsupported platform type %v", r.DeviceDetails.PlatformType)
	}
	values := make(url.Values)
	values.Add("device_friendly_name", r.DeviceDetails.FriendlyName)
	values.Add("account_name", r.AccountName)
	values.Add("encrypted_password", r.EncryptedPassword)
	values.Add("encryption_timestamp", r.EncryptionTimestamp)
	values.Add("remember_login", strconv.FormatBool(r.Persistence == steamproto.ESessionPersistence_k_ESessionPersistence_Persistent))
	values.Add("platform_type", strconv.Itoa(int(r.DeviceDetails.PlatformType)))
	values.Add("persistence", strconv.Itoa(int(r.Persistence)))
	values.Add("website_id", websiteId)
	values.Add("device_details", string(deviceDetailsBytes))
	values.Add("guard_data", "")
	values.Add("language", strconv.FormatUint(uint64(r.Language), 10))
	values.Add("qos_level", strconv.Itoa(int(r.QosLevel)))
	return values, nil
}

func (r StartSessionRequest) Values() (interface{}, error) {
	var websiteId string
	switch r.DeviceDetails.PlatformType {
	case steamproto.EAuthTokenPlatformType_k_EAuthTokenPlatformType_WebBrowser:
		websiteId = "Community"
	case steamproto.EAuthTokenPlatformType_k_EAuthTokenPlatformType_MobileApp:
		websiteId = "Mobile"
	case steamproto.EAuthTokenPlatformType_k_EAuthTokenPlatformType_SteamClient:
		websiteId = "Unknown"
	default:
		return nil, eris.Errorf("unsupported platform type %v", r.DeviceDetails.PlatformType)
	}

	rememberLogin := r.Persistence == steamproto.ESessionPersistence_k_ESessionPersistence_Persistent

	request := steamproto.CAuthentication_BeginAuthSessionViaCredentials_Request{
		AccountName:         &r.AccountName,
		EncryptedPassword:   nil,
		EncryptionTimestamp: nil,
		RememberLogin:       &rememberLogin,
		Persistence:         &r.Persistence,
		WebsiteId:           &websiteId,
		DeviceDetails: &steamproto.CAuthentication_DeviceDetails{
			DeviceFriendlyName: &r.DeviceDetails.FriendlyName,
			PlatformType:       &r.DeviceDetails.PlatformType,
			OsType:             &r.DeviceDetails.OsType,
			//GamingDeviceType:   &r.DeviceDetails.GamingDeviceType,
			//ClientCount:        nil,
			//MachineId:          nil,
			//AppType:            nil,
		},
		GuardData: nil,
		Language:  &r.Language,
		QosLevel:  &r.QosLevel,
	}

	marshalled, err := proto.Marshal(&request)
	if err != nil {
		return nil, eris.Errorf("marshal failed %v", err)
	}

	return fmt.Sprintf("input_protobuf_encoded=%s", url.QueryEscape(base64.StdEncoding.EncodeToString(marshalled))), nil
}

func (r StartSessionRequest) Url() string {
	return fmt.Sprintf("%v/IAuthenticationService/BeginAuthSessionViaCredentials/v1/", api.BaseURL)
}

type StartSessionResponse struct {
	Response struct {
		ClientId             string `json:"client_id"`
		RequestId            string `json:"request_id"`
		Interval             int    `json:"interval"`
		SteamId              string `json:"steam_id"`
		WeakToken            string `json:"weak_token,omitempty"`
		AgreementSessionUrl  string `json:"agreement_session_url,omitempty"`
		ExtendedErrorMessage string `json:"extended_error_message,omitempty"`
		AllowedConfirmations []struct {
			ConfirmationType  GuardType `json:"confirmation_type"`
			AssociatedMessage string    `json:"associated_message,omitempty"`
		} `json:"allowed_confirmations,omitempty"`
	} `json:"response"`
}

func (c *Client) StartSessionWithCredentials(
	ctx context.Context,
	accountName string,
	password EncryptedPassword,
	deviceDetails DeviceDetails,
) (StartSessionResponse, error) {
	request := StartSessionRequest{
		AccountName:         accountName,
		EncryptedPassword:   password.Base64,
		EncryptionTimestamp: password.TimeStamp,
		DeviceDetails:       deviceDetails,
		Language:            0,
		QosLevel:            2,
	}
	var response StartSessionResponse
	sendErr := c.Transport.Send(ctx, request, &response)
	if sendErr != nil {
		return StartSessionResponse{}, sendErr
	}

	return response, nil
}

type UpdateSessionWithSteamGuardCodeRequest struct {
	ClientID string
	SteamID  string
	Code     string
	CodeType GuardType
}

func (r UpdateSessionWithSteamGuardCodeRequest) Values() (interface{}, error) {
	return r.OldValues()
}

func (r UpdateSessionWithSteamGuardCodeRequest) CacheTTL() time.Duration {
	return 0
}

func (r UpdateSessionWithSteamGuardCodeRequest) EnsureResponseSuccess(httpResponse *http.Response) error {
	return steamlang.EnsureSuccessResponse(httpResponse)
}

func (r UpdateSessionWithSteamGuardCodeRequest) Headers() (http.Header, error) {
	return nil, nil
}

func (r UpdateSessionWithSteamGuardCodeRequest) Retryable() bool {
	return false
}

func (r UpdateSessionWithSteamGuardCodeRequest) RequiresApiKey() bool {
	return false
}

func (r UpdateSessionWithSteamGuardCodeRequest) Method() string {
	return http.MethodPost
}

func (r UpdateSessionWithSteamGuardCodeRequest) OldValues() (url.Values, error) {
	values := make(url.Values)
	values.Add("client_id", r.ClientID)
	values.Add("steamid", r.SteamID)
	values.Add("code", r.Code)
	values.Add("code_type", strconv.Itoa(int(r.CodeType)))
	return values, nil
}

func (r UpdateSessionWithSteamGuardCodeRequest) Url() string {
	return fmt.Sprintf("%v/IAuthenticationService/UpdateAuthSessionWithSteamGuardCode/v1/", api.BaseURL)
}

func (c *Client) SubmitSteamGuardCode(
	ctx context.Context,
	clientID string,
	steamID steamid.SteamID,
	code string,
) error {
	if !steamID.IsValidIndividual() {
		return eris.Errorf("steamID is not valid individual: %v", steamID.String())
	}

	request := UpdateSessionWithSteamGuardCodeRequest{
		ClientID: clientID,
		SteamID:  steamID.String(),
		Code:     code,
		CodeType: DeviceCodeGuardType,
	}
	sendErr := c.Transport.Send(ctx, request, nil)
	if sendErr != nil {
		return sendErr
	}

	return nil
}

type PollSessionStatusRequest struct {
	ClientID  string
	RequestID string
}

func (r PollSessionStatusRequest) Values() (interface{}, error) {
	return r.OldValues()
}

func (r PollSessionStatusRequest) CacheTTL() time.Duration {
	return 0
}

func (r PollSessionStatusRequest) EnsureResponseSuccess(httpResponse *http.Response) error {
	return steamlang.EnsureSuccessResponse(httpResponse)
}

func (r PollSessionStatusRequest) Headers() (http.Header, error) {
	return nil, nil
}

func (r PollSessionStatusRequest) Retryable() bool {
	return false
}

func (r PollSessionStatusRequest) RequiresApiKey() bool {
	return false
}

func (r PollSessionStatusRequest) Method() string {
	return http.MethodPost
}

func (r PollSessionStatusRequest) OldValues() (url.Values, error) {
	values := make(url.Values)
	values.Add("client_id", r.ClientID)
	values.Add("request_id", r.RequestID)
	return values, nil
}

func (r PollSessionStatusRequest) Url() string {
	return fmt.Sprintf("%v/IAuthenticationService/PollAuthSessionStatus/v1/", api.BaseURL)
}

type PollSessionStatusResponse struct {
	Response struct {
		NewClientID          string `json:"new_client_id,omitempty"`
		NewChallenge         string `json:"new_challenge,omitempty"`
		RefreshToken         string `json:"refresh_token,omitempty"`
		AccessToken          string `json:"access_token,omitempty"`
		HadRemoteInteraction bool   `json:"had_remote_interaction,omitempty"`
		AccountName          string `json:"account_name,omitempty"`
	} `json:"response"`
}

func (c *Client) PollSessionStatus(
	ctx context.Context,
	clientID string,
	requestID string,
) (PollSessionStatusResponse, error) {
	request := PollSessionStatusRequest{
		ClientID:  clientID,
		RequestID: requestID,
	}
	var response PollSessionStatusResponse
	sendErr := c.Transport.Send(ctx, request, &response)
	if sendErr != nil {
		return PollSessionStatusResponse{}, sendErr
	}
	return response, nil
}

type GenerateAccessTokenRequest struct {
	RefreshToken string           `json:"refresh_token"`
	SteamID      string           `json:"steamid"`
	RenewalType  TokenRenewalType `json:"renewal_type"`
}

func (r GenerateAccessTokenRequest) Values() (interface{}, error) {
	return r.OldValues()
}

func (r GenerateAccessTokenRequest) CacheTTL() time.Duration {
	return 0
}

func (r GenerateAccessTokenRequest) EnsureResponseSuccess(httpResponse *http.Response) error {
	return steamlang.EnsureSuccessResponse(httpResponse)
}

func (r GenerateAccessTokenRequest) Headers() (http.Header, error) {
	return nil, nil
}

func (r GenerateAccessTokenRequest) Retryable() bool {
	return false
}

func (r GenerateAccessTokenRequest) RequiresApiKey() bool {
	return false
}

func (r GenerateAccessTokenRequest) Method() string {
	return http.MethodPost
}

func (r GenerateAccessTokenRequest) OldValues() (url.Values, error) {
	values := make(url.Values)
	values.Add("refresh_token", r.RefreshToken)
	values.Add("steamid", r.SteamID)
	values.Add("renewal_type", strconv.Itoa(int(r.RenewalType)))
	return values, nil
}

func (r GenerateAccessTokenRequest) Url() string {
	return fmt.Sprintf("%v/IAuthenticationService/GenerateAccessTokenForApp/v1/", api.BaseURL)
}

type GenerateAccessTokenResponse struct {
	Response struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	} `json:"response"`
}

func (c *Client) GenerateAccessTokenForApp(
	ctx context.Context,
	refreshToken string,
	renew bool,
) (GenerateAccessTokenResponse, error) {
	jwt, err := DecodeSimpleJwt(refreshToken)
	if err != nil {
		return GenerateAccessTokenResponse{}, err
	}

	renewalType := NoneRenewalType
	if renew {
		renewalType = AllowRenewalType
	}

	request := GenerateAccessTokenRequest{
		RefreshToken: refreshToken,
		SteamID:      jwt.Sub,
		RenewalType:  renewalType,
	}
	var response GenerateAccessTokenResponse
	sendErr := c.Transport.Send(ctx, request, &response)
	if sendErr != nil {
		return GenerateAccessTokenResponse{}, sendErr
	}

	return response, nil
}
