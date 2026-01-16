package client

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/soyacen/goose"
	"google.golang.org/genproto/googleapis/api/httpbody"
	rpchttp "google.golang.org/genproto/googleapis/rpc/http"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// DecodeMessage decodes an HTTP response into a protobuf message.
// It reads the response body, unmarshals the JSON data into the provided protobuf message,
// and properly closes the response body.
//
// Parameters:
//   - ctx: The context.Context for the request
//   - response: The HTTP response to decode
//   - resp: The protobuf message to unmarshal the response data into
//   - unmarshalOptions: Options for protobuf JSON unmarshaling
//
// Returns:
//   - error: Any error that occurred during decoding, or nil if successful
func DecodeMessage(ctx context.Context, response *http.Response, resp proto.Message, unmarshalOptions protojson.UnmarshalOptions) error {
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return errors.Join(err, response.Body.Close())
	}
	if err := unmarshalOptions.Unmarshal(data, resp); err != nil {
		return errors.Join(err, response.Body.Close())
	}
	return response.Body.Close()
}

// DecodeHttpBody decodes an HTTP response into an HttpBody message.
// It extracts the content type from the response headers and copies the raw response body data.
//
// Parameters:
//   - ctx: The context.Context for the request
//   - response: The HTTP response to decode
//   - resp: The HttpBody message to populate with response data
//
// Returns:
//   - error: Any error that occurred during decoding, or nil if successful
func DecodeHttpBody(ctx context.Context, response *http.Response, resp *httpbody.HttpBody) error {
	resp.ContentType = response.Header.Get(goose.ContentTypeKey)
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	resp.Data = body
	return response.Body.Close()
}

// DecodeHttpResponse decodes an HTTP response into an HttpResponse message.
// It extracts the status code, reason phrase, headers, and body from the HTTP response.
//
// Parameters:
//   - ctx: The context.Context for the request
//   - response: The HTTP response to decode
//   - resp: The HttpResponse message to populate with response data
//
// Returns:
//   - error: Any error that occurred during decoding, or nil if successful
func DecodeHttpResponse(ctx context.Context, response *http.Response, resp *rpchttp.HttpResponse) error {
	resp.Status = int32(response.StatusCode)
	resp.Reason = http.StatusText(response.StatusCode)
	resp.Headers = make([]*rpchttp.HttpHeader, 0, len(response.Header))
	for key, values := range response.Header {
		for _, value := range values {
			elems := &rpchttp.HttpHeader{
				Key:   key,
				Value: value,
			}
			resp.Headers = append(resp.Headers, elems)
		}
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return errors.Join(err, response.Body.Close())
	}
	resp.Body = data
	return response.Body.Close()
}
