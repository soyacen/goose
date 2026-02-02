package response_body

import (
	"context"
	errors "errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	httpbody "google.golang.org/genproto/googleapis/api/httpbody"
	rpchttp "google.golang.org/genproto/googleapis/rpc/http"
)

// ---- Mock Service ----

type MockResponseBodyService struct{}

func (m *MockResponseBodyService) OmittedResponse(ctx context.Context, req *Request) (*Response, error) {
	return &Response{Message: req.GetMessage()}, nil
}

func (m *MockResponseBodyService) StarResponse(ctx context.Context, req *Request) (*Response, error) {
	return &Response{Message: req.GetMessage()}, nil
}

func (m *MockResponseBodyService) NamedResponse(ctx context.Context, req *Request) (*NamedBodyResponse, error) {
	return &NamedBodyResponse{
		Body: &NamedBodyResponse_Body{Message: req.GetMessage()},
	}, nil
}

func (m *MockResponseBodyService) HttpBodyResponse(ctx context.Context, req *Request) (*httpbody.HttpBody, error) {
	return &httpbody.HttpBody{
		ContentType: "text/plain",
		Data:        []byte(req.GetMessage()),
	}, nil
}

func (m *MockResponseBodyService) HttpBodyNamedResponse(ctx context.Context, req *Request) (*NamedHttpBodyResponse, error) {
	return &NamedHttpBodyResponse{
		Body: &httpbody.HttpBody{
			ContentType: "application/json",
			Data:        []byte(req.GetMessage()),
		},
	}, nil
}

func (m *MockResponseBodyService) HttpResponse(ctx context.Context, req *Request) (*rpchttp.HttpResponse, error) {
	return &rpchttp.HttpResponse{
		Status: 200,
		Body:   []byte(req.GetMessage()),
	}, nil
}

func runServer(server *http.Server, port int) {
	router := http.NewServeMux()
	router = AppendResponseBodyHttpRoute(router, &MockResponseBodyService{})
	server.Addr = fmt.Sprintf(":%d", port)
	server.Handler = router
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

func newClient(port int) ResponseBodyService {
	return NewResponseBodyHttpClient(fmt.Sprintf("http://localhost:%d", port))
}

// ---- Test Cases ----

func TestOmittedResponse(t *testing.T) {
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 38081)
	time.Sleep(1 * time.Second)

	client := newClient(38081)
	resp, err := client.OmittedResponse(context.Background(), &Request{Message: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Message != "hello" {
		t.Fatal("resp is not equal")
	}
}

func TestStarResponse(t *testing.T) {
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 38082)
	time.Sleep(1 * time.Second)

	client := newClient(38082)
	resp, err := client.StarResponse(context.Background(), &Request{Message: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Message != "hello" {
		t.Fatal("resp is not equal")
	}
}

func TestNamedResponse(t *testing.T) {
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 38083)
	time.Sleep(1 * time.Second)

	client := newClient(38083)
	resp, err := client.NamedResponse(context.Background(), &Request{Message: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetBody().GetMessage() != "hello" {
		t.Fatal("resp is not equal")
	}
}

func TestHttpBodyResponse(t *testing.T) {
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 38084)
	time.Sleep(1 * time.Second)

	client := newClient(38084)
	resp, err := client.HttpBodyResponse(context.Background(), &Request{Message: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if string(resp.GetData()) != "hello" {
		t.Fatal("resp is not equal")
	}
}

func TestHttpBodyNamedResponse(t *testing.T) {
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 38085)
	time.Sleep(1 * time.Second)

	client := newClient(38085)
	resp, err := client.HttpBodyNamedResponse(context.Background(), &Request{Message: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if string(resp.GetBody().GetData()) != "hello" {
		t.Fatal("resp is not equal")
	}
}

func TestHttpResponse(t *testing.T) {
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 38085)
	time.Sleep(1 * time.Second)

	client := newClient(38085)
	resp, err := client.HttpResponse(context.Background(), &Request{Message: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if string(resp.GetBody()) != "hello" {
		t.Fatal("resp is not equal")
	}
}
