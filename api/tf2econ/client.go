package tf2econ

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/escrow-tf/steam/api"
	"github.com/escrow-tf/steam/steamid"
	"github.com/escrow-tf/steam/steamlang"
)

const WebApiBaseUrl = "https://api.steampowered.com"

type Client struct {
	Transport api.Transport
}

type PlayerItemsRequest struct {
	steamId steamid.SteamID
}

func (p PlayerItemsRequest) CacheTTL() time.Duration {
	return 0
}

func (p PlayerItemsRequest) EnsureResponseSuccess(httpResponse *http.Response) error {
	return steamlang.EnsureSuccessResponse(httpResponse)
}

func (p PlayerItemsRequest) Headers() (http.Header, error) {
	return nil, nil
}

func (p PlayerItemsRequest) Retryable() bool {
	return true
}

func (p PlayerItemsRequest) RequiresApiKey() bool {
	return true
}

func (p PlayerItemsRequest) Method() string {
	return http.MethodGet
}

func (p PlayerItemsRequest) Url() string {
	return fmt.Sprintf("%s/IEconItems_440/GetPlayerItems/v1/", WebApiBaseUrl)
}

func (p PlayerItemsRequest) OldValues() (url.Values, error) {
	return url.Values{
		"steamid": []string{p.steamId.String()},
	}, nil
}

func (p PlayerItemsRequest) Values() (interface{}, error) {
	return url.Values{
		"steamid": []string{p.steamId.String()},
	}, nil
}

func (client *Client) GetPlayerItems(ctx context.Context, steamId steamid.SteamID) (*PlayerItemsResponse, error) {
	request := PlayerItemsRequest{
		steamId: steamId,
	}
	var response PlayerItemsResponse
	sendErr := client.Transport.Send(ctx, request, &response)
	if sendErr != nil {
		return nil, sendErr
	}
	return &response, nil
}
