package transport

import (
	"encoding/json"
	"io"

	"github.com/rotisserie/eris"
	"google.golang.org/protobuf/proto"
)

type Decoder[T any] interface {
	Decode(response io.ReadCloser, result T) error
}

func JsonDecoder[T any]() Decoder[T] {
	return jsonDecoder[T]{}
}

func ProtoDecoder[T proto.Message]() Decoder[T] {
	return protoDecoder[T]{}
}

type jsonDecoder[T any] struct{}

func (j jsonDecoder[T]) Decode(response io.ReadCloser, result T) error {
	body, err := io.ReadAll(response)
	if err != nil {
		return eris.Wrap(err, "error reading response body")
	}

	return eris.Wrap(json.Unmarshal(body, result), "error unmarshalling json response")
}

type protoDecoder[T proto.Message] struct{}

func (p protoDecoder[T]) Decode(response io.ReadCloser, result T) error {
	body, err := io.ReadAll(response)
	if err != nil {
		return eris.Wrap(err, "error reading response body")
	}

	return eris.Wrap(proto.Unmarshal(body, result), "error unmarshalling protobuf response")
}
