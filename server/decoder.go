package server

import (
	"context"
	"io"
	"net/http"

	"github.com/soyacen/goose"
	"google.golang.org/genproto/googleapis/api/httpbody"
	rpchttp "google.golang.org/genproto/googleapis/rpc/http"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// CustomDecodeRequest provides custom request decoding functionality
// Parameters:
//   - ctx: Context object
//   - request: HTTP request object
//   - req: Proto.Message to be decoded
//
// Returns:
//   - bool: Indicates whether custom decoding was performed
//   - error: Any error that occurred during decoding
//
// Behavior:
//  1. Checks if req implements UnmarshalRequest method
//  2. If implemented, invokes the method for decoding
//  3. If not implemented, returns false indicating no custom decoding was done
func CustomDecodeRequest(ctx context.Context, request *http.Request, req proto.Message) (bool, error) {
	unmarshaler, ok := req.(interface {
		UnmarshalRequest(context.Context, *http.Request) error
	})
	if ok {
		return true, unmarshaler.UnmarshalRequest(ctx, request)
	}
	return false, nil
}

// DecodeRequest decodes HTTP request body into a proto.Message
// Parameters:
//   - ctx: Context object
//   - request: HTTP request object
//   - req: Target proto.Message
//   - unmarshalOptions: protojson unmarshal options
//
// Returns:
//   - error: Decoding error if any
//
// Behavior:
//  1. Reads the request body
//  2. Unmarshals the data into target proto.Message using protojson
func DecodeRequest(ctx context.Context, request *http.Request, req proto.Message, unmarshalOptions protojson.UnmarshalOptions) error {
	data, err := io.ReadAll(request.Body)
	if err != nil {
		return err
	}
	if err := unmarshalOptions.Unmarshal(data, req); err != nil {
		return err
	}
	return nil
}

// DecodeHttpBody decodes HTTP request body into HttpBody object
// Parameters:
//   - ctx: Context object
//   - request: HTTP request object
//   - body: Target HttpBody object
//
// Returns:
//   - error: Decoding error if any
//
// Behavior:
//  1. Reads the request body data
//  2. Sets HttpBody's Data and ContentType fields
func DecodeHttpBody(ctx context.Context, request *http.Request, body *httpbody.HttpBody) error {
	data, err := io.ReadAll(request.Body)
	if err != nil {
		return err
	}
	body.Data = data
	body.ContentType = request.Header.Get(goose.ContentTypeKey)
	return nil
}

// DecodeHttpRequest decodes HTTP request into HttpRequest object
// Parameters:
//   - ctx: Context object
//   - request: HTTP request object
//   - request: Target HttpRequest object
//
// Returns:
//   - error: Decoding error if any
//
// Behavior:
//  1. Reads the request body data
//  2. Sets method, URI, headers and body fields
func DecodeHttpRequest(ctx context.Context, request *http.Request, req *rpchttp.HttpRequest) error {
	data, err := io.ReadAll(request.Body)
	if err != nil {
		return err
	}
	req.Method = request.Method
	req.Uri = request.URL.String()
	for key, values := range request.Header {
		for _, value := range values {
			req.Headers = append(req.Headers, &rpchttp.HttpHeader{Key: key, Value: value})
		}
	}
	req.Body = data
	return nil
}
