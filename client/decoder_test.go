package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/soyacen/goose"
	"google.golang.org/genproto/googleapis/api/httpbody"
	rpchttp "google.golang.org/genproto/googleapis/rpc/http"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestDecodeMessage(t *testing.T) {
	// Create a test struct proto message
	testStruct, err := structpb.NewStruct(map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	})
	if err != nil {
		t.Fatalf("Failed to create test struct: %v", err)
	}

	// Marshal the test struct to JSON
	testData, err := protojson.Marshal(testStruct)
	if err != nil {
		t.Fatalf("Failed to marshal test struct: %v", err)
	}

	// Create a mock HTTP response with the test data
	response := &http.Response{
		Body: io.NopCloser(bytes.NewReader(testData)),
	}

	// Create a target struct to decode into
	targetStruct := &structpb.Struct{}

	// Test successful decoding
	err = DecodeMessage(context.Background(), response, targetStruct, protojson.UnmarshalOptions{})
	if err != nil {
		t.Errorf("DecodeMessage returned error: %v", err)
	}

	// Verify the decoded data
	if !proto.Equal(testStruct, targetStruct) {
		t.Errorf("Decoded struct does not match expected. Got: %v, Want: %v", targetStruct, testStruct)
	}

	// Test with invalid JSON data
	invalidResponse := &http.Response{
		Body: io.NopCloser(bytes.NewReader([]byte("invalid json"))),
	}

	invalidTarget := &structpb.Struct{}
	err = DecodeMessage(context.Background(), invalidResponse, invalidTarget, protojson.UnmarshalOptions{})
	if err == nil {
		t.Error("DecodeMessage should return error for invalid JSON")
	}
}

func TestDecodeHttpBody(t *testing.T) {
	// Test data
	testContentType := "application/json"
	testData := []byte(`{"test": "data"}`)

	// Create a mock HTTP response
	response := &http.Response{
		Header: http.Header{
			goose.ContentTypeKey: []string{testContentType},
		},
		Body: io.NopCloser(bytes.NewReader(testData)),
	}

	// Create target HttpBody
	target := &httpbody.HttpBody{}

	// Test successful decoding
	err := DecodeHttpBody(context.Background(), response, target)
	if err != nil {
		t.Errorf("DecodeHttpBody returned error: %v", err)
	}

	// Verify the decoded data
	if target.GetContentType() != testContentType {
		t.Errorf("ContentType mismatch. Got: %s, Want: %s", target.GetContentType(), testContentType)
	}

	if !bytes.Equal(target.GetData(), testData) {
		t.Errorf("Data mismatch. Got: %s, Want: %s", string(target.GetData()), string(testData))
	}

	// Test with unreadable body
	errorResponse := &http.Response{
		Body: &errorReader{},
	}

	err = DecodeHttpBody(context.Background(), errorResponse, &httpbody.HttpBody{})
	if err == nil {
		t.Error("DecodeHttpBody should return error for unreadable body")
	}
}

func TestDecodeHttpResponse(t *testing.T) {
	// Test data
	statusCode := 200
	reason := "OK"
	testHeaders := http.Header{
		"Content-Type":  []string{"application/json"},
		"Custom-Header": []string{"custom-value"},
	}
	testBody := []byte(`{"response": "data"}`)

	// Create a mock HTTP response
	response := &http.Response{
		StatusCode: statusCode,
		Header:     testHeaders,
		Body:       io.NopCloser(bytes.NewReader(testBody)),
	}

	// Create target HttpResponse
	target := &rpchttp.HttpResponse{}

	// Test successful decoding
	err := DecodeHttpResponse(context.Background(), response, target)
	if err != nil {
		t.Errorf("DecodeHttpResponse returned error: %v", err)
	}

	// Verify the decoded data
	if int(target.GetStatus()) != statusCode {
		t.Errorf("Status code mismatch. Got: %d, Want: %d", target.GetStatus(), statusCode)
	}

	if target.GetReason() != reason {
		t.Errorf("Reason mismatch. Got: %s, Want: %s", target.GetReason(), reason)
	}

	if !bytes.Equal(target.GetBody(), testBody) {
		t.Errorf("Body mismatch. Got: %s, Want: %s", string(target.GetBody()), string(testBody))
	}

	// Verify headers
	expectedHeaders := []*rpchttp.HttpHeader{
		{Key: "Content-Type", Value: "application/json"},
		{Key: "Custom-Header", Value: "custom-value"},
	}

	if !reflect.DeepEqual(target.GetHeaders(), expectedHeaders) {
		t.Errorf("Headers mismatch. Got: %v, Want: %v", target.GetHeaders(), expectedHeaders)
	}

	// Test with unreadable body
	errorResponse := &http.Response{
		StatusCode: statusCode,
		Body:       &errorReader{},
	}

	err = DecodeHttpResponse(context.Background(), errorResponse, &rpchttp.HttpResponse{})
	if err == nil {
		t.Error("DecodeHttpResponse should return error for unreadable body")
	}
}

// errorReader is a test helper that always returns an error when reading
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func (e *errorReader) Close() error {
	return nil
}
