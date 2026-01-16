package client

import (
	"context"
	"net/http"

	"github.com/soyacen/goose"
	"github.com/soyacen/goose/client/resolver"
	"google.golang.org/protobuf/encoding/protojson"
)

// Options interface defines methods to access all configurable options for the client
type Options interface {
	// Client returns the HTTP client used for making requests
	Client() *http.Client

	// UnmarshalOptions returns the protojson unmarshal options used for decoding responses
	UnmarshalOptions() protojson.UnmarshalOptions

	// MarshalOptions returns the protojson marshal options used for encoding requests
	MarshalOptions() protojson.MarshalOptions

	// ErrorDecoder returns the error decoder used for decoding error responses
	ErrorDecoder() goose.ErrorDecoder

	// ErrorFactory returns the error factory used for creating error instances
	ErrorFactory() goose.ErrorFactory

	// Middlewares returns the list of middlewares applied to requests
	Middlewares() []Middleware

	// ShouldFailFast indicates if fail-fast mode is enabled
	ShouldFailFast() bool

	// OnValidationErrCallback returns the validation error callback
	OnValidationErrCallback() goose.OnValidationErrCallback

	// Resolver returns the resolver used for resolving URLs
	Resolver() resolver.Resolver
}

// options holds the configuration options for the client
type options struct {
	client                  *http.Client                  // HTTP client for making requests
	unmarshalOptions        protojson.UnmarshalOptions    // Options for unmarshaling protobuf messages
	marshalOptions          protojson.MarshalOptions      // Options for marshaling protobuf messages
	errorDecoder            goose.ErrorDecoder            // Decoder for error responses
	errorFactory            goose.ErrorFactory            // Factory for creating error instances
	middlewares             []Middleware                  // Middlewares applied to requests
	shouldFailFast          bool                          // Flag indicating if fail-fast mode is enabled
	onValidationErrCallback goose.OnValidationErrCallback // Callback for validation errors
	resolver                resolver.Resolver             // Resolver used for resolving URLs
}

// Option defines a function type for modifying client options
type Option func(o *options)

// Apply applies the given options to the current options struct
//
// Parameters:
//   - opts: A variadic list of Option functions to Apply
//
// Returns:
//   - *options: The modified options struct
func (o *options) Apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func (o *options) Correct() *options {
	if o.client == nil {
		o.client = &http.Client{}
	}
	if o.errorDecoder == nil {
		o.errorDecoder = goose.DefaultDecodeError
	}
	if o.errorFactory == nil {
		o.errorFactory = goose.DefaultErrorFactory
	}
	if o.onValidationErrCallback == nil {
		o.onValidationErrCallback = func(ctx context.Context, err error) {}
	}
	return o
}

// Client returns the HTTP client used for making requests
//
// Returns:
//   - *http.Client: The HTTP client
func (o *options) Client() *http.Client {
	return o.client
}

// UnmarshalOptions returns the protojson unmarshal options used for decoding responses
//
// Returns:
//   - protojson.UnmarshalOptions: The unmarshal options
func (o *options) UnmarshalOptions() protojson.UnmarshalOptions {
	return o.unmarshalOptions
}

// MarshalOptions returns the protojson marshal options used for encoding requests
//
// Returns:
//   - protojson.MarshalOptions: The marshal options
func (o *options) MarshalOptions() protojson.MarshalOptions {
	return o.marshalOptions
}

// ErrorDecoder returns the error decoder used for decoding error responses
//
// Returns:
//   - goose.ErrorDecoder: The error decoder
func (o *options) ErrorDecoder() goose.ErrorDecoder {
	return o.errorDecoder
}

// ErrorFactory returns the error factory used for creating error instances
//
// Returns:
//   - goose.ErrorFactory: The error factory
func (o *options) ErrorFactory() goose.ErrorFactory {
	return o.errorFactory
}

// Middlewares returns the list of middlewares applied to requests
//
// Returns:
//   - []Middleware: The list of middlewares
func (o *options) Middlewares() []Middleware {
	return o.middlewares
}

// ShouldFailFast indicates if fail-fast mode is enabled
//
// Returns:
//   - bool: True if fail-fast mode is enabled, false otherwise
func (o *options) ShouldFailFast() bool {
	return o.shouldFailFast
}

// OnValidationErrCallback returns the validation error callback
//
// Returns:
//   - goose.OnValidationErrCallback: The validation error callback
func (o *options) OnValidationErrCallback() goose.OnValidationErrCallback {
	return o.onValidationErrCallback
}

// Resolver returns the resolver used for resolving URLs
//
// Returns:
//   - Resolver: The URL resolver
func (o *options) Resolver() resolver.Resolver {
	return o.resolver
}

// Client sets the HTTP client to be used for making requests
//
// Parameters:
//   - client: The HTTP client to use
//
// Returns:
//   - Option: A function that sets the client option
func Client(client *http.Client) Option {
	return func(o *options) {
		o.client = client
	}
}

// UnmarshalOptions sets the protojson unmarshal options used for decoding responses
//
// Parameters:
//   - opts: The protojson unmarshal options to use
//
// Returns:
//   - Option: A function that sets the unmarshal options
func UnmarshalOptions(opts protojson.UnmarshalOptions) Option {
	return func(o *options) {
		o.unmarshalOptions = opts
	}
}

// MarshalOptions sets the protojson marshal options used for encoding requests
//
// Parameters:
//   - opts: The protojson marshal options to use
//
// Returns:
//   - Option: A function that sets the marshal options
func MarshalOptions(opts protojson.MarshalOptions) Option {
	return func(o *options) {
		o.marshalOptions = opts
	}
}

// ErrorEncoder configures a custom error decoder
//
// Parameters:
//   - decoder: The error decoder to use
//
// Returns:
//   - Option: A function that sets the error decoder option
func ErrorEncoder(decoder goose.ErrorDecoder) Option {
	return func(o *options) {
		o.errorDecoder = decoder
	}
}

// ErrorFactory sets the error factory to be used for creating error instances
//
// Parameters:
//   - factory: The error factory to use
//
// Returns:
//   - Option: A function that sets the error factory option
func ErrorFactory(factory goose.ErrorFactory) Option {
	return func(o *options) {
		o.errorFactory = factory
	}
}

// Middlewares appends middlewares to the chain of middlewares
//
// Parameters:
//   - middlewares: A variadic list of middlewares to append
//
// Returns:
//   - Option: A function that appends the middlewares
func Middlewares(middlewares ...Middleware) Option {
	return func(o *options) {
		o.middlewares = append(o.middlewares, middlewares...)
	}
}

// FailFast enables fail-fast mode
//
// Returns:
//   - Option: A function that enables fail-fast mode
func FailFast() Option {
	return func(o *options) {
		o.shouldFailFast = true
	}
}

// OnValidationErrCallback sets the validation error callback
//
// Parameters:
//   - OnValidationErrCallback: The validation error callback to use
//
// Returns:
//   - Option: A function that sets the validation error callback
func OnValidationErrCallback(OnValidationErrCallback goose.OnValidationErrCallback) Option {
	return func(o *options) {
		o.onValidationErrCallback = OnValidationErrCallback
	}
}

func Resolvers(resolver resolver.Resolver) Option {
	return func(o *options) {
		o.resolver = resolver
	}
}

// NewOptions creates a new Options instance with default values and applies the provided options
//
// Parameters:
//   - opts: A variadic list of Option functions to apply
//
// Returns:
//   - Options: A new Options instance with the applied options
func NewOptions(opts ...Option) Options {
	o := &options{}
	o = o.Apply(opts...).Correct()
	return o
}
