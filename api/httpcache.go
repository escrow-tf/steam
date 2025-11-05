package api

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

type CacheAdaptor interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
}

type cachingTransport struct {
	next     http.RoundTripper
	cacheKey func(*http.Request) string
	cache    CacheAdaptor
}

func (c *cachingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	// only cache idempotent requests
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		return c.next.RoundTrip(request)
	}

	ctx := request.Context()

	ttl, ttlOk := ctx.Value("steam-cachingTransport-cache-ttl").(time.Duration)
	if !ttlOk || ttl == 0 {
		return c.next.RoundTrip(request)
	}

	requestKey := c.cacheKey(request)
	cachedResponse, cacheErr := c.cache.Get(ctx, requestKey)
	if cacheErr != nil {
		// TODO: log this error?
	} else {
		reader := bufio.NewReader(strings.NewReader(cachedResponse))

		response, readErr := http.ReadResponse(reader, request)
		if readErr != nil {
			// TODO: log this error?
		} else {
			return response, nil
		}
	}

	response, err := c.next.RoundTrip(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return response, nil
	}

	if err := c.cacheResponse(ctx, requestKey, response, ttl); err != nil {
		// TODO: log this error?
	}

	return response, nil
}

func (c *cachingTransport) cacheResponse(
	ctx context.Context,
	key string,
	response *http.Response,
	ttl time.Duration,
) error {
	responseDump, dumpErr := httputil.DumpResponse(response, true)
	if dumpErr != nil {
		return dumpErr
	}

	if err := c.cache.Set(ctx, key, string(responseDump), ttl); err != nil {
		return err
	}

	return nil
}

func ContextWithCachingTtl(ctx context.Context, ttl time.Duration) context.Context {
	return context.WithValue(ctx, "steam-cachingTransport-cache-ttl", ttl)
}

func newCachingTransport(next http.RoundTripper, cache CacheAdaptor) http.RoundTripper {
	return &cachingTransport{
		next:     next,
		cacheKey: func(request *http.Request) string { return request.URL.String() },
		cache:    cache,
	}
}
