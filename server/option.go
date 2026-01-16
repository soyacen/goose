package server

import (
	"github.com/soyacen/goose"
	"google.golang.org/protobuf/encoding/protojson"
)

// Options interface defines methods to access all configurable options for the server
type Options interface {
	// UnmarshalOptions returns the protojson unmarshal options used for decoding requests
	UnmarshalOptions() protojson.UnmarshalOptions

	// MarshalOptions returns the protojson marshal options used for encoding responses
	MarshalOptions() protojson.MarshalOptions

	// ErrorEncoder returns the error encoder used for encoding error responses
	ErrorEncoder() goose.ErrorEncoder

	// Middlewares returns the list of middlewares applied to requests
	Middlewares() []Middleware

	// ShouldFailFast indicates if fail-fast mode is enabled
	ShouldFailFast() bool

	// OnValidationErrCallback returns the validation error callback
	OnValidationErrCallback() goose.OnValidationErrCallback
}

// options holds the configuration options for the server
type options struct {
	unmarshalOptions        protojson.UnmarshalOptions    // Options for unmarshaling protobuf messages
	marshalOptions          protojson.MarshalOptions      // Options for marshaling protobuf messages
	errorEncoder            goose.ErrorEncoder            // Encoder for error responses
	middlewares             []Middleware                  // Middlewares applied to requests
	shouldFailFast          bool                          // Flag indicating if fail-fast mode is enabled
	onValidationErrCallback goose.OnValidationErrCallback // Callback for validation errors
}

// Option defines a function type for modifying server options
type Option func(o *options)

// apply applies the given options to the current options struct
//
// Parameters:
//   - opts: A variadic list of Option functions to apply
//
// Returns:
//   - *options: The modified options struct
func (o *options) apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// UnmarshalOptions returns the protojson unmarshal options used for decoding requests
//
// Returns:
//   - protojson.UnmarshalOptions: The unmarshal options
func (o *options) UnmarshalOptions() protojson.UnmarshalOptions {
	return o.unmarshalOptions
}

// MarshalOptions returns the protojson marshal options used for encoding responses
//
// Returns:
//   - protojson.MarshalOptions: The marshal options
func (o *options) MarshalOptions() protojson.MarshalOptions {
	return o.marshalOptions
}

// ErrorEncoder returns the error encoder used for encoding error responses
//
// Returns:
//   - goose.ErrorEncoder: The error encoder
func (o *options) ErrorEncoder() goose.ErrorEncoder {
	return o.errorEncoder
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

// UnmarshalOptions sets the protojson unmarshal options used for decoding requests
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

// MarshalOptions sets the protojson marshal options used for encoding responses
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

// ErrorEncoder configures a custom error encoder
//
// Parameters:
//   - encoder: The error encoder to use
//
// Returns:
//   - Option: A function that sets the error encoder option
func ErrorEncoder(encoder goose.ErrorEncoder) Option {
	return func(o *options) {
		o.errorEncoder = encoder
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

// FailFast enables fail-fast mode
//
// Returns:
//   - Option: A function that enables fail-fast mode
func FailFast() Option {
	return func(o *options) {
		o.shouldFailFast = true
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
	o := &options{
		unmarshalOptions:        protojson.UnmarshalOptions{},
		marshalOptions:          protojson.MarshalOptions{},
		errorEncoder:            goose.DefaultEncodeError,
		middlewares:             []Middleware{},
		shouldFailFast:          false,
		onValidationErrCallback: nil,
	}
	o = o.apply(opts...)
	return o
}
