package api

import (
	"encoding/json"
	"fmt"
	"github.com/escrow-tf/steam/steamlang"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"reflect"
	"strings"
)

type TokenRenewalType int
type GuardType int
type Persistence int
type PlatformType int

const (
	NoneRenewalType TokenRenewalType = iota
	AllowRenewalType
)

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

//goland:noinspection GoUnusedConst,GoNameStartsWithPackageName
const (
	UnknownPlatformType PlatformType = iota
	SteamClientPlatformType
	WebBrowserPlatformType
	MobileAppPlatformType
)

//goland:noinspection GoUnusedConst
const (
	InvalidSessionPersistence    Persistence = -1
	EphemeralSessionPersistence  Persistence = 0
	PersistentSessionPersistence Persistence = 1
)

//goland:noinspection GoUnusedConst
const (
	AndroidUnknownOsType    int = -500
	DefaultGamingDeviceType     = 528
)

//goland:noinspection GoUnusedConst
const JsonContentType = "application/json"
const FormContentType = "application/x-www-form-urlencoded"

type DeviceDetails struct {
	FriendlyName     string       `json:"device_friendly_name"`
	PlatformType     PlatformType `json:"platform_type"`
	OsType           int          `json:"os_type"`
	GamingDeviceType int          `json:"gaming_device_type"`
}

const BaseURL = "https://api.steampowered.com"

type SteamRequest interface {
	Retryable() bool
	RequiresApiKey() bool
	Method() string
	Url() string
	Values() (url.Values, error)
}

type Transport struct {
	webApiKey   string
	client      *http.Client
	retryClient *retryablehttp.Client
}

func NewTransport(webApiKey string) *Transport {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic("Failed to create cookie jar, which should never happen as cookiejar.New does not return any errors")
	}

	cookieUrl := &url.URL{Scheme: "https", Host: "steamcommunity.com", Path: "/"}
	jar.SetCookies(cookieUrl, []*http.Cookie{
		&http.Cookie{
			Name:  "mobileClient",
			Value: "android",
		},
		&http.Cookie{
			Name:  "mobileClientVersion",
			Value: "777777 3.0.0",
		},
	})

	httpClient := &http.Client{
		Transport: cleanhttp.DefaultPooledTransport(),
		Jar:       jar,
	}

	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient = httpClient

	return &Transport{
		webApiKey:   webApiKey,
		client:      httpClient,
		retryClient: retryClient,
	}
}

func (c Transport) CookieJar() http.CookieJar {
	return c.client.Jar
}

// Send sends a specialized HTTP Request to steam.
func (c Transport) Send(request SteamRequest, response any) error {
	rv := reflect.ValueOf(response)
	if !rv.IsNil() && rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("response type must be a pointer when not nil")
	}

	httpMethod := request.Method()

	values, valuesErr := request.Values()
	if valuesErr != nil {
		return valuesErr
	}

	requestUrl := request.Url()
	if !strings.HasSuffix(requestUrl, "?") {
		requestUrl += "?"
	}

	if request.RequiresApiKey() {
		if values == nil {
			values = make(url.Values)
		}
		values.Add("key", c.webApiKey)
	}

	var httpBody io.Reader
	if values != nil {
		if httpMethod == http.MethodGet {
			requestUrl += values.Encode()
		} else {
			httpBody = strings.NewReader(values.Encode())
		}
	}

	httpRequest, httpRequestErr := http.NewRequest(httpMethod, requestUrl, httpBody)
	if httpRequestErr != nil {
		return httpRequestErr
	}

	httpRequest.Header.Add("Accept", JsonContentType)
	httpRequest.Header.Add("User-Agent", "okhttp/3.12.12")
	if httpMethod == http.MethodPost {
		httpRequest.Header.Add("Content-Type", FormContentType)
	}

	httpClient := c.client
	if request.Retryable() {
		httpClient = c.retryClient.StandardClient()
	}

	httpResponse, httpResponseErr := httpClient.Do(httpRequest)
	if httpResponseErr != nil {
		return fmt.Errorf("request to Steam failed: %v", httpResponseErr)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing steam response body: %v", err)
		}
	}(httpResponse.Body)

	if err := steamlang.EnsureSuccessResponse(httpResponse); err != nil {
		return err
	}

	if err := steamlang.EnsureEResultResponse(httpResponse); err != nil {
		return err
	}

	if response != nil {
		responseBody, err := io.ReadAll(httpResponse.Body)
		if err != nil {
			return fmt.Errorf("couldn't read request: %v", err)
		}

		err = json.Unmarshal(responseBody, response)
		if err != nil {
			return fmt.Errorf("couldnt unmarshal response: %v", err)
		}
	}

	return nil
}

func (c Transport) HttpClient() *http.Client {
	return c.client
}
