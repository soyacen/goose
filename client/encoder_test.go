package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/soyacen/goose"
	"google.golang.org/genproto/googleapis/api/httpbody"
	rpchttp "google.golang.org/genproto/googleapis/rpc/http"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestEncodeMessage(t *testing.T) {
	// Create a test struct proto message
	testStruct, err := structpb.NewStruct(map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	})
	if err != nil {
		t.Fatalf("Failed to create test struct: %v", err)
	}

	// Expected JSON output
	expectedData, err := protojson.MarshalOptions{}.Marshal(testStruct)
	if err != nil {
		t.Fatalf("Failed to marshal test struct: %v", err)
	}

	// Test successful encoding
	header := http.Header{}
	body := &bytes.Buffer{}
	marshalOptions := protojson.MarshalOptions{}

	err = EncodeMessage(context.Background(), testStruct, header, body, marshalOptions)
	if err != nil {
		t.Errorf("EncodeMessage returned error: %v", err)
	}

	// Verify content type header
	if contentType := header.Get(goose.ContentTypeKey); contentType != goose.JsonContentType {
		t.Errorf("Content type header mismatch. Got: %s, Want: %s", contentType, goose.JsonContentType)
	}

	// Verify body content
	if !bytes.Equal(body.Bytes(), expectedData) {
		t.Errorf("Body content mismatch. Got: %s, Want: %s", body.String(), string(expectedData))
	}

	// Test with writer that returns error
	errorWriter := &errorWriter{}
	err = EncodeMessage(context.Background(), testStruct, http.Header{}, errorWriter, protojson.MarshalOptions{})
	if err == nil {
		t.Error("EncodeMessage should return error when writer fails")
	}
}

func TestEncodeHttpBody(t *testing.T) {
	// Test data
	testContentType := "application/json"
	testData := []byte(`{"test": "data"}`)

	// Create HttpBody message
	httpBody := &httpbody.HttpBody{
		ContentType: testContentType,
		Data:        testData,
	}

	// Test successful encoding
	header := http.Header{}
	body := &bytes.Buffer{}

	err := EncodeHttpBody(context.Background(), httpBody, header, body)
	if err != nil {
		t.Errorf("EncodeHttpBody returned error: %v", err)
	}

	// Verify content type header
	if contentType := header.Get(goose.ContentTypeKey); contentType != testContentType {
		t.Errorf("Content type header mismatch. Got: %s, Want: %s", contentType, testContentType)
	}

	// Verify body content
	if !bytes.Equal(body.Bytes(), testData) {
		t.Errorf("Body content mismatch. Got: %s, Want: %s", body.String(), string(testData))
	}

	// Test with writer that returns error
	errorWriter := &errorWriter{}
	err = EncodeHttpBody(context.Background(), httpBody, http.Header{}, errorWriter)
	if err == nil {
		t.Error("EncodeHttpBody should return error when writer fails")
	}
}

func TestEncodeHttpRequest(t *testing.T) {
	// Test data
	testBody := []byte(`{"request": "data"}`)
	testHeaders := []*rpchttp.HttpHeader{
		{Key: "Content-Type", Value: "application/json"},
		{Key: "Custom-Header", Value: "custom-value"},
	}

	// Create HttpRequest message
	httpRequest := &rpchttp.HttpRequest{
		Body:    testBody,
		Headers: testHeaders,
	}

	// Test successful encoding
	header := http.Header{}
	body := &bytes.Buffer{}

	err := EncodeHttpRequest(context.Background(), httpRequest, header, body)
	if err != nil {
		t.Errorf("EncodeHttpRequest returned error: %v", err)
	}

	// Verify headers
	if contentType := header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type header mismatch. Got: %s, Want: application/json", contentType)
	}

	if customHeader := header.Get("Custom-Header"); customHeader != "custom-value" {
		t.Errorf("Custom-Header mismatch. Got: %s, Want: custom-value", customHeader)
	}

	// Verify body content
	if !bytes.Equal(body.Bytes(), testBody) {
		t.Errorf("Body content mismatch. Got: %s, Want: %s", body.String(), string(testBody))
	}

	// Test with writer that returns error
	errorWriter := &errorWriter{}
	err = EncodeHttpRequest(context.Background(), httpRequest, http.Header{}, errorWriter)
	if err == nil {
		t.Error("EncodeHttpRequest should return error when writer fails")
	}
}

// errorWriter is a test helper that always returns an error when writing
type errorWriter struct{}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}
