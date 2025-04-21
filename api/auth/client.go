package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/escrow-tf/steam/api/web"
	"github.com/escrow-tf/steam/gorsa"
	"github.com/escrow-tf/steam/steamid"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
)

type GetRsaKeyRequest struct {
	accountName string
}

func (g GetRsaKeyRequest) RequiresApiKey() bool {
	return false
}

func (g GetRsaKeyRequest) Method() string {
	return http.MethodGet
}

func (g GetRsaKeyRequest) Values() (url.Values, error) {
	values := make(url.Values)
	values.Add("account_name", g.accountName)
	return values, nil
}

func (g GetRsaKeyRequest) Url() string {
	return fmt.Sprintf("%v/IAuthenticationService/GetPasswordRSAPublicKey/v1/", web.BaseURL)
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
		return PublicRsaKey{}, fmt.Errorf("error parsing public key exponent %v", err)
	}

	modulus := big.NewInt(0)
	modulus, ok := modulus.SetString(r.Response.PublicKeyMod, 16)
	if !ok {
		return PublicRsaKey{}, fmt.Errorf("error parsing public key modulus")
	}

	return PublicRsaKey{
		PublicKey: rsa.PublicKey{
			E: int(exponent),
			N: modulus,
		},
		Timestamp: r.Response.Timestamp,
	}, nil
}

type Client struct {
	webClient *web.Transport
}

func NewClient(webClient *web.Transport) *Client {
	return &Client{webClient}
}

func (c Client) GetPublicRsaKey(accountName string) (PublicRsaKey, error) {
	request := GetRsaKeyRequest{accountName: accountName}
	var response GetRsaKeyResponse
	sendErr := c.webClient.Send(request, &response)
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
func (c Client) EncryptAccountPassword(accountName string, password string) (EncryptedPassword, error) {
	publicKey, err := c.GetPublicRsaKey(accountName)
	if err != nil {
		return EncryptedPassword{}, fmt.Errorf("GetPublicRsaKey failed: %v", err)
	}

	encryptedPassword, err := gorsa.EncryptPKCS1([]byte(password), &publicKey.PublicKey)
	if err != nil {
		return EncryptedPassword{}, fmt.Errorf("gorsa.EncryptPKCS1() failed: %v", err)
	}

	return EncryptedPassword{
		Base64:    base64.StdEncoding.EncodeToString(encryptedPassword),
		TimeStamp: publicKey.Timestamp,
	}, nil
}

type Persistence int

const (
	InvalidSessionPersistence    Persistence = -1
	EphemeralSessionPersistence  Persistence = 0
	PersistentSessionPersistence Persistence = 1
)

type PlatformType int

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
	AndroidUnknownOsType    int = -500
	DefaultGamingDeviceType     = 528
)

type DeviceDetails struct {
	FriendlyName     string       `json:"device_friendly_name"`
	PlatformType     PlatformType `json:"platform_type"`
	OsType           int          `json:"os_type"`
	GamingDeviceType int          `json:"gaming_device_type"`
}

type StartSessionRequest struct {
	AccountName         string
	EncryptedPassword   string
	EncryptionTimestamp string
	Persistence         Persistence
	DeviceDetails       DeviceDetails
	Language            int
	QosLevel            int
}

func (r StartSessionRequest) RequiresApiKey() bool {
	return false
}

func (r StartSessionRequest) Method() string {
	return http.MethodPost
}

func (r StartSessionRequest) Values() (url.Values, error) {
	deviceDetailsBytes, err := json.Marshal(r.DeviceDetails)
	if err != nil {
		return nil, fmt.Errorf("json marshal failed %v", err)
	}

	values := make(url.Values)
	values.Add("account_name", r.AccountName)
	values.Add("encrypted_password", r.EncryptedPassword)
	values.Add("encryption_timestamp", r.EncryptionTimestamp)
	values.Add("persistence", strconv.Itoa(int(r.Persistence)))
	values.Add("language", strconv.Itoa(r.Language))
	values.Add("qos_level", strconv.Itoa(r.QosLevel))
	values.Add("device_details", string(deviceDetailsBytes))
	return values, nil
}

func (r StartSessionRequest) Url() string {
	return fmt.Sprintf("%v/IAuthenticationService/BeginAuthSessionViaCredentials/v1/", web.BaseURL)
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

func (c Client) StartSessionWithCredentials(accountName string, password EncryptedPassword, deviceDetails DeviceDetails) (StartSessionResponse, error) {
	request := StartSessionRequest{
		AccountName:         accountName,
		EncryptedPassword:   password.Base64,
		EncryptionTimestamp: password.TimeStamp,
		DeviceDetails:       deviceDetails,
		Language:            0,
		QosLevel:            2,
	}
	var response StartSessionResponse
	sendErr := c.webClient.Send(request, &response)
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

func (r UpdateSessionWithSteamGuardCodeRequest) RequiresApiKey() bool {
	return false
}

func (r UpdateSessionWithSteamGuardCodeRequest) Method() string {
	return http.MethodPost
}

func (r UpdateSessionWithSteamGuardCodeRequest) Values() (url.Values, error) {
	values := make(url.Values)
	values.Add("client_id", r.ClientID)
	values.Add("steamid", r.SteamID)
	values.Add("code", r.Code)
	values.Add("code_type", strconv.Itoa(int(r.CodeType)))
	return values, nil
}

func (r UpdateSessionWithSteamGuardCodeRequest) Url() string {
	return fmt.Sprintf("%v/IAuthenticationService/UpdateAuthSessionWithSteamGuardCode/v1/", web.BaseURL)
}

func (c Client) SubmitSteamGuardCode(clientID string, steamID steamid.SteamID, code string) error {
	if !steamID.IsValidIndividual() {
		return fmt.Errorf("steamID is not valid individual: %v", steamID.String())
	}

	request := UpdateSessionWithSteamGuardCodeRequest{
		ClientID: clientID,
		SteamID:  steamID.String(),
		Code:     code,
		CodeType: DeviceCodeGuardType,
	}
	sendErr := c.webClient.Send(request, nil)
	if sendErr != nil {
		return sendErr
	}

	return nil
}

type PollSessionStatusRequest struct {
	ClientID  string
	RequestID string
}

func (r PollSessionStatusRequest) RequiresApiKey() bool {
	return false
}

func (r PollSessionStatusRequest) Method() string {
	return http.MethodPost
}

func (r PollSessionStatusRequest) Values() (url.Values, error) {
	values := make(url.Values)
	values.Add("client_id", r.ClientID)
	values.Add("request_id", r.RequestID)
	return values, nil
}

func (r PollSessionStatusRequest) Url() string {
	return fmt.Sprintf("%v/IAuthenticationService/PollAuthSessionStatus/v1/", web.BaseURL)
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

func (c Client) PollSessionStatus(clientID string, requestID string) (PollSessionStatusResponse, error) {
	request := PollSessionStatusRequest{
		ClientID:  clientID,
		RequestID: requestID,
	}
	var response PollSessionStatusResponse
	sendErr := c.webClient.Send(request, &response)
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

func (r GenerateAccessTokenRequest) RequiresApiKey() bool {
	return false
}

func (r GenerateAccessTokenRequest) Method() string {
	return http.MethodPost
}

func (r GenerateAccessTokenRequest) Values() (url.Values, error) {
	values := make(url.Values)
	values.Add("refresh_token", r.RefreshToken)
	values.Add("steamid", r.SteamID)
	values.Add("renewal_type", strconv.Itoa(int(r.RenewalType)))
	return values, nil
}

func (r GenerateAccessTokenRequest) Url() string {
	return fmt.Sprintf("%v/IAuthenticationService/GenerateAccessTokenForApp/v1/", web.BaseURL)
}

type GenerateAccessTokenResponse struct {
	Response struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	} `json:"response"`
}

func (c Client) GenerateAccessTokenForApp(refreshToken string, renew bool) (GenerateAccessTokenResponse, error) {
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
	sendErr := c.webClient.Send(request, &response)
	if sendErr != nil {
		return GenerateAccessTokenResponse{}, sendErr
	}

	return response, nil
}
