package server

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/soyacen/goose"
	"google.golang.org/genproto/googleapis/api/httpbody"
	rpchttp "google.golang.org/genproto/googleapis/rpc/http"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// --- Mocks ---

type mockProto struct {
	proto.Message
}

func (m *mockProto) Reset()         {}
func (m *mockProto) String() string { return "mock" }
func (m *mockProto) ProtoMessage()  {}

// --- Tests ---

func TestEncodeResponse(t *testing.T) {
	rr := httptest.NewRecorder()
	msg := &httpbody.HttpBody{
		ContentType: "application/test",
		Data:        []byte("hello"),
	}
	opts := protojson.MarshalOptions{}
	err := EncodeResponse(context.Background(), rr, msg, opts)
	if err != nil {
		t.Fatalf("EncodeResponse error: %v", err)
	}
	resp := rr.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get(goose.ContentTypeKey); ct != goose.JsonContentType {
		t.Errorf("Content-Type = %q, want %q", ct, goose.JsonContentType)
	}
}

func TestEncodeHttpBody(t *testing.T) {
	rr := httptest.NewRecorder()
	msg := &httpbody.HttpBody{
		ContentType: "application/test",
		Data:        []byte("hello"),
	}
	err := EncodeHttpBody(context.Background(), rr, msg)
	if err != nil {
		t.Fatalf("EncodeHttpBody error: %v", err)
	}
	resp := rr.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get(goose.ContentTypeKey); ct != "application/test" {
		t.Errorf("Content-Type = %q, want application/test", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Equal(body, []byte("hello")) {
		t.Errorf("body = %q, want %q", body, "hello")
	}
}

func TestEncodeHttpResponse(t *testing.T) {
	rr := httptest.NewRecorder()
	msg := &rpchttp.HttpResponse{
		Status: 201,
		Body:   []byte("abc"),
		Headers: []*rpchttp.HttpHeader{
			{Key: "X-Foo", Value: "bar"},
			{Key: "X-Foo", Value: "baz"},
		},
	}
	err := EncodeHttpResponse(context.Background(), rr, msg)
	if err != nil {
		t.Fatalf("EncodeHttpResponse error: %v", err)
	}
	resp := rr.Result()
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Errorf("status = %d, want 201", resp.StatusCode)
	}
	if got := resp.Header["X-Foo"]; !reflect.DeepEqual(got, []string{"bar", "baz"}) {
		t.Errorf("X-Foo header = %v, want [bar baz]", got)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Equal(body, []byte("abc")) {
		t.Errorf("body = %q, want %q", body, "abc")
	}
}
