package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/escrow-tf/steam/steamlang"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rotisserie/eris"
	"google.golang.org/protobuf/proto"
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
	OldValues() (url.Values, error)
	Values() (interface{}, error)
	Headers() (http.Header, error)
	EnsureResponseSuccess(httpResponse *http.Response) error
}

type Transport interface {
	CookieJar() http.CookieJar
	Send(ctx context.Context, request Request, response any) error
	HttpClient() *http.Client
}

type HttpTransport struct {
	webApiKey     string
	client        *http.Client
	retryClient   *retryablehttp.Client
	dumpRequests  bool
	dumpResponses bool
}

type HttpTransportOptions struct {
	WebApiKey     string
	ResponseCache CacheAdaptor
	DumpRequests  bool
	DumpResponses bool
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
			Value: "777777 3.10.3",
		},
	})

	var httpTransport http.RoundTripper = cleanhttp.DefaultPooledTransport()
	if options.ResponseCache != nil {
		httpTransport = newCachingTransport(httpTransport, options.ResponseCache)
	}

	httpClient := &http.Client{
		Transport: httpTransport,
		Jar:       jar,
	}

	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient = httpClient

	return &HttpTransport{
		webApiKey:     options.WebApiKey,
		client:        httpClient,
		retryClient:   retryClient,
		dumpRequests:  options.DumpRequests,
		dumpResponses: options.DumpResponses,
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

	requestValues, valuesErr := request.OldValues()
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

	contentType := FormContentType
	var httpBody io.Reader
	if requestValues != nil {
		if httpMethod == http.MethodGet {
			requestUrl += requestValues.Encode()
		} else {
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			defer func(writer *multipart.Writer) {
				_ = writer.Close()
			}(writer)

			for key, val := range requestValues {
				fieldWriter, fieldErr := writer.CreateFormField(key)
				if fieldErr != nil {
					return fieldErr
				}

				if _, writeErr := fieldWriter.Write([]byte(val[0])); writeErr != nil {
					panic(writeErr)
				}

				for idx, item := range val {
					if idx == 0 {
						continue
					}

					if _, writeErr := fieldWriter.Write([]byte(";")); writeErr != nil {
						panic(writeErr)
					}

					if _, writeErr := fieldWriter.Write([]byte(item)); writeErr != nil {
						panic(writeErr)
					}
				}
			}

			contentType = writer.FormDataContentType()
			httpBody = bytes.NewReader(body.Bytes())
		}
	}

	httpRequest, httpRequestErr := http.NewRequestWithContext(ctx, httpMethod, requestUrl, httpBody)
	if httpRequestErr != nil {
		return httpRequestErr
	}

	httpRequest.Header.Add("Accept", JsonContentType)
	httpRequest.Header.Add("User-Agent", "okhttp/4.9.2")
	if httpMethod == http.MethodPost {
		httpRequest.Header.Add("Content-Type", contentType)
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

	if c.dumpRequests {
		dump, dumpErr := httputil.DumpRequest(httpRequest, true)
		if dumpErr == nil {
			log.Println(string(dump))
		}
	}

	httpResponse, httpResponseErr := httpClient.Do(httpRequest)
	if httpResponseErr != nil {
		return eris.Errorf("request to Steam failed: %v", httpResponseErr)
	}

	if c.dumpResponses {
		dump, dumpErr := httputil.DumpResponse(httpResponse, true)
		if dumpErr == nil {
			log.Println(string(dump))
		}
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

		if strings.Contains(httpResponse.Header.Get("Content-Type"), JsonContentType) {
			err = json.Unmarshal(responseBody, response)
			if err != nil {
				return eris.Errorf("couldnt unmarshal response: %v", err)
			}
		} else {
			responseMessage, isMessage := response.(proto.Message)
			if !isMessage {
				return eris.New("http response is not json, but response parameter is not proto.Message")
			}
			err = proto.Unmarshal(responseBody, responseMessage)
			if err != nil {
				return eris.Errorf("couldnt unmarshal response: %v", err)
			}
		}
	}

	return nil
}

func (c HttpTransport) HttpClient() *http.Client {
	return c.client
}
