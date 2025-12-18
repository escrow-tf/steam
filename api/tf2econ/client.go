package tf2econ

import (
	"net/url"

	"github.com/escrow-tf/steam/api"
	"github.com/escrow-tf/steam/api/transport"
	"github.com/escrow-tf/steam/steamid"
)

const WebApiBaseUrl = "https://api.steampowered.com"

var WebApiBase = url.URL{
	Scheme: "https",
	Host:   "api.steampowered.com",
}

type Client struct {
	Transport api.Transport
}

type PlayerItemsRequest struct {
	SteamID steamid.SteamID
}

func (p PlayerItemsRequest) IntoTransportRequest() (transport.PublicTransportRequest[PlayerItemsResponse], error) {
	request := transport.PublicTransportRequest[PlayerItemsResponse]{
		CanRetry:       true,
		RequiresApiKey: true,
		Params:         url.Values{"steamid": []string{p.SteamID.String()}},
		BaseUrl:        WebApiBase,
		Path:           "/IEconItems_440/GetPlayerItems/v1/",
	}

	return request, nil
}

// func TestTransport() {
// 	steamId, _ := steamid.ParseSteamID64("")
// 	request := PlayerItemsRequest{steamId: steamId}
// 	transportRequest, _ := request.IntoTransportRequest()

// 	publicTransport := transport.PublicTransport{}
// 	response, err := transport.Send(context.TODO(), publicTransport, transportRequest)
// 	if err != nil {

// 	}

// 	if response != nil {

// 	}
// }
