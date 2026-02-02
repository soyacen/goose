package outgoing

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/soyacen/goose"
	"github.com/soyacen/goose/client"
	"github.com/soyacen/goose/internal/iox"
	"github.com/soyacen/goose/internal/strconvx"
	"google.golang.org/protobuf/proto"
)

func Get(opts ...MethodOption) URLSetter {
	return Method(http.MethodGet, opts...)
}

func Head(opts ...MethodOption) URLSetter {
	return Method(http.MethodHead, opts...)
}

func Post(opts ...MethodOption) URLSetter {
	return Method(http.MethodPost, opts...)
}

func Put(opts ...MethodOption) URLSetter {
	return Method(http.MethodPut, opts...)
}

func Patch(opts ...MethodOption) URLSetter {
	return Method(http.MethodPatch, opts...)
}

func Delete(opts ...MethodOption) URLSetter {
	return Method(http.MethodDelete, opts...)
}

func Connect(opts ...MethodOption) URLSetter {
	return Method(http.MethodConnect, opts...)
}

func Options(opts ...MethodOption) URLSetter {
	return Method(http.MethodOptions, opts...)
}

func Trace(opts ...MethodOption) URLSetter {
	return Method(http.MethodTrace, opts...)
}

func Method(method string, opts ...MethodOption) URLSetter {
	return new(sender).Method(method, opts...)
}

type MarshalError struct {
	Body  any
	Query any
	Err   error
}

func (e MarshalError) Error() string {
	if e.Body != nil {
		return "outgoing: failed to marshal body: " + e.Err.Error()
	} else if e.Query != nil {
		return "outgoing: failed to marshal query: " + e.Err.Error()
	}
	return "outgoing: unknown error"
}

func (e MarshalError) Unwrap() error {
	return e.Err
}

type UnmarshalError struct {
	Body []byte
	Err  error
}

func (e UnmarshalError) Error() string {
	return "outgoing: failed to unmarshal body"
}

func (e UnmarshalError) Unwrap() error {
	return e.Err
}

type options struct {
	err         error
	client      *http.Client
	middlewares []client.Middleware
	uri         *url.URL
	queries     url.Values
	headers     http.Header
	cookies     map[string][]*http.Cookie
	body        io.Reader
}

type (
	MethodOptions interface {
		Middleware(middlewares ...client.Middleware)
		Client(client *http.Client)
	}
	MethodOption func(MethodOptions)
	MethodSetter interface {
		Method(method string, opts ...MethodOption) URLSetter
	}
)

func Middlewares(middlewares ...client.Middleware) MethodOption {
	return func(o MethodOptions) {
		o.Middleware(middlewares...)
	}
}

func Client(client *http.Client) MethodOption {
	return func(o MethodOptions) {
		o.Client(client)
	}
}

func (s *options) Client(client *http.Client) {
	if s.err != nil {
		return
	}
	s.client = client
}

func (s *options) Middleware(middlewares ...client.Middleware) {
	if s.err != nil {
		return
	}
	s.middlewares = append(s.middlewares, middlewares...)
}

func (s *sender) Method(method string, opts ...MethodOption) URLSetter {
	if s.options == nil {
		s.options = new(options)
	}
	for _, opt := range opts {
		opt(s.options)
	}
	s.method = method
	if s.options.client == nil {
		s.options.client = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				MaxConnsPerHost:     100,
				IdleConnTimeout:     90 * time.Second,
				DisableKeepAlives:   false,
				DisableCompression:  false,
			},
		}
	}
	return s
}

type (
	URLOptions interface {
		URL(uri *url.URL)
		URLString(rawURL string)
	}
	URLOption func(URLOptions)
	URLSetter interface {
		URL(opts ...URLOption) QuerySetter
	}
)

func URL(uri *url.URL) URLOption {
	return func(o URLOptions) {
		o.URL(uri)
	}
}

func URLString(rawURL string) URLOption {
	return func(o URLOptions) {
		o.URLString(rawURL)
	}
}

func (s *options) URL(uri *url.URL) {
	if s.err != nil {
		return
	}
	s.uri = uri
}

func (s *options) URLString(rawURL string) {
	if s.err != nil {
		return
	}
	uri, err := url.Parse(rawURL)
	s.err = err
	s.uri = uri
}

func (s *sender) URL(opts ...URLOption) QuerySetter {
	if s.options == nil {
		s.options = new(options)
	}
	for _, opt := range opts {
		opt(s.options)
	}
	return s
}

type (
	QueryOptions interface {
		SetQuery(key, value string)
		AddQuery(key, value string)
		DelQuery(key string)
		QueryString(query string)
		Queries(values url.Values)
		QueryObject(obj any)
	}
	QueryOption func(QueryOptions)
	QuerySetter interface {
		Query(opts ...QueryOption) HeaderSetter
	}
)

func SetQuery(key, value string) QueryOption {
	return func(o QueryOptions) {
		o.SetQuery(key, value)
	}
}

func AddQuery(key, value string) QueryOption {
	return func(o QueryOptions) {
		o.AddQuery(key, value)
	}
}

func DelQuery(key string) QueryOption {
	return func(o QueryOptions) {
		o.DelQuery(key)
	}
}

func QueryString(query string) QueryOption {
	return func(o QueryOptions) {
		o.QueryString(query)
	}
}

func Queries(queries url.Values) QueryOption {
	return func(o QueryOptions) {
		o.Queries(queries)
	}
}

func QueryObject(obj any) QueryOption {
	return func(o QueryOptions) {
		o.QueryObject(obj)
	}
}

func (s *options) query() url.Values {
	if s.queries == nil {
		s.queries = make(url.Values)
	}
	return s.queries
}

func (s *options) SetQuery(key, value string) {
	if s.err != nil {
		return
	}
	s.query().Set(key, value)
}

func (s *options) AddQuery(key, value string) {
	if s.err != nil {
		return
	}
	s.query().Add(key, value)
}

func (s *options) DelQuery(key string) {
	if s.err != nil {
		return
	}
	s.query().Del(key)
}

func (s *options) QueryString(query string) {
	if s.err != nil {
		return
	}
	queries, err := url.ParseQuery(query)
	if err != nil {
		s.err = err
		return
	}
	s.Queries(queries)
}

func (s *options) Queries(values url.Values) {
	if s.err != nil {
		return
	}
	query := s.query()
	for key, values := range values {
		query[key] = append(query[key], values...)
	}
}

func (s *options) QueryObject(obj any) {
	if s.err != nil {
		return
	}
	values, err := query.Values(obj)
	if err != nil {
		s.err = MarshalError{Query: obj, Err: err}
		return
	}
	s.Queries(values)
}

func (s *sender) Query(opts ...QueryOption) HeaderSetter {
	if s.options == nil {
		s.options = new(options)
	}
	for _, opt := range opts {
		opt(s.options)
	}
	return s
}

type (
	HeaderOptions interface {
		SetHeader(key, value string, uncanonical ...bool)
		AddHeader(key, value string, uncanonical ...bool)
		DelHeader(key string)
		Headers(header http.Header)

		UserAgent(ua string)

		BasicAuth(username, password string)
		BearerAuth(token string)
		CustomAuth(scheme, token string)

		CacheControl(directives ...string)
		IfModifiedSince(t time.Time)
		IfUnmodifiedSince(t time.Time)
		IfNoneMatch(etag string)
		IfMatch(etags ...string)

		SetCookie(cookie *http.Cookie)
		AddCookie(cookie *http.Cookie)
		DelCookie(cookie *http.Cookie)
		Cookies(cookies ...*http.Cookie)
	}

	HeaderOption func(HeaderOptions)
	HeaderSetter interface {
		Header(opts ...HeaderOption) BodySetter
	}
)

// Header adds a header to the request.
func SetHeader(key, value string, uncanonical ...bool) HeaderOption {
	return func(o HeaderOptions) {
		o.SetHeader(key, value, uncanonical...)
	}
}

// AddHeader adds a header to the request.
func AddHeader(key, value string, uncanonical ...bool) HeaderOption {
	return func(o HeaderOptions) {
		o.AddHeader(key, value, uncanonical...)
	}
}

// DelHeader deletes a header from the request.
func DelHeader(key string) HeaderOption {
	return func(o HeaderOptions) {
		o.DelHeader(key)
	}
}

// Headers sets multiple headers to the request.
func Headers(header http.Header) HeaderOption {
	return func(o HeaderOptions) {
		o.Headers(header)
	}
}

// UserAgent sets the User-Agent header to the request.
func UserAgent(ua string) HeaderOption {
	return func(o HeaderOptions) {
		o.UserAgent(ua)
	}
}

// BasicAuth sets basic authentication credentials to the request.
func BasicAuth(username, password string) HeaderOption {
	return func(o HeaderOptions) {
		o.BasicAuth(username, password)
	}
}

// BearerAuth sets bearer authentication token to the request.
func BearerAuth(token string) HeaderOption {
	return func(o HeaderOptions) {
		o.BearerAuth(token)
	}
}

// CustomAuth sets custom authentication scheme and token to the request.
func CustomAuth(scheme, token string) HeaderOption {
	return func(o HeaderOptions) {
		o.CustomAuth(scheme, token)
	}
}

// CacheControl sets cache control directives to the request.
func CacheControl(directives ...string) HeaderOption {
	return func(o HeaderOptions) {
		o.CacheControl(directives...)
	}
}

// IfModifiedSince sets the If-Modified-Since header to the request.
func IfModifiedSince(t time.Time) HeaderOption {
	return func(o HeaderOptions) {
		o.IfModifiedSince(t)
	}
}

// IfUnmodifiedSince sets the If-Unmodified-Since header to the request.
func IfUnmodifiedSince(t time.Time) HeaderOption {
	return func(o HeaderOptions) {
		o.IfUnmodifiedSince(t)
	}
}

// IfNoneMatch sets the If-None-Match header to the request.
func IfNoneMatch(etag string) HeaderOption {
	return func(o HeaderOptions) {
		o.IfNoneMatch(etag)
	}
}

// IfMatch sets the If-Match header to the request.
func IfMatch(etags ...string) HeaderOption {
	return func(o HeaderOptions) {
		o.IfMatch(etags...)
	}
}

// Cookie sets a cookie to the request.
func SetCookie(cookie *http.Cookie) HeaderOption {
	return func(o HeaderOptions) {
		o.SetCookie(cookie)
	}
}

// AddCookie adds a cookie to the request.
func AddCookie(cookie *http.Cookie) HeaderOption {
	return func(o HeaderOptions) {
		o.AddCookie(cookie)
	}
}

// DelCookie deletes a cookie from the request.
func DelCookie(cookie *http.Cookie) HeaderOption {
	return func(o HeaderOptions) {
		o.DelCookie(cookie)
	}
}

// Cookies sets multiple cookies to the request.
func Cookies(cookies ...*http.Cookie) HeaderOption {
	return func(o HeaderOptions) {
		o.Cookies(cookies...)
	}
}

func (s *options) header() http.Header {
	if s.headers == nil {
		s.headers = make(http.Header)
	}
	return s.headers
}

func (s *options) SetHeader(key, value string, uncanonical ...bool) {
	if s.err != nil {
		return
	}
	header := s.header()
	if len(uncanonical) > 0 && uncanonical[0] {
		header[key] = []string{value}
		return
	}
	header.Set(key, value)
}

func (s *options) AddHeader(key, value string, uncanonical ...bool) {
	if s.err != nil {
		return
	}
	header := s.header()
	if len(uncanonical) > 0 && uncanonical[0] {
		header[key] = append(header[key], value)
		return
	}
	header.Add(key, value)
}

func (s *options) DelHeader(key string) {
	if s.err != nil {
		return
	}
	s.header().Del(key)
}

func (s *options) Headers(headers http.Header) {
	if s.err != nil {
		return
	}
	header := s.header()
	for key, values := range headers {
		header[key] = append(header[key], values...)
	}
}

func (s *options) BasicAuth(username, password string) {
	if s.err != nil {
		return
	}
	s.CustomAuth("Basic", base64.StdEncoding.EncodeToString(strconvx.StringToBytes(username+":"+password)))
}

func (s *options) BearerAuth(token string) {
	if s.err != nil {
		return
	}
	s.CustomAuth("Bearer", token)
}

func (s *options) CustomAuth(scheme, token string) {
	if s.err != nil {
		return
	}
	s.SetHeader("Authorization", scheme+" "+token)
}

func (s *options) UserAgent(ua string) {
	if s.err != nil {
		return
	}
	s.SetHeader("User-Agent", ua)
}

func (s *options) IfModifiedSince(t time.Time) {
	if s.err != nil {
		return
	}
	s.SetHeader("If-Modified-Since", t.UTC().Format(http.TimeFormat))
}

func (s *options) IfUnmodifiedSince(t time.Time) {
	if s.err != nil {
		return
	}
	s.SetHeader("If-Unmodified-Since", t.UTC().Format(http.TimeFormat))
}

func (s *options) IfNoneMatch(etag string) {
	if s.err != nil {
		return
	}
	s.SetHeader("If-None-Match", etag)
}

func (s *options) IfMatch(etags ...string) {
	if s.err != nil {
		return
	}
	s.SetHeader("If-Match", strings.Join(etags, ", "))
}

func (s *options) CacheControl(directives ...string) {
	if s.err != nil {
		return
	}
	s.SetHeader("Cache-Control", strings.Join(directives, ", "))
}

func (s *options) cookie() map[string][]*http.Cookie {
	if s.cookies == nil {
		s.cookies = make(map[string][]*http.Cookie)
	}
	return s.cookies
}

func (s *options) SetCookie(cookie *http.Cookie) {
	if s.err != nil {
		return
	}
	s.cookie()[cookie.Name] = []*http.Cookie{cookie}
}

func (s *options) AddCookie(cookie *http.Cookie) {
	if s.err != nil {
		return
	}
	s.Cookies(cookie)
}

func (s *options) DelCookie(cookie *http.Cookie) {
	if s.err != nil {
		return
	}
	delete(s.cookie(), cookie.Name)
}

func (s *options) Cookies(cookies ...*http.Cookie) {
	if s.err != nil {
		return
	}
	cookie := s.cookie()
	for _, item := range cookies {
		cookie[item.Name] = append(cookie[item.Name], item)
	}
}

func (s *sender) Header(opts ...HeaderOption) BodySetter {
	if s.options == nil {
		s.options = new(options)
	}
	for _, opt := range opts {
		opt(s.options)
	}
	return s
}

type (
	FormData struct {
		FieldName string
		Value     string
		File      io.Reader
		Filename  string
	}
	BodyOptions interface {
		Body(body io.Reader, contentType string)
		BytesBody(body []byte, contentType string)
		TextBody(body string, contentType string)
		ObjectBody(body any, marshal func(any) ([]byte, error), contentType string)
		JSONBody(body any)
		XMLBody(body any)
		ProtobufBody(body proto.Message)
		GobBody(body any)
		FormBody(form url.Values)
		FormObjectBody(body any)
		MultipartBody(formData ...*FormData)
	}
	BodyOption func(BodyOptions)
	BodySetter interface {
		Body(opts ...BodyOption) Sender
	}
)

// Body sets the body of the request.
func Body(body io.Reader, contentType string) BodyOption {
	return func(o BodyOptions) {
		o.Body(body, contentType)
	}
}

// BytesBody sets the body of the request as bytes.
func BytesBody(body []byte, contentType string) BodyOption {
	return func(o BodyOptions) {
		o.BytesBody(body, contentType)
	}
}

// TextBody sets the body of the request as text.
func TextBody(body string, contentType string) BodyOption {
	return func(o BodyOptions) {
		o.TextBody(body, contentType)
	}
}

// ObjectBody sets the body of the request using a custom marshal function.
func ObjectBody(body any, marshal func(any) ([]byte, error), contentType string) BodyOption {
	return func(o BodyOptions) {
		o.ObjectBody(body, marshal, contentType)
	}
}

// JSONBody sets the body of the request as JSON.
func JSONBody(body any) BodyOption {
	return func(o BodyOptions) {
		o.JSONBody(body)
	}
}

// XMLBody sets the body of the request as XML.
func XMLBody(body any) BodyOption {
	return func(o BodyOptions) {
		o.XMLBody(body)
	}
}

// ProtobufBody sets the body of the request as Protobuf.
func ProtobufBody(body proto.Message) BodyOption {
	return func(o BodyOptions) {
		o.ProtobufBody(body)
	}
}

// GobBody sets the body of the request as Gob.
func GobBody(body any) BodyOption {
	return func(o BodyOptions) {
		o.GobBody(body)
	}
}

// FormBody sets the body of the request as form data.
func FormBody(form url.Values) BodyOption {
	return func(o BodyOptions) {
		o.FormBody(form)
	}
}

// FormObjectBody sets the body of the request using an object to generate form data.
func FormObjectBody(body any) BodyOption {
	return func(o BodyOptions) {
		o.FormObjectBody(body)
	}
}

// MultipartBody sets the body of the request as multipart form data.
func MultipartBody(formData ...*FormData) BodyOption {
	return func(o BodyOptions) {
		o.MultipartBody(formData...)
	}
}

func (s *options) Body(body io.Reader, contentType string) {
	if s.err != nil {
		return
	}
	s.body = body
	s.SetHeader("Content-Type", contentType)
	l, ok := iox.Len(body)
	if ok {
		s.SetHeader("Content-Length", strconvx.FormatUint(l, 10))
	}
}

func (s *options) BytesBody(body []byte, contentType string) {
	if s.err != nil {
		return
	}
	s.Body(bytes.NewReader(body), contentType)
}

func (s *options) TextBody(body string, contentType string) {
	if s.err != nil {
		return
	}
	s.Body(strings.NewReader(body), contentType)
}

func (s *options) ObjectBody(body any, marshal func(any) ([]byte, error), contentType string) {
	if s.err != nil {
		return
	}
	data, err := marshal(body)
	if err != nil {
		s.err = MarshalError{Body: body, Err: err}
		return
	}
	s.BytesBody(data, contentType)
}

func marshalJSON(v any) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(v)
	return buffer.Bytes(), err
}

func (s *options) JSONBody(body any) {
	if s.err != nil {
		return
	}
	s.ObjectBody(body, marshalJSON, "application/json")
}

func (s *options) XMLBody(body any) {
	if s.err != nil {
		return
	}
	s.ObjectBody(body, xml.Marshal, "application/xml")
}

func marshalProtobuf(v any) ([]byte, error) {
	message, _ := v.(proto.Message)
	return proto.Marshal(message)
}

func (s *options) ProtobufBody(body proto.Message) {
	if s.err != nil {
		return
	}
	s.ObjectBody(body, marshalProtobuf, "application/x-protobuf")
}

func marshal(v any) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := gob.NewEncoder(buffer)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (s *options) GobBody(body any) {
	if s.err != nil {
		return
	}
	s.ObjectBody(body, marshal, "application/x-gob")
}

func (s *options) FormBody(form url.Values) {
	if s.err != nil {
		return
	}
	s.TextBody(form.Encode(), "application/x-www-form-urlencoded")
}

func (s *options) FormObjectBody(body any) {
	if s.err != nil {
		return
	}
	form, err := query.Values(body)
	if err != nil {
		s.err = MarshalError{Body: body, Err: err}
		return
	}
	s.FormBody(form)
}

func (s *options) MultipartBody(formData ...*FormData) {
	if s.err != nil {
		return
	}
	payload := new(bytes.Buffer)
	writer := multipart.NewWriter(payload)
	for _, form := range formData {
		if form.File == nil {
			_ = writer.WriteField(form.FieldName, form.Value)
			continue
		}
		mf, err := writer.CreateFormFile(form.FieldName, filepath.Base(form.Filename))
		if err != nil {
			s.err = err
			return
		}
		if _, err = io.Copy(mf, form.File); err != nil {
			s.err = err
			return
		}
	}
	if err := writer.Close(); err != nil {
		s.err = err
		return
	}
	s.Body(payload, writer.FormDataContentType())
}

func (s *sender) Body(opts ...BodyOption) Sender {
	if s.options == nil {
		s.options = new(options)
	}
	for _, opt := range opts {
		opt(s.options)
	}
	return s
}

type Sender interface {
	Send(ctx context.Context) (Receiver, error)
}

func (s *sender) Send(ctx context.Context) (Receiver, error) {
	if s.options == nil {
		return nil, errors.New("outgoing: options not initialized")
	}
	if s.options.err != nil {
		return nil, s.options.err
	}
	if s.options.uri == nil {
		return nil, errors.New("outgoing: uri not set")
	}
	if s.options.client == nil {
		return nil, errors.New("outgoing: client not initialized")
	}
	query := s.options.uri.Query()
	for name, values := range s.options.queries {
		query[name] = append(query[name], values...)
	}
	s.options.uri.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, s.method, s.options.uri.String(), s.options.body)
	if err != nil {
		return nil, err
	}
	for key, values := range s.options.headers {
		req.Header[key] = append(req.Header[key], values...)
	}
	for _, cookies := range s.options.cookies {
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
	}
	resp, err := client.Invoke(client.Chain(s.options.middlewares...), s.options.client, req, &goose.RouteInfo{HttpMethod: s.method, Pattern: s.options.uri.Path})
	if err != nil {
		return nil, err
	}
	return &receiver{req: req, resp: resp}, nil
}

type Receiver interface {
	Request() *http.Request
	Response() *http.Response
	Status() string
	StatusCode() int
	Proto() string
	ProtoMajor() int
	ProtoMinor() int
	ContentLength() int64
	TransferEncoding() []string
	Headers() http.Header
	Trailers() http.Header
	Cookies() []*http.Cookie

	Body(file io.Writer) error
	BytesBody() ([]byte, error)
	TextBody() (string, error)
	ObjectBody(body any, unmarshal func([]byte, any) error) error
	JSONBody(body any) error
	XMLBody(body any) error
	ProtobufBody(body proto.Message) error
	GobBody(body any) error
}

func (r *receiver) Request() *http.Request {
	return r.req
}

func (r *receiver) Response() *http.Response {
	return r.resp
}

func (r *receiver) Status() string {
	return r.resp.Status
}

func (r *receiver) StatusCode() int {
	return r.resp.StatusCode
}

func (r *receiver) Proto() string {
	return r.resp.Proto
}

func (r *receiver) ProtoMajor() int {
	return r.resp.ProtoMajor
}

func (r *receiver) ProtoMinor() int {
	return r.resp.ProtoMinor
}

func (r *receiver) ContentLength() int64 {
	return r.resp.ContentLength
}

func (r *receiver) TransferEncoding() []string {
	return r.resp.TransferEncoding
}

func (r *receiver) Headers() http.Header {
	return r.resp.Header
}

func (r *receiver) Trailers() http.Header {
	return r.resp.Trailer
}

func (r *receiver) Cookies() []*http.Cookie {
	return r.resp.Cookies()
}

func (r *receiver) Body(file io.Writer) error {
	_, err := io.Copy(file, r.resp.Body)
	defer r.resp.Body.Close()
	return err
}

func (r *receiver) BytesBody() ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := r.Body(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (r *receiver) TextBody() (string, error) {
	bodyBytes, err := r.BytesBody()
	if err != nil {
		return "", err
	}
	return strconvx.BytesToString(bodyBytes), nil
}

func (r *receiver) ObjectBody(body any, unmarshal func([]byte, any) error) error {
	bodyBytes, err := r.BytesBody()
	if err != nil {
		return err
	}
	if err := unmarshal(bodyBytes, body); err != nil {
		return UnmarshalError{Body: bodyBytes, Err: err}
	}
	return nil
}

func (r *receiver) JSONBody(body any) error {
	return r.ObjectBody(body, json.Unmarshal)
}

func (r *receiver) XMLBody(body any) error {
	return r.ObjectBody(body, xml.Unmarshal)
}

func unmarshalProtobuf(data []byte, v any) error { return proto.Unmarshal(data, v.(proto.Message)) }

func (r *receiver) ProtobufBody(body proto.Message) error {
	return r.ObjectBody(body, unmarshalProtobuf)
}

func unmarshalGob(data []byte, v any) error { return gob.NewDecoder(bytes.NewReader(data)).Decode(v) }

func (r *receiver) GobBody(body any) error {
	return r.ObjectBody(body, unmarshalGob)
}

type sender struct {
	options *options
	err     error
	method  string
}

type receiver struct {
	req  *http.Request
	resp *http.Response
}
