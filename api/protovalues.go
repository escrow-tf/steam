package api

import (
	"encoding/base64"
	"net/url"

	"google.golang.org/protobuf/proto"
)

func MarshalSteamEncodedValues(m proto.Message) (url.Values, error) {
	marshalled, err := proto.Marshal(m)
	if err != nil {
		return nil, err
	}

	return url.Values{
		"input_protobuf_encoded": []string{base64.StdEncoding.EncodeToString(marshalled)},
	}, nil
}
