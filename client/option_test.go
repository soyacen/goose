package client

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/soyacen/goose"
	"google.golang.org/protobuf/encoding/protojson"
)

// mockMiddlewareOpt is a test middleware for testing purposes
func mockMiddlewareOpt( cli *http.Client, request *http.Request, invoker Invoker) (*http.Response, error) {
	return invoker( cli, request)
}

// mockErrorDecoder is a test error decoder for testing purposes
func mockErrorDecoder(ctx context.Context, response *http.Response, factory goose.ErrorFactory) (error, bool) {
	return nil, false
}

// mockErrorFactory is a test error factory for testing purposes
func mockErrorFactory() error {
	return nil
}

// mockValidationCallback is a test validation error callback for testing purposes
func mockValidationCallback(ctx context.Context, err error) {
	// Do nothing
}

func TestOptionsInterface(t *testing.T) {
	// Test that options struct implements Options interface
	var _ Options = &options{}
}

func TestOptionsGetters(t *testing.T) {
	// Create test options with specific values
	testClient := &http.Client{}
	testUnmarshalOptions := protojson.UnmarshalOptions{DiscardUnknown: true}
	testMarshalOptions := protojson.MarshalOptions{EmitUnpopulated: true}
	testErrorDecoder := mockErrorDecoder
	testErrorFactory := mockErrorFactory
	testMiddlewares := []Middleware{mockMiddlewareOpt}
	testShouldFailFast := true
	testValidationCallback := mockValidationCallback

	opts := &options{
		client:                  testClient,
		unmarshalOptions:        testUnmarshalOptions,
		marshalOptions:          testMarshalOptions,
		errorDecoder:            testErrorDecoder,
		errorFactory:            testErrorFactory,
		middlewares:             testMiddlewares,
		shouldFailFast:          testShouldFailFast,
		onValidationErrCallback: testValidationCallback,
	}

	// Test Client getter
	if opts.Client() != testClient {
		t.Error("Client() returned unexpected value")
	}

	// Test UnmarshalOptions getter
	if !reflect.DeepEqual(opts.UnmarshalOptions(), testUnmarshalOptions) {
		t.Error("UnmarshalOptions() returned unexpected value")
	}

	// Test MarshalOptions getter
	if !reflect.DeepEqual(opts.MarshalOptions(), testMarshalOptions) {
		t.Error("MarshalOptions() returned unexpected value")
	}

	// Test ErrorDecoder getter
	if reflect.ValueOf(opts.ErrorDecoder()).Pointer() != reflect.ValueOf(testErrorDecoder).Pointer() {
		t.Error("ErrorDecoder() returned unexpected value")
	}

	// Test ErrorFactory getter
	if reflect.ValueOf(opts.ErrorFactory()).Pointer() != reflect.ValueOf(testErrorFactory).Pointer() {
		t.Error("ErrorFactory() returned unexpected value")
	}

	// Test Middlewares getter
	if !reflect.DeepEqual(opts.Middlewares(), testMiddlewares) {
		t.Error("Middlewares() returned unexpected value")
	}

	// Test ShouldFailFast getter
	if opts.ShouldFailFast() != testShouldFailFast {
		t.Error("ShouldFailFast() returned unexpected value")
	}

	// Test OnValidationErrCallback getter
	if reflect.ValueOf(opts.OnValidationErrCallback()).Pointer() != reflect.ValueOf(testValidationCallback).Pointer() {
		t.Error("OnValidationErrCallback() returned unexpected value")
	}
}

func TestClientOption(t *testing.T) {
	opts := &options{}
	testClient := &http.Client{}

	// Apply Client option
	option := Client(testClient)
	option(opts)

	// Verify the client was set
	if opts.client != testClient {
		t.Error("Client option did not set the client correctly")
	}
}

func TestUnmarshalOptionsOption(t *testing.T) {
	opts := &options{}
	testUnmarshalOptions := protojson.UnmarshalOptions{DiscardUnknown: true}

	// Apply UnmarshalOptions option
	option := UnmarshalOptions(testUnmarshalOptions)
	option(opts)

	// Verify the unmarshal options were set
	if !reflect.DeepEqual(opts.unmarshalOptions, testUnmarshalOptions) {
		t.Error("UnmarshalOptions option did not set the unmarshal options correctly")
	}
}

func TestMarshalOptionsOption(t *testing.T) {
	opts := &options{}
	testMarshalOptions := protojson.MarshalOptions{EmitUnpopulated: true}

	// Apply MarshalOptions option
	option := MarshalOptions(testMarshalOptions)
	option(opts)

	// Verify the marshal options were set
	if !reflect.DeepEqual(opts.marshalOptions, testMarshalOptions) {
		t.Error("MarshalOptions option did not set the marshal options correctly")
	}
}

func TestErrorEncoderOption(t *testing.T) {
	opts := &options{}
	testErrorDecoder := mockErrorDecoder

	// Apply ErrorEncoder option (which sets the error decoder)
	option := ErrorEncoder(testErrorDecoder)
	option(opts)

	// Verify the error decoder was set
	if reflect.ValueOf(opts.errorDecoder).Pointer() != reflect.ValueOf(testErrorDecoder).Pointer() {
		t.Error("ErrorEncoder option did not set the error decoder correctly")
	}
}

func TestErrorFactoryOption(t *testing.T) {
	opts := &options{}
	testErrorFactory := mockErrorFactory

	// Apply ErrorFactory option
	option := ErrorFactory(testErrorFactory)
	option(opts)

	// Verify the error factory was set
	if reflect.ValueOf(opts.errorFactory).Pointer() != reflect.ValueOf(testErrorFactory).Pointer() {
		t.Error("ErrorFactory option did not set the error factory correctly")
	}
}

func TestMiddlewaresOption(t *testing.T) {
	opts := &options{}
	testMiddlewares := []Middleware{mockMiddlewareOpt}

	// Apply Middlewares option
	option := Middlewares(testMiddlewares...)
	option(opts)

	// Verify the middlewares were set
	if len(opts.middlewares) != len(testMiddlewares) {
		t.Error("Middlewares option did not set the middlewares correctly")
	}

	// Test appending additional middlewares
	additionalMiddlewares := []Middleware{mockMiddlewareOpt}
	option2 := Middlewares(additionalMiddlewares...)
	option2(opts)

	// Verify the middlewares were appended
	expectedMiddlewares := append(testMiddlewares, additionalMiddlewares...)
	if len(opts.middlewares) != len(expectedMiddlewares) {
		t.Error("Middlewares option did not append the middlewares correctly")
	}
}

func TestFailFastOption(t *testing.T) {
	opts := &options{}

	// Apply FailFast option
	option := FailFast()
	option(opts)

	// Verify fail fast mode was enabled
	if !opts.shouldFailFast {
		t.Error("FailFast option did not enable fail fast mode")
	}
}

func TestOnValidationErrCallbackOption(t *testing.T) {
	opts := &options{}
	testCallback := mockValidationCallback

	// Apply OnValidationErrCallback option
	option := OnValidationErrCallback(testCallback)
	option(opts)

	// Verify the callback was set
	if reflect.ValueOf(opts.onValidationErrCallback).Pointer() != reflect.ValueOf(testCallback).Pointer() {
		t.Error("OnValidationErrCallback option did not set the callback correctly")
	}
}

func TestNewOptions(t *testing.T) {
	// Test NewOptions with no options
	opts := NewOptions()
	if opts == nil {
		t.Fatal("NewOptions returned nil")
	}

	// Verify default values
	if opts.Client() == nil {
		t.Error("Default client should not be nil")
	}

	if opts.UnmarshalOptions() != (protojson.UnmarshalOptions{}) {
		t.Error("Default unmarshal options are incorrect")
	}

	if opts.MarshalOptions() != (protojson.MarshalOptions{}) {
		t.Error("Default marshal options are incorrect")
	}

	if opts.ErrorDecoder() == nil {
		t.Error("Default error decoder should not be nil")
	}

	if opts.ErrorFactory() == nil {
		t.Error("Default error factory should not be nil")
	}

	if opts.ShouldFailFast() != false {
		t.Error("Default shouldFailFast should be false")
	}

	// Test NewOptions with custom options
	testClient := &http.Client{}
	customOpts := NewOptions(Client(testClient), FailFast())

	if customOpts.Client() != testClient {
		t.Error("Custom client option was not applied")
	}

	if !customOpts.ShouldFailFast() {
		t.Error("Custom fail fast option was not applied")
	}
}

func TestApplyMethod(t *testing.T) {
	opts := &options{}

	// Test applying multiple options
	testClient := &http.Client{}
	testMiddlewares := []Middleware{mockMiddlewareOpt}

	opts = opts.Apply(
		Client(testClient),
		Middlewares(testMiddlewares...),
		FailFast(),
	)

	// Verify all options were applied
	if opts.client != testClient {
		t.Error("Client option was not applied in apply method")
	}

	if len(opts.middlewares) != len(testMiddlewares) {
		t.Error("Middlewares option was not applied in apply method")
	}

	if !opts.shouldFailFast {
		t.Error("FailFast option was not applied in apply method")
	}
}
