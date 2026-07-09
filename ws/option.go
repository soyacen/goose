package ws

import (
	"github.com/soyacen/goose/server"
	"google.golang.org/protobuf/encoding/protojson"
)

// Options interface defines methods to access all configurable options for the server
type Options interface {
	// UnmarshalOptions returns the protojson unmarshal options used for decoding requests
	UnmarshalOptions() protojson.UnmarshalOptions

	// MarshalOptions returns the protojson marshal options used for encoding responses
	MarshalOptions() protojson.MarshalOptions

	// Middlewares returns the list of middlewares applied to requests
	Middlewares() []server.Middleware
}

// options holds the configuration options for the server
type options struct {
	unmarshalOptions protojson.UnmarshalOptions // Options for unmarshaling protobuf messages
	marshalOptions   protojson.MarshalOptions   // Options for marshaling protobuf messages
	middlewares      []server.Middleware        // Middlewares applied to requests
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

// Middlewares returns the list of middlewares applied to requests
//
// Returns:
//   - []Middleware: The list of middlewares
func (o *options) Middlewares() []server.Middleware {
	return o.middlewares
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

// Middlewares appends middlewares to the chain of middlewares
//
// Parameters:
//   - middlewares: A variadic list of middlewares to append
//
// Returns:
//   - Option: A function that appends the middlewares
func Middlewares(middlewares ...server.Middleware) Option {
	return func(o *options) {
		o.middlewares = append(o.middlewares, middlewares...)
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
		unmarshalOptions: protojson.UnmarshalOptions{},
		marshalOptions:   protojson.MarshalOptions{},
		middlewares:      []server.Middleware{},
	}
	o = o.apply(opts...)
	return o
}
