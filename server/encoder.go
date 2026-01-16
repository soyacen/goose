package server

import (
	"context"
	"net/http"

	"github.com/soyacen/goose"
	"google.golang.org/genproto/googleapis/api/httpbody"
	rpchttp "google.golang.org/genproto/googleapis/rpc/http"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// EncodeResponse encodes a protobuf message as JSON into an HTTP response.
// Sets Content-Type to application/json and status code to 200 OK.
//
// Parameters:
//
//	ctx - context.Context for the request
//	response - http.ResponseWriter to write the response
//	resp - proto.Message to encode
//	marshalOptions - protojson.MarshalOptions for JSON encoding
//
// Returns:
//
//	error - if encoding or writing fails
func EncodeResponse(ctx context.Context, response http.ResponseWriter, resp proto.Message, marshalOptions protojson.MarshalOptions) error {
	// Set response headers for JSON content and HTTP 200 status
	response.Header().Set(goose.ContentTypeKey, goose.JsonContentType)
	response.WriteHeader(http.StatusOK)

	// Marshal the protocol buffer message into JSON
	data, err := marshalOptions.Marshal(resp)
	if err != nil {
		return err
	}

	// Write the JSON data to the response body
	if _, err := response.Write(data); err != nil {
		return err
	}

	return nil
}

// EncodeHttpBody encodes an httpbody.HttpBody into an HTTP response.
// Sets Content-Type from the HttpBody and status code to 200 OK.
//
// Parameters:
//
//	ctx - context.Context for the request
//	response - http.ResponseWriter to write the response
//	resp - *httpbody.HttpBody to encode
//
// Returns:
//
//	error - if writing fails
func EncodeHttpBody(ctx context.Context, response http.ResponseWriter, resp *httpbody.HttpBody) error {
	// Set response headers
	response.Header().Set(goose.ContentTypeKey, resp.GetContentType())
	response.WriteHeader(http.StatusOK)

	// Write response data
	if _, err := response.Write(resp.GetData()); err != nil {
		return err
	}
	return nil
}

// EncodeHttpResponse encodes an rpchttp.HttpResponse into an HTTP response.
// Sets headers, status code and body from the HttpResponse.
//
// Parameters:
//
//	ctx - context.Context for the request
//	response - http.ResponseWriter to write the response
//	resp - *rpchttp.HttpResponse to encode
//
// Returns:
//
//	error - if writing fails
func EncodeHttpResponse(ctx context.Context, response http.ResponseWriter, resp *rpchttp.HttpResponse) error {
	header := response.Header()
	// Set all headers from the RPC response
	for _, httpHeader := range resp.GetHeaders() {
		header.Add(httpHeader.GetKey(), httpHeader.GetValue())
	}

	// Write HTTP status code before body
	response.WriteHeader(int(resp.GetStatus()))

	// Write response body and return any write errors
	if _, err := response.Write(resp.GetBody()); err != nil {
		return err
	}
	return nil
}
