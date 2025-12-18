package transport

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/escrow-tf/steam/steamlang"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rotisserie/eris"
	"google.golang.org/protobuf/proto"
)

type PrivateTransport struct {
	client      *http.Client
	retryClient *retryablehttp.Client
}

type PrivateTransportRequest[I Transformer, O any] struct {
	CanRetry bool
	BaseUrl  url.URL
	Path     string
	Headers  http.Header
	Params   url.Values
	Method   string
	Body     I
}

func SendReq[I Transformer, O any](
	ctx context.Context,
	transport PrivateTransport,
	request PrivateTransportRequest[I, O],
) (*http.Response, error) {
	requestUrl := request.BaseUrl.JoinPath(request.Path).String() + "?"

	if request.Params != nil {
		requestUrl += request.Params.Encode()
	}

	httpClient := transport.client
	if request.CanRetry {
		httpClient = transport.retryClient.StandardClient()
	}

	httpRequest, err := http.NewRequestWithContext(ctx, request.Method, requestUrl, nil)
	if err != nil {
		return nil, eris.Wrap(err, "error creating new request")
	}

	httpRequest.Header.Set("Accept", "application/json, text/plain, */*")
	httpRequest.Header.Set("User-Agent", "okhttp/4.9.2")
	for header, values := range request.Headers {
		value := httpRequest.Header.Get(header)
		if len(value) > 0 {
			value += ", "
		}

		value += strings.Join(values, ", ")

		httpRequest.Header.Set(header, value)
	}

	if err = request.Body.Transform(httpRequest); err != nil {
		return nil, err
	}

	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, eris.Wrap(err, "error performing request")
	}

	if err = steamlang.EnsureSuccessResponse(httpResponse); err != nil {
		return nil, err
	}

	if err = steamlang.EnsureEResultResponse(httpResponse); err != nil {
		return nil, err
	}

	return httpResponse, nil
}

func SendProto[I Transformer, O proto.Message](
	ctx context.Context,
	transport PrivateTransport,
	request PrivateTransportRequest[I, O],
	response O,
) error {
	httpResponse, sendErr := SendReq(ctx, transport, request)
	if sendErr != nil {
		return sendErr
	}

	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return eris.Wrap(err, "couldn't read request")
	}

	if err = proto.Unmarshal(responseBody, response); err != nil {
		return eris.Wrap(err, "error unmarshalling protobuf response body")
	}

	return nil
}

func SendJson[I Transformer, O any](
	ctx context.Context,
	transport PrivateTransport,
	request PrivateTransportRequest[I, O],
	response O,
) error {
	httpResponse, sendErr := SendReq(ctx, transport, request)
	if sendErr != nil {
		return sendErr
	}

	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return eris.Wrap(err, "couldn't read request")
	}

	if err = json.Unmarshal(responseBody, response); err != nil {
		return eris.Wrap(err, "error unmarshalling json response body")
	}

	return nil
}

type PublicTransportRequest[R any] struct {
	CanRetry       bool
	RequiresApiKey bool
	Params         url.Values
	BaseUrl        url.URL
	Path           string
}

type PublicTransport struct {
	webApiKey     string
	client        *http.Client
	retryClient   *retryablehttp.Client
	dumpRequests  bool
	dumpResponses bool
}

func Send[R any](
	ctx context.Context,
	transport PublicTransport,
	request PublicTransportRequest[R],
) (response *R, err error) {
	requestUrl := request.BaseUrl.JoinPath(request.Path).String() + "?"

	if request.RequiresApiKey {
		requestUrl += "key=" + url.QueryEscape(transport.webApiKey)
	}

	if request.Params != nil {
		requestUrl += request.Params.Encode()
	}

	httpRequest, httpRequestErr := http.NewRequestWithContext(ctx, http.MethodGet, requestUrl, nil)
	if httpRequestErr != nil {
		return nil, eris.Wrap(httpRequestErr, "error creating new request")
	}

	httpRequest.Header.Add("Accept", "application/json, text/plain, */*")
	httpRequest.Header.Add("User-Agent", "okhttp/4.9.2")

	httpClient := transport.client
	if request.CanRetry {
		httpClient = transport.retryClient.StandardClient()
	}

	if transport.dumpRequests {
		dump, dumpErr := httputil.DumpRequest(httpRequest, true)
		if dumpErr == nil {
			log.Println(string(dump))
		}
	}

	httpResponse, httpResponseErr := httpClient.Do(httpRequest)
	if httpResponseErr != nil {
		return nil, eris.Wrap(httpResponseErr, "error performing request")
	}

	if transport.dumpResponses {
		dump, dumpErr := httputil.DumpResponse(httpResponse, true)
		if dumpErr == nil {
			log.Println(string(dump))
		}
	}

	defer func(Body io.ReadCloser) {
		closeErr := Body.Close()
		if closeErr != nil {
			log.Printf("Error closing steam response body: %v", closeErr)
		}
	}(httpResponse.Body)

	if err = steamlang.EnsureSuccessResponse(httpResponse); err != nil {
		return nil, err
	}

	if err = steamlang.EnsureEResultResponse(httpResponse); err != nil {
		return nil, err
	}

	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, eris.Wrapf(err, "error reading response body")
	}

	var marshalResponse R
	err = json.Unmarshal(responseBody, &marshalResponse)
	if err != nil {
		return nil, eris.Wrapf(err, "error unmarshaling response body")
	}

	return &marshalResponse, nil
}
