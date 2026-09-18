package goose

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
)

// ErrorEncoder is a function type that defines how to encode errors into HTTP responses.
// It takes a context, an error to encode, and an http.ResponseWriter to write the response to.
//
// Parameters:
//   - ctx: The context.Context for the request
//   - err: The error to encode
//   - response: The http.ResponseWriter to write the encoded error response
type ErrorEncoder func(ctx context.Context, err error, response http.ResponseWriter)

// ErrorDecoder is a function type that defines how to decode errors from HTTP responses.
// It takes a context, an HTTP response, and an ErrorFactory, and returns the decoded error
// along with a boolean indicating whether decoding was successful.
//
// Parameters:
//   - ctx: The context.Context for the request
//   - response: The http.Response to decode the error from
//   - factory: The ErrorFactory used to create a new error instance
//
// Returns:
//   - error: The decoded error, or nil if decoding failed
//   - bool: True if decoding was successful, false otherwise
type ErrorDecoder func(ctx context.Context, response *http.Response, factory ErrorFactory) (error, bool)

// ErrorFactory is a function type that creates new error instances.
// It's used by ErrorDecoder to create the appropriate error type for decoding.
//
// Returns:
//   - error: A new error instance
type ErrorFactory func() error

// defaultError represents a standard HTTP error with status code, headers, and body.
// It implements several interfaces to provide rich error information.
type defaultError struct {
	statusCode int         // HTTP status code for the error
	headers    http.Header // HTTP headers associated with the error
	body       any         // Error body content
}

// NewError creates a new defaultError with the specified status code, body, and optional headers.
// Headers must be provided in key-value pairs, so the number of header arguments must be even.
//
// Parameters:
//   - statusCode: The HTTP status code for the error
//   - body: The error body content
//   - headers: Optional key-value pairs of HTTP headers (must be even number of arguments)
//
// Returns:
//   - error: A new error instance
//
// Panics:
//   - If the number of header arguments is odd
func NewError(statusCode int, body any, headers ...string) error {
	err := &defaultError{
		statusCode: statusCode,
		body:       body,
		headers:    http.Header{},
	}
	if len(headers)/2 != 0 {
		panic("goose: headers length must be even")
	}
	for i := 0; i < len(headers); i += 2 {
		err.headers.Add(headers[i], headers[i+1])
	}
	return err
}

// Error returns a string representation of the error, including status code and body.
//
// Returns:
//   - string: Formatted error message
func (e *defaultError) Error() string {
	return fmt.Sprintf("goose: http error, status code: %d, body: %s", e.statusCode, e.body)
}

// StatusCode returns the HTTP status code associated with this error.
//
// Returns:
//   - int: The HTTP status code
func (e *defaultError) StatusCode() int {
	return e.statusCode
}

// SetStatusCode sets the HTTP status code for this error.
//
// Parameters:
//   - code: The HTTP status code to set
func (e *defaultError) SetStatusCode(code int) {
	e.statusCode = code
}

// Headers returns the HTTP headers associated with this error.
//
// Returns:
//   - http.Header: The HTTP headers
func (e *defaultError) Headers() http.Header {
	return e.headers
}

// SetHeaders sets the HTTP headers for this error.
//
// Parameters:
//   - h: The HTTP headers to set
func (e *defaultError) SetHeaders(h http.Header) {
	e.headers = h
}

// MarshalJSON marshals the error body as JSON.
//
// Returns:
//   - []byte: The JSON-encoded error body
//   - error: Any error that occurred during marshaling
func (e *defaultError) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.body)
}

// UnmarshalJSON unmarshals JSON data into the error body.
//
// Parameters:
//   - data: The JSON data to unmarshal
//
// Returns:
//   - error: Any error that occurred during unmarshaling
func (e *defaultError) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &e.body)
}

// StatusCodeGetter defines an interface for errors that can provide a status code.
type StatusCodeGetter interface {
	StatusCode() int
}

// HeaderGetter defines an interface for errors that can provide HTTP headers.
type HeaderGetter interface {
	Headers() http.Header
}

// DefaultEncodeError encodes errors into HTTP responses with appropriate
// status codes and content type. Handles several error types:
// - json.Marshaler: encodes error as JSON if implemented
// - Headers() http.Header: adds headers to response if implemented
// - StatusCode() int: uses custom status code if implemented
//
// Parameters:
//   - ctx: context.Context for the request
//   - respErr: error to encode
//   - response: http.ResponseWriter to write the error response
func DefaultEncodeError(ctx context.Context, respErr error, response http.ResponseWriter) {
	if respErr == nil {
		return
	}
	// Default to 500 status code unless error provides specific status code
	code := http.StatusInternalServerError
	if statusCodeGetter, ok := respErr.(StatusCodeGetter); ok {
		code = statusCodeGetter.StatusCode()
	}

	// Default to plain text content type and error message as body
	contentType, body := PlainContentType, []byte(respErr.Error())
	// If the error implements json.Marshaler, try to marshal it as JSON
	if marshaler, ok := respErr.(json.Marshaler); ok {
		if jsonBody, err := marshaler.MarshalJSON(); err != nil {
			slog.ErrorContext(ctx, "goose: body marshal error", slog.String("error", err.Error()))
		} else {
			contentType, body = JsonContentType, jsonBody
		}
	}

	header := response.Header()
	// Set response content type header
	header.Set(ContentTypeKey, contentType)
	// If error provides custom headers, add them to the response
	keys := make([]string, 0)
	if headerGetter, ok := respErr.(HeaderGetter); ok {
		for key, values := range headerGetter.Headers() {
			for _, v := range values {
				header.Add(key, v)
				keys = append(keys, key)
			}
		}
	}
	keysJson, _ := json.Marshal(keys)
	header.Add(ErrorKey, string(keysJson))

	// Write HTTP status code and response body
	response.WriteHeader(code)
	_, respErr = response.Write(body)
	if respErr != nil {
		log.Println("goose: DefaultEncodeError, response write error: ", respErr)
	}
}

// StatusCodeSetter defines an interface for errors that can have their status code set.
type StatusCodeSetter interface {
	SetStatusCode(code int)
}

// HeaderSetter defines an interface for errors that can have their headers set.
type HeaderSetter interface {
	SetHeaders(h http.Header)
}

// DefaultErrorFactory creates a new defaultError instance.
//
// Returns:
//   - error: A new defaultError instance
func DefaultErrorFactory() error {
	return &defaultError{}
}

// DefaultDecodeError decodes errors from HTTP responses. It extracts error information
// including status code, headers, and body from the response.
//
// Parameters:
//   - ctx: The context.Context for the request
//   - response: The http.Response to decode the error from
//   - factory: The ErrorFactory used to create a new error instance
//
// Returns:
//   - error: The decoded error, or nil if decoding failed
//   - bool: True if decoding was successful, false otherwise
func DefaultDecodeError(ctx context.Context, response *http.Response, factory ErrorFactory) (error, bool) {
	keysJson := response.Header.Get(ErrorKey)
	if keysJson == "" {
		return nil, false
	}
	respErr := factory()

	if statusCodeGetter, ok := respErr.(StatusCodeSetter); ok {
		statusCodeGetter.SetStatusCode(response.StatusCode)
	}

	if headerSetter, ok := respErr.(HeaderSetter); ok {
		keys := make([]string, 0)
		err := json.Unmarshal([]byte(keysJson), &keys)
		if err != nil {
			log.Println("goose: header key unmarshal error: ", err)
		} else {
			headers := make(http.Header, len(keys))
			for _, key := range keys {
				for _, value := range response.Header.Values(key) {
					headers.Add(key, value)
				}
			}
			headerSetter.SetHeaders(headers)
		}
	}

	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if unmarshaler, ok := respErr.(json.Unmarshaler); ok {
		if err := unmarshaler.UnmarshalJSON(body); err != nil {
			log.Println("goose: body unmarshal error: ", err)
		}
	}
	return respErr, true
}
