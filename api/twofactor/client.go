package twofactor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/escrow-tf/steam/api"
	"github.com/escrow-tf/steam/steamlang"
)

type Client struct {
	Transport api.Transport

	aligned  bool
	timeDiff time.Duration
}

func (c *Client) SteamTime() (time.Time, error) {
	if !c.aligned {
		return time.Time{}, errors.New("AlignTime must be called before SteamTime can be retrieved")
	}
	return time.Now().UTC().Add(c.timeDiff), nil
}

func (c *Client) AlignTime(ctx context.Context) error {
	unixNow := time.Now().Unix()
	timeResponse, err := c.QueryTime(ctx)
	if err != nil {
		return err
	}
	c.timeDiff = time.Second * time.Duration(timeResponse.Response.ServerTime-unixNow)
	c.aligned = true
	return nil
}

type QueryTimeRequest struct{}

func (q QueryTimeRequest) CacheTTL() time.Duration {
	return 0
}

func (q QueryTimeRequest) EnsureResponseSuccess(httpResponse *http.Response) error {
	return steamlang.EnsureSuccessResponse(httpResponse)
}

func (q QueryTimeRequest) Headers() (http.Header, error) {
	return nil, nil
}

func (q QueryTimeRequest) Retryable() bool {
	return false
}

func (q QueryTimeRequest) RequiresApiKey() bool {
	return false
}

func (q QueryTimeRequest) Method() string {
	return http.MethodPost
}

func (q QueryTimeRequest) Url() string {
	return fmt.Sprintf("%s/ITwoFactorService/QueryTime/v0001", api.BaseURL)
}

func (q QueryTimeRequest) Values() (url.Values, error) {
	return url.Values{
		"steamid": []string{"0"},
	}, nil
}

type QueryTimeResponse struct {
	Response struct {
		ServerTime int64 `json:"server_time,string"`
	} `json:"response"`
}

func (c *Client) QueryTime(ctx context.Context) (*QueryTimeResponse, error) {
	request := QueryTimeRequest{}
	var response QueryTimeResponse
	sendErr := c.Transport.Send(ctx, request, &response)
	if sendErr != nil {
		return nil, sendErr
	}
	return &response, nil
}
