package client

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

// EncodeMessage encodes a protobuf message into an HTTP request.
// It marshals the protobuf message to JSON, writes it to the body writer,
// and sets the appropriate content type header.
//
// Parameters:
//   - ctx: The context.Context for the request
//   - req: The protobuf message to encode
//   - header: The HTTP headers to set content type information
//   - body: The io.Writer to write the encoded message data
//   - marshalOptions: Options for protobuf JSON marshaling
//
// Returns:
//   - error: Any error that occurred during encoding, or nil if successful
func EncodeMessage(ctx context.Context, req proto.Message, header http.Header, body io.Writer, marshalOptions protojson.MarshalOptions) error {
	data, err := marshalOptions.Marshal(req)
	if err != nil {
		return err
	}
	if _, err = body.Write(data); err != nil {
		return err
	}
	header.Set(goose.ContentTypeKey, goose.JsonContentType)
	return nil
}

// EncodeHttpBody encodes an HttpBody message into an HTTP request.
// It writes the raw data from the HttpBody to the body writer
// and sets the content type header from the HttpBody.
//
// Parameters:
//   - ctx: The context.Context for the request
//   - req: The HttpBody message to encode
//   - header: The HTTP headers to set content type information
//   - body: The io.Writer to write the encoded message data
//
// Returns:
//   - error: Any error that occurred during encoding, or nil if successful
func EncodeHttpBody(ctx context.Context, req *httpbody.HttpBody, header http.Header, body io.Writer) error {
	if _, err := body.Write(req.GetData()); err != nil {
		return err
	}
	header.Set(goose.ContentTypeKey, req.GetContentType())
	return nil
}

// EncodeHttpRequest encodes an HttpRequest message into an HTTP request.
// It writes the body data from the HttpRequest to the body writer
// and adds all headers from the HttpRequest to the header collection.
//
// Parameters:
//   - ctx: The context.Context for the request
//   - req: The HttpRequest message to encode
//   - header: The HTTP headers to add header information from the HttpRequest
//   - body: The io.Writer to write the encoded message data
//
// Returns:
//   - error: Any error that occurred during encoding, or nil if successful
func EncodeHttpRequest(ctx context.Context, req *rpchttp.HttpRequest, header http.Header, body io.Writer) error {
	if _, err := body.Write(req.GetBody()); err != nil {
		return err
	}
	for _, httpHeader := range req.GetHeaders() {
		header.Add(httpHeader.GetKey(), httpHeader.GetValue())
	}
	return nil
}
