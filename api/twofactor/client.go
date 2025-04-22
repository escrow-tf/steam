package twofactor

import (
	"errors"
	"fmt"
	"github.com/escrow-tf/steam/api"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	aligned   bool
	timeDiff  time.Duration
	transport *api.Transport
}

func NewClient(transport *api.Transport) *Client {
	return &Client{
		aligned:   false,
		timeDiff:  0,
		transport: transport,
	}
}

func (c *Client) SteamTime() (time.Time, error) {
	if !c.aligned {
		return time.Time{}, errors.New("AlignTime must be called before SteamTime can be retrieved")
	}
	return time.Now().UTC().Add(c.timeDiff), nil
}

func (c *Client) AlignTime() error {
	unixNow := time.Now().Unix()
	timeResponse, err := c.QueryTime()
	if err != nil {
		return err
	}
	c.timeDiff = time.Second * time.Duration(timeResponse.Response.ServerTime-unixNow)
	c.aligned = true
	return nil
}

type QueryTimeRequest struct{}

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

func (c *Client) QueryTime() (*QueryTimeResponse, error) {
	request := QueryTimeRequest{}
	var response QueryTimeResponse
	sendErr := c.transport.Send(request, &response)
	if sendErr != nil {
		return nil, sendErr
	}
	return &response, nil
}
