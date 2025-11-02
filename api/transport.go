package api

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/escrow-tf/steam/steamlang"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rotisserie/eris"
)

type TokenRenewalType int
type GuardType int
type Persistence int
type PlatformType int

//goland:noinspection GoUnusedConst
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

type Request interface {
	Retryable() bool
	CacheTTL() time.Duration
	RequiresApiKey() bool
	Method() string
	Url() string
	Values() (url.Values, error)
	Headers() (http.Header, error)
	EnsureResponseSuccess(httpResponse *http.Response) error
}

type Transport interface {
	CookieJar() http.CookieJar
	Send(ctx context.Context, request Request, response any) error
	HttpClient() *http.Client
}

type HttpTransport struct {
	webApiKey   string
	client      *http.Client
	retryClient *retryablehttp.Client
}

type HttpTransportOptions struct {
	WebApiKey     string
	ResponseCache CacheAdaptor
}

func NewTransport(options HttpTransportOptions) *HttpTransport {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic("Failed to create cookie jar, which should never happen as cookiejar.New does not return any errors")
	}

	cookieUrl := &url.URL{Scheme: "https", Host: "steamcommunity.com", Path: "/"}
	jar.SetCookies(cookieUrl, []*http.Cookie{
		{
			Name:  "mobileClient",
			Value: "android",
		},
		{
			Name:  "mobileClientVersion",
			Value: "777777 3.0.0",
		},
	})

	httpClient := &http.Client{
		Transport: newCachingTransport(cleanhttp.DefaultPooledTransport(), options.ResponseCache),
		Jar:       jar,
	}

	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient = httpClient

	return &HttpTransport{
		webApiKey:   options.WebApiKey,
		client:      httpClient,
		retryClient: retryClient,
	}
}

func (c HttpTransport) CookieJar() http.CookieJar {
	return c.client.Jar
}

// Send sends a specialized HTTP Request to steam.
func (c HttpTransport) Send(ctx context.Context, request Request, response any) error {
	//rv := reflect.ValueOf(response)
	//if rv.!(rv.IsZero() || rv.IsNil()) && rv.Kind() != reflect.Pointer {
	//	return eris.Errorf("response type must be a pointer when not nil")
	//}

	httpMethod := request.Method()

	requestValues, valuesErr := request.Values()
	if valuesErr != nil {
		return valuesErr
	}

	requestUrl := request.Url()
	if !strings.HasSuffix(requestUrl, "?") {
		requestUrl += "?"
	}

	if request.RequiresApiKey() {
		if requestValues == nil {
			requestValues = make(url.Values)
		}
		requestValues.Add("key", c.webApiKey)
	}

	var httpBody io.Reader
	if requestValues != nil {
		if httpMethod == http.MethodGet {
			requestUrl += requestValues.Encode()
		} else {
			httpBody = strings.NewReader(requestValues.Encode())
		}
	}

	httpRequest, httpRequestErr := http.NewRequestWithContext(ctx, httpMethod, requestUrl, httpBody)
	if httpRequestErr != nil {
		return httpRequestErr
	}

	httpRequest.Header.Add("Accept", JsonContentType)
	httpRequest.Header.Add("User-Agent", "okhttp/3.12.12")
	if httpMethod == http.MethodPost {
		httpRequest.Header.Add("Content-Type", FormContentType)
	}

	headers, headersErr := request.Headers()
	if headersErr != nil {
		return headersErr
	}

	if headers != nil {
		for headerKey, headerValues := range headers {
			for _, headerValue := range headerValues {
				httpRequest.Header.Add(headerKey, headerValue)
			}
		}
	}

	httpClient := c.client
	if request.Retryable() {
		httpClient = c.retryClient.StandardClient()
	}

	httpResponse, httpResponseErr := httpClient.Do(httpRequest)
	if httpResponseErr != nil {
		return eris.Errorf("request to Steam failed: %v", httpResponseErr)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing steam response body: %v", err)
		}
	}(httpResponse.Body)

	if err := request.EnsureResponseSuccess(httpResponse); err != nil {
		return err
	}

	if err := steamlang.EnsureEResultResponse(httpResponse); err != nil {
		return err
	}

	if response != nil {
		responseBody, err := io.ReadAll(httpResponse.Body)
		if err != nil {
			return eris.Errorf("couldn't read request: %v", err)
		}

		err = json.Unmarshal(responseBody, response)
		if err != nil {
			return eris.Errorf("couldnt unmarshal response: %v", err)
		}
	}

	return nil
}

func (c HttpTransport) HttpClient() *http.Client {
	return c.client
}
