package server

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/soyacen/goose"
	"google.golang.org/genproto/googleapis/api/httpbody"
	rpchttp "google.golang.org/genproto/googleapis/rpc/http"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// --- Mocks ---

type mockProto_Decoder struct {
	proto.Message
	Data string `json:"data"`
}

func (m *mockProto_Decoder) Reset()         {}
func (m *mockProto_Decoder) String() string { return m.Data }
func (m *mockProto_Decoder) ProtoMessage()  {}

func (m *mockProto_Decoder) MarshalJSON() ([]byte, error) {
	return []byte(`{"data":"` + m.Data + `"}`), nil
}

func (m *mockProto_Decoder) UnmarshalJSON(b []byte) error {
	m.Data = string(bytes.Trim(b, `{"data":}`))
	return nil
}

func TestDecodeRequest(t *testing.T) {
	msg := &httpbody.HttpBody{}
	body := `{"content_type":"json"}`
	r := &http.Request{
		Body: io.NopCloser(strings.NewReader(body)),
	}
	opts := protojson.UnmarshalOptions{}
	err := DecodeRequest(context.Background(), r, msg, opts)
	if err != nil {
		t.Fatalf("DecodeRequest error: %v", err)
	}
	if msg.ContentType != `"json"` && msg.ContentType != "json" {
		t.Errorf("msg.Data = %q, want \"hello\"", msg.Data)
	}
}

func TestDecodeHttpBody(t *testing.T) {
	data := "abc"
	r := &http.Request{
		Body:   io.NopCloser(strings.NewReader(data)),
		Header: http.Header{goose.ContentTypeKey: []string{"application/test"}},
	}
	body := &httpbody.HttpBody{}
	err := DecodeHttpBody(context.Background(), r, body)
	if err != nil {
		t.Fatalf("DecodeHttpBody error: %v", err)
	}
	if string(body.Data) != data {
		t.Errorf("body.Data = %q, want %q", body.Data, data)
	}
	if body.ContentType != "application/test" {
		t.Errorf("body.ContentType = %q, want application/test", body.ContentType)
	}
}

func TestDecodeHttpRequest(t *testing.T) {
	r := &http.Request{
		Method: "POST",
		URL:    &url.URL{Scheme: "http", Host: "localhost", Path: "/foo"},
		Body:   io.NopCloser(strings.NewReader("xyz")),
		Header: http.Header{"X-Foo": []string{"bar", "baz"}},
	}
	req := &rpchttp.HttpRequest{}
	err := DecodeHttpRequest(context.Background(), r, req)
	if err != nil {
		t.Fatalf("DecodeHttpRequest error: %v", err)
	}
	if req.Method != "POST" {
		t.Errorf("req.Method = %q, want POST", req.Method)
	}
	if req.Uri != "http://localhost/foo" {
		t.Errorf("req.Uri = %q, want http://localhost/foo", req.Uri)
	}
	if string(req.Body) != "xyz" {
		t.Errorf("req.Body = %q, want xyz", req.Body)
	}
	var foundBar, foundBaz bool
	for _, h := range req.Headers {
		if h.Key == "X-Foo" && h.Value == "bar" {
			foundBar = true
		}
		if h.Key == "X-Foo" && h.Value == "baz" {
			foundBaz = true
		}
	}
	if !foundBar || !foundBaz {
		t.Errorf("req.Headers missing expected values: %v", req.Headers)
	}
}
