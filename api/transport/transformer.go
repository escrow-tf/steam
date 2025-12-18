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

type Transformer interface {
	Transform(request *http.Request) error
}

type EmptyTransformer struct{}

func (e *EmptyTransformer) Transform(request *http.Request) error {
	return nil
}

type UrlEncodeTransformer[T any] struct {
	Value T
}

func (u *UrlEncodeTransformer[T]) Transform(request *http.Request) error {
	values, err := query.Values(u.Value)
	if err != nil {
		return eris.Wrap(err, "error encoding values from type")
	}

	request.Body = io.NopCloser(strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return nil
}

type ProtoTransformer[T proto.Message] struct {
	Value T
}

func (p *ProtoTransformer[T]) Transform(request *http.Request) error {
	marshalled, err := proto.Marshal(p.Value)
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
