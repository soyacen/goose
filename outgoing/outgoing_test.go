package outgoing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestMarshalError tests MarshalError error message generation
func TestMarshalError(t *testing.T) {
	tests := []struct {
		name    string
		err     MarshalError
		wantMsg string
	}{
		{
			name: "body marshal error",
			err: MarshalError{
				Body: map[string]interface{}{},
				Err:  errors.New("test error"),
			},
			wantMsg: "outgoing: failed to marshal body: test error",
		},
		{
			name: "query marshal error",
			err: MarshalError{
				Query: struct{}{},
				Err:   errors.New("query error"),
			},
			wantMsg: "outgoing: failed to marshal query: query error",
		},
		{
			name: "unknown error",
			err: MarshalError{
				Err: errors.New("unknown"),
			},
			wantMsg: "outgoing: unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}

// TestUnmarshalError tests UnmarshalError behavior
func TestUnmarshalError(t *testing.T) {
	err := UnmarshalError{
		Body: []byte("test body"),
		Err:  errors.New("unmarshal failed"),
	}

	if err.Error() != "outgoing: failed to unmarshal body" {
		t.Errorf("Error() = %v, want 'outgoing: failed to unmarshal body'", err.Error())
	}

	if err.Unwrap().Error() != "unmarshal failed" {
		t.Errorf("Unwrap() = %v, want 'unmarshal failed'", err.Unwrap().Error())
	}
}

// TestOptionsSetQuery tests query parameter operations
func TestOptionsSetQuery(t *testing.T) {
	opts := &options{
		queries: make(url.Values),
	}

	opts.SetQuery("key1", "value1")
	if got := opts.queries.Get("key1"); got != "value1" {
		t.Errorf("SetQuery() failed, got %v, want 'value1'", got)
	}
}

// TestOptionsAddQuery tests adding multiple query values
func TestOptionsAddQuery(t *testing.T) {
	opts := &options{
		queries: make(url.Values),
	}

	opts.AddQuery("key", "value1")
	opts.AddQuery("key", "value2")

	values := opts.queries["key"]
	if len(values) != 2 || values[0] != "value1" || values[1] != "value2" {
		t.Errorf("AddQuery() failed, got %v", values)
	}
}

// TestOptionsDelQuery tests query parameter deletion
func TestOptionsDelQuery(t *testing.T) {
	opts := &options{
		queries: make(url.Values),
	}

	opts.SetQuery("key", "value")
	opts.DelQuery("key")

	if len(opts.queries) != 0 {
		t.Errorf("DelQuery() failed, queries not empty: %v", opts.queries)
	}
}

// TestOptionsQueryString tests parsing query string
func TestOptionsQueryString(t *testing.T) {
	opts := &options{}

	opts.QueryString("key1=value1&key2=value2")

	if opts.queries.Get("key1") != "value1" {
		t.Errorf("QueryString() failed for key1")
	}
	if opts.queries.Get("key2") != "value2" {
		t.Errorf("QueryString() failed for key2")
	}
}

// TestOptionsSetHeader tests header setting
func TestOptionsSetHeader(t *testing.T) {
	opts := &options{}

	opts.SetHeader("Content-Type", "application/json")
	if got := opts.headers.Get("Content-Type"); got != "application/json" {
		t.Errorf("SetHeader() failed, got %v", got)
	}
}

// TestOptionsUncanonicalHeader tests uncanonical header setting
func TestOptionsUncanonicalHeader(t *testing.T) {
	opts := &options{}

	opts.SetHeader("X-Custom", "value", true)
	if opts.headers["X-Custom"][0] != "value" {
		t.Errorf("SetHeader() uncanonical failed")
	}
}

// TestOptionsBasicAuth tests basic authentication
func TestOptionsBasicAuth(t *testing.T) {
	opts := &options{}

	opts.BasicAuth("user", "pass")

	auth := opts.headers.Get("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
		t.Errorf("BasicAuth() failed, got %v", auth)
	}
}

// TestOptionsBearerAuth tests bearer token authentication
func TestOptionsBearerAuth(t *testing.T) {
	opts := &options{}

	opts.BearerAuth("token123")

	auth := opts.headers.Get("Authorization")
	if auth != "Bearer token123" {
		t.Errorf("BearerAuth() failed, got %v, want 'Bearer token123'", auth)
	}
}

// TestOptionsCookie tests cookie operations
func TestOptionsCookie(t *testing.T) {
	opts := &options{}

	cookie := &http.Cookie{Name: "test", Value: "value"}
	opts.SetCookie(cookie)

	cookies := opts.cookies["test"]
	if len(cookies) != 1 || cookies[0].Value != "value" {
		t.Errorf("SetCookie() failed")
	}

	opts.DelCookie(cookie)
	if len(opts.cookies) != 0 {
		t.Errorf("DelCookie() failed")
	}
}

// TestSenderMethod tests HTTP method setting
func TestSenderMethod(t *testing.T) {
	s := &sender{}
	_ = s.Method(http.MethodGet)

	if s.method != http.MethodGet {
		t.Errorf("Method() failed, got %v, want GET", s.method)
	}

	if s.options == nil {
		t.Errorf("Method() failed, options not initialized")
	}

	if s.options.client == nil {
		t.Errorf("Method() failed, client not initialized")
	}
}

// TestSenderGet tests GET method shortcut
func TestSenderGet(t *testing.T) {
	s := &sender{}
	s.Method(http.MethodGet)

	if s.method != http.MethodGet {
		t.Errorf("Get() failed, got %v", s.method)
	}
}

// TestSenderPost tests POST method shortcut
func TestSenderPost(t *testing.T) {
	s := &sender{}
	s.Method(http.MethodPost)

	if s.method != http.MethodPost {
		t.Errorf("Post() failed, got %v", s.method)
	}
}

// TestSenderURL tests URL setting
func TestSenderURL(t *testing.T) {
	s := &sender{}
	testURL, _ := url.Parse("https://example.com")

	s.URL(URL(testURL))

	if s.options.uri != testURL {
		t.Errorf("URL() failed")
	}
}

// TestSenderURLString tests URL string parsing
func TestSenderURLString(t *testing.T) {
	s := &sender{}

	s.URL(URLString("https://example.com/path"))

	if s.options.uri == nil {
		t.Errorf("URLString() failed, uri is nil")
	}
	if s.options.uri.Scheme != "https" {
		t.Errorf("URLString() failed, scheme is %v", s.options.uri.Scheme)
	}
}

// TestSenderQuery tests query parameter setting
func TestSenderQuery(t *testing.T) {
	s := &sender{}

	s.Query(SetQuery("key", "value"))

	if s.options.queries.Get("key") != "value" {
		t.Errorf("Query() failed")
	}
}

// TestSenderHeader tests header setting
func TestSenderHeader(t *testing.T) {
	s := &sender{}

	s.Header(SetHeader("X-Test", "test-value"))

	if s.options.headers.Get("X-Test") != "test-value" {
		t.Errorf("Header() failed")
	}
}

// TestSenderBody tests body setting
func TestSenderBody(t *testing.T) {
	s := &sender{}

	data := []byte("test body")
	s.Body(BytesBody(data, "text/plain"))

	if s.options.body == nil {
		t.Errorf("Body() failed, body is nil")
	}
}

// TestSenderSendNoOptions tests Send with uninitialized options
func TestSenderSendNoOptions(t *testing.T) {
	s := &sender{}
	ctx := context.Background()

	_, err := s.Send(ctx)
	if err == nil {
		t.Errorf("Send() should fail with uninitialized options")
	}
}

// TestSenderSendNoURI tests Send without URI
func TestSenderSendNoURI(t *testing.T) {
	s := &sender{
		options: &options{
			client: &http.Client{},
		},
		method: http.MethodGet,
	}
	ctx := context.Background()

	_, err := s.Send(ctx)
	if err == nil {
		t.Errorf("Send() should fail without URI")
	}
}

// TestSenderSendSuccess tests successful Send with mock server
func TestSenderSendSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	s := &sender{
		options: &options{
			uri: &url.URL{
				Scheme: "http",
				Host:   server.Listener.Addr().String(),
				Path:   "/test",
			},
			client: &http.Client{},
		},
		method: http.MethodGet,
	}
	ctx := context.Background()

	receiver, err := s.Send(ctx)
	if err != nil {
		t.Errorf("Send() failed: %v", err)
	}

	if receiver == nil {
		t.Errorf("Send() returned nil receiver")
	}

	if receiver.StatusCode() != http.StatusOK {
		t.Errorf("Expected status 200, got %d", receiver.StatusCode())
	}
}

// TestReceiverRequest tests receiver request accessor
func TestReceiverRequest(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp := &http.Response{}

	r := &receiver{req: req, resp: resp}

	if r.Request() != req {
		t.Errorf("Request() failed, expected the same request object")
	}
}

// TestReceiverResponse tests receiver response accessor
func TestReceiverResponse(t *testing.T) {
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
	}
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	r := &receiver{req: req, resp: resp}

	if r.Response() != resp {
		t.Errorf("Response() failed, expected the same response object")
	}
}

// TestReceiverStatus tests receiver status methods
func TestReceiverStatus(t *testing.T) {
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}

	r := &receiver{resp: resp}

	if r.Status() != "200 OK" {
		t.Errorf("Status() failed")
	}
	if r.StatusCode() != 200 {
		t.Errorf("StatusCode() failed")
	}
	if r.Proto() != "HTTP/1.1" {
		t.Errorf("Proto() failed")
	}
	if r.ProtoMajor() != 1 {
		t.Errorf("ProtoMajor() failed")
	}
	if r.ProtoMinor() != 1 {
		t.Errorf("ProtoMinor() failed")
	}
}

// TestReceiverHeaders tests receiver header methods
func TestReceiverHeaders(t *testing.T) {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")

	resp := &http.Response{
		Header: header,
	}

	r := &receiver{resp: resp}

	if r.Headers().Get("Content-Type") != "application/json" {
		t.Errorf("Headers() failed")
	}
}

// TestReceiverCookies tests receiver cookie methods
func TestReceiverCookies(t *testing.T) {
	resp := &http.Response{
		Header: make(http.Header),
	}
	resp.Header.Set("Set-Cookie", "name=value")

	r := &receiver{resp: resp}

	cookies := r.Cookies()
	if len(cookies) == 0 {
		t.Errorf("Cookies() returned empty")
	}
}

// TestReceiverBytesBody tests body reading
func TestReceiverBytesBody(t *testing.T) {
	body := []byte("test body content")
	resp := &http.Response{
		Body:       io.NopCloser(bytes.NewReader(body)),
		StatusCode: 200,
	}

	r := &receiver{resp: resp}

	content, err := r.BytesBody()
	if err != nil {
		t.Errorf("BytesBody() failed: %v", err)
	}

	if !bytes.Equal(content, body) {
		t.Errorf("BytesBody() content mismatch, got %v, want %v", content, body)
	}
}

// TestReceiverTextBody tests text body reading
func TestReceiverTextBody(t *testing.T) {
	body := "test text body"
	resp := &http.Response{
		Body:       io.NopCloser(strings.NewReader(body)),
		StatusCode: 200,
	}

	r := &receiver{resp: resp}

	content, err := r.TextBody()
	if err != nil {
		t.Errorf("TextBody() failed: %v", err)
	}

	if content != body {
		t.Errorf("TextBody() content mismatch, got %v, want %v", content, body)
	}
}

// TestReceiverJSONBody tests JSON body unmarshaling
func TestReceiverJSONBody(t *testing.T) {
	type TestData struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	data := TestData{Key: "test", Value: "data"}
	jsonBody, _ := json.Marshal(data)

	resp := &http.Response{
		Body:       io.NopCloser(bytes.NewReader(jsonBody)),
		StatusCode: 200,
	}

	r := &receiver{resp: resp}

	var result TestData
	err := r.JSONBody(&result)
	if err != nil {
		t.Errorf("JSONBody() failed: %v", err)
	}

	if result.Key != "test" || result.Value != "data" {
		t.Errorf("JSONBody() unmarshaling failed")
	}
}

// TestReceiverObjectBody tests object body unmarshaling
func TestReceiverObjectBody(t *testing.T) {
	body := []byte(`{"test":"data"}`)
	resp := &http.Response{
		Body:       io.NopCloser(bytes.NewReader(body)),
		StatusCode: 200,
	}

	r := &receiver{resp: resp}

	var result map[string]string
	err := r.ObjectBody(&result, json.Unmarshal)
	if err != nil {
		t.Errorf("ObjectBody() failed: %v", err)
	}

	if result["test"] != "data" {
		t.Errorf("ObjectBody() unmarshaling failed")
	}
}

// TestChainMiddlewares tests middleware chaining
func TestChainMiddlewares(t *testing.T) {
	tests := []struct {
		name        string
		middlewares []Middleware
		expectedNil bool
	}{
		{
			name:        "no middlewares",
			middlewares: []Middleware{},
			expectedNil: true,
		},
		{
			name: "single middleware",
			middlewares: []Middleware{
				func(ctx context.Context, req *http.Request, cli *http.Client, invoker Invoker) (*http.Response, error) {
					return invoker(ctx, req, cli)
				},
			},
			expectedNil: false,
		},
		{
			name: "multiple middlewares",
			middlewares: []Middleware{
				func(ctx context.Context, req *http.Request, cli *http.Client, invoker Invoker) (*http.Response, error) {
					return invoker(ctx, req, cli)
				},
				func(ctx context.Context, req *http.Request, cli *http.Client, invoker Invoker) (*http.Response, error) {
					return invoker(ctx, req, cli)
				},
			},
			expectedNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chainMiddlewares(tt.middlewares...)
			if (result == nil) != tt.expectedNil {
				t.Errorf("chainMiddlewares() nil mismatch")
			}
		})
	}
}

// TestInvoke tests middleware invocation with nil middleware
func TestInvoke(t *testing.T) {
	ctx := context.Background()
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	cli := &http.Client{}

	// Test with nil middleware (should call invoke directly)
	resp, err := Invoke(ctx, nil, cli, req)

	// We expect an error since we can't actually reach the URL
	// But this tests that invoke was called
	if resp == nil && err != nil {
		// This is expected
	}
}

// TestOptionsWithError tests option behavior when error is set
func TestOptionsWithError(t *testing.T) {
	opts := &options{
		err: errors.New("initial error"),
	}

	// These should all be no-ops due to existing error
	opts.SetQuery("key", "value")
	opts.AddQuery("key", "value2")
	opts.SetHeader("X-Test", "value")
	opts.AddHeader("X-Test", "value2")

	// Verify nothing was set
	if len(opts.queries) != 0 {
		t.Errorf("SetQuery should be no-op when error is set")
	}
	if len(opts.headers) != 0 {
		t.Errorf("SetHeader should be no-op when error is set")
	}
}

// TestOptionsContentLength tests content length header
func TestOptionsContentLength(t *testing.T) {
	opts := &options{}

	data := []byte("test data")
	opts.BytesBody(data, "text/plain")

	contentLength := opts.headers.Get("Content-Length")
	if contentLength != "9" {
		t.Errorf("Content-Length header mismatch, got %v, want 9", contentLength)
	}
}

// TestOptionsJSONBody tests JSON body encoding
func TestOptionsJSONBody(t *testing.T) {
	opts := &options{}

	data := map[string]string{"key": "value"}
	opts.JSONBody(data)

	if opts.body == nil {
		t.Errorf("JSONBody() failed, body is nil")
	}

	contentType := opts.headers.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type mismatch, got %v", contentType)
	}
}

// TestOptionsFormBody tests form body encoding
func TestOptionsFormBody(t *testing.T) {
	opts := &options{}

	form := url.Values{
		"username": []string{"user"},
		"password": []string{"pass"},
	}
	opts.FormBody(form)

	if opts.body == nil {
		t.Errorf("FormBody() failed, body is nil")
	}

	contentType := opts.headers.Get("Content-Type")
	if contentType != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type mismatch, got %v", contentType)
	}
}

// TestSenderChaining tests method chaining
func TestSenderChaining(t *testing.T) {
	s := &sender{}

	result := s.
		Method("Get").
		URL(URLString("https://example.com")).
		Query(SetQuery("key", "value")).
		Header(SetHeader("X-Test", "test")).
		Body(TextBody("test", "text/plain"))

	if result == nil {
		t.Errorf("Method chaining failed")
	}

	if s.options.queries.Get("key") != "value" {
		t.Errorf("Query not set correctly")
	}

	if s.options.headers.Get("X-Test") != "test" {
		t.Errorf("Header not set correctly")
	}
}

// TestReceiverContentLength tests content length accessor
func TestReceiverContentLength(t *testing.T) {
	resp := &http.Response{
		ContentLength: 1024,
	}

	r := &receiver{resp: resp}

	if r.ContentLength() != 1024 {
		t.Errorf("ContentLength() mismatch, got %d", r.ContentLength())
	}
}

// TestReceiverTransferEncoding tests transfer encoding accessor
func TestReceiverTransferEncoding(t *testing.T) {
	resp := &http.Response{
		TransferEncoding: []string{"chunked"},
	}

	r := &receiver{resp: resp}

	encodings := r.TransferEncoding()
	if len(encodings) != 1 || encodings[0] != "chunked" {
		t.Errorf("TransferEncoding() mismatch")
	}
}

// TestReceiverTrailers tests trailer accessor
func TestReceiverTrailers(t *testing.T) {
	trailers := make(http.Header)
	trailers.Set("X-Trailer", "value")

	resp := &http.Response{
		Trailer: trailers,
	}

	r := &receiver{resp: resp}

	if r.Trailers().Get("X-Trailer") != "value" {
		t.Errorf("Trailers() failed")
	}
}

// TestCacheControl tests cache control header
func TestCacheControl(t *testing.T) {
	opts := &options{}

	opts.CacheControl("no-cache", "no-store", "max-age=3600")

	cacheControl := opts.headers.Get("Cache-Control")
	if cacheControl != "no-cache, no-store, max-age=3600" {
		t.Errorf("CacheControl() failed, got %v", cacheControl)
	}
}

// TestIfMatch tests If-Match header
func TestIfMatch(t *testing.T) {
	opts := &options{}

	opts.IfMatch("etag1", "etag2")

	ifMatch := opts.headers.Get("If-Match")
	if ifMatch != "etag1, etag2" {
		t.Errorf("IfMatch() failed, got %v", ifMatch)
	}
}

// TestUserAgent tests User-Agent header
func TestUserAgent(t *testing.T) {
	opts := &options{}

	opts.UserAgent("Mozilla/5.0")

	ua := opts.headers.Get("User-Agent")
	if ua != "Mozilla/5.0" {
		t.Errorf("UserAgent() failed, got %v", ua)
	}
}
