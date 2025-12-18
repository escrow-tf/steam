package transport

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/google/go-querystring/query"
	"github.com/rotisserie/eris"
	"google.golang.org/protobuf/proto"
)

type Transformer[T any] interface {
	Transform(input T, request *http.Request) error
}

func EmptyTransformer[T any]() Transformer[T] {
	return emptyTransformer[T]{}
}

func UrlEncodeTransformer[T any]() Transformer[T] {
	return urlEncodeTransformer[T]{}
}

func ProtoTransformer[T proto.Message]() Transformer[T] {
	return protoTransformer[T]{}
}

type emptyTransformer[T any] struct{}

func (e emptyTransformer[T]) Transform(input T, request *http.Request) error {
	return nil
}

type urlEncodeTransformer[T any] struct{}

func (u urlEncodeTransformer[T]) Transform(input T, request *http.Request) error {
	values, err := query.Values(input)
	if err != nil {
		return eris.Wrap(err, "error encoding values from type")
	}

	request.Body = io.NopCloser(strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return nil
}

type protoTransformer[T proto.Message] struct{}

func (p protoTransformer[T]) Transform(input T, request *http.Request) error {
	marshalled, err := proto.Marshal(input)
	if err != nil {
		return eris.Wrap(err, "failed to marshall proto message")
	}

	encoded := base64.StdEncoding.EncodeToString(marshalled)

	bodyBuffer := &bytes.Buffer{}
	multipartWriter := multipart.NewWriter(bodyBuffer)

	err = multipartWriter.WriteField("input_protobuf_encoded", encoded)
	if err != nil {
		_ = multipartWriter.Close()
		return eris.Wrap(err, "failed to write multipart field in transport request")
	}

	err = multipartWriter.Close()
	if err != nil {
		return eris.Wrap(err, "failed to close multipart writer in transport request")
	}

	request.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	request.Body = io.NopCloser(bodyBuffer)
	return nil
}
