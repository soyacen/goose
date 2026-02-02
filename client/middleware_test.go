package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockMiddleware is a test middleware that adds a header to the request
func mockMiddleware(headerKey, headerValue string) Middleware {
	return func(cli *http.Client, request *http.Request, invoker Invoker) (*http.Response, error) {
		request.Header.Add(headerKey, headerValue)
		return invoker(cli, request)
	}
}

// errorMiddleware is a test middleware that returns an error
func errorMiddleware(errorMessage string) Middleware {
	return func(cli *http.Client, request *http.Request, invoker Invoker) (*http.Response, error) {
		return nil, &testError{msg: errorMessage}
	}
}

// testError is a simple error implementation for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestChain(t *testing.T) {
	// Test with no middlewares
	mdw := Chain()
	if mdw != nil {
		t.Error("Chain should return nil when no middlewares provided")
	}

	// Test with one middleware
	mdw1 := mockMiddleware("Test-Key-1", "Test-Value-1")
	mdw = Chain(mdw1)
	if mdw == nil {
		t.Error("Chain should not return nil when one middleware provided")
	}

	// Test with multiple middlewares
	mdw2 := mockMiddleware("Test-Key-2", "Test-Value-2")
	mdw3 := mockMiddleware("Test-Key-3", "Test-Value-3")
	mdw = Chain(mdw1, mdw2, mdw3)
	if mdw == nil {
		t.Error("Chain should not return nil when multiple middlewares provided")
	}
}

func TestInvoke(t *testing.T) {
	var key1, key2 bool
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if headers were added by middleware
		if key1 && r.Header.Get("Test-Key-1") != "Test-Value-1" {
			t.Errorf("Expected header Test-Key-1 to be Test-Value-1, got %s", r.Header.Get("Test-Key-1"))
		}
		if key2 && r.Header.Get("Test-Key-2") != "Test-Value-2" {
			t.Errorf("Expected header Test-Key-2 to be Test-Value-2, got %s", r.Header.Get("Test-Key-2"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Create HTTP client
	cli := &http.Client{}

	// Create HTTP request
	request, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Test with nil middleware
	response, err := Invoke(nil, cli, request, nil)
	if err != nil {
		t.Errorf("Invoke with nil middleware returned error: %v", err)
	}
	if response == nil {
		t.Error("Invoke with nil middleware should return a response")
	}
	response.Body.Close()

	// Test with one middleware
	key1 = true
	key2 = false
	mdw1 := mockMiddleware("Test-Key-1", "Test-Value-1")
	response, err = Invoke(mdw1, cli, request, nil)
	if err != nil {
		t.Errorf("Invoke with one middleware returned error: %v", err)
	}
	if response == nil {
		t.Error("Invoke with one middleware should return a response")
	}
	response.Body.Close()

	// Test with chained middleware
	key1 = true
	key2 = true
	mdw2 := mockMiddleware("Test-Key-2", "Test-Value-2")
	chain := Chain(mdw1, mdw2)
	response, err = Invoke(chain, cli, request, nil)
	if err != nil {
		t.Errorf("Invoke with chained middleware returned error: %v", err)
	}
	if response == nil {
		t.Error("Invoke with chained middleware should return a response")
	}
	response.Body.Close()

	// Test with error middleware
	errorMdw := errorMiddleware("test error")
	response, err = Invoke(errorMdw, cli, request, nil)
	if err == nil {
		t.Error("Invoke with error middleware should return an error")
	}
	if response != nil {
		t.Error("Invoke with error middleware should not return a response")
		response.Body.Close()
	}
}

func TestGetInvoker(t *testing.T) {
	// Create test middlewares
	mdw1 := mockMiddleware("Test-Key-1", "Test-Value-1")
	mdw2 := mockMiddleware("Test-Key-2", "Test-Value-2")
	mdw3 := mockMiddleware("Test-Key-3", "Test-Value-3")

	middlewares := []Middleware{mdw1, mdw2, mdw3}

	// Test getInvoker for middle middleware
	invoker := getInvoker(middlewares, 0, func(cli *http.Client, request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200}, nil
	})

	if invoker == nil {
		t.Error("getInvoker should not return nil")
	}

	// Test that the invoker chain works correctly
	response, err := invoker(&http.Client{}, &http.Request{Header: http.Header{}})
	if err != nil {
		t.Errorf("Invoker chain returned error: %v", err)
	}
	if response == nil {
		t.Error("Invoker chain should return a response")
	}
}

func TestMiddlewareChainExecution(t *testing.T) {
	// Create a test server that verifies the order of middleware execution
	executionOrder := []int{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executionOrder = append(executionOrder, 0) // 0 represents the final handler
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Create middlewares that record their execution order
	mdw1 := func(cli *http.Client, request *http.Request, invoker Invoker) (*http.Response, error) {
		executionOrder = append(executionOrder, 1)
		return invoker(cli, request)
	}

	mdw2 := func(cli *http.Client, request *http.Request, invoker Invoker) (*http.Response, error) {
		executionOrder = append(executionOrder, 2)
		return invoker(cli, request)
	}

	mdw3 := func(cli *http.Client, request *http.Request, invoker Invoker) (*http.Response, error) {
		executionOrder = append(executionOrder, 3)
		return invoker(cli, request)
	}

	// Chain the middlewares
	chain := Chain(mdw1, mdw2, mdw3)

	// Create HTTP client and request
	cli := &http.Client{}
	request, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Execute the request
	executionOrder = []int{} // Reset execution order
	_, err = Invoke(chain, cli, request, nil)
	if err != nil {
		t.Errorf("Invoke returned error: %v", err)
	}

	// Verify execution order: middleware1 -> middleware2 -> middleware3 -> handler -> middleware3 -> middleware2 -> middleware1
	expectedOrder := []int{1, 2, 3, 0}
	if len(executionOrder) != len(expectedOrder) {
		t.Errorf("Execution order length mismatch. Got: %v, Want: %v", executionOrder, expectedOrder)
	} else {
		for i, v := range expectedOrder {
			if executionOrder[i] != v {
				t.Errorf("Execution order mismatch at position %d. Got: %d, Want: %d", i, executionOrder[i], v)
			}
		}
	}
}
