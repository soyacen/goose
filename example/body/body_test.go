package body

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"google.golang.org/genproto/googleapis/api/httpbody"
	httprpc "google.golang.org/genproto/googleapis/rpc/http"
	rpchttp "google.golang.org/genproto/googleapis/rpc/http"
	"google.golang.org/protobuf/types/known/emptypb"
)

type MockBodyService struct{}

func (m *MockBodyService) HttpBodyNamedBody(ctx context.Context, request *HttpBodyRequest) (*Response, error) {
	var value BodyRequest
	if err := json.Unmarshal(request.GetBody().GetData(), &value); err != nil {
		return nil, err
	}
	return &Response{Message: value.GetMessage()}, nil
}

func (m *MockBodyService) HttpBodyStarBody(ctx context.Context, request *httpbody.HttpBody) (*Response, error) {
	var value BodyRequest
	if err := json.Unmarshal(request.GetData(), &value); err != nil {
		return nil, err
	}
	return &Response{Message: value.GetMessage()}, nil
}

func (m *MockBodyService) HttpRequest(ctx context.Context, request *rpchttp.HttpRequest) (*Response, error) {
	var value BodyRequest
	if err := json.Unmarshal(request.GetBody(), &value); err != nil {
		return nil, err
	}
	return &Response{Message: value.GetMessage()}, nil
}

func (m *MockBodyService) NamedBody(ctx context.Context, request *NamedBodyRequest) (*Response, error) {
	return &Response{Message: string(request.GetBody().GetMessage())}, nil
}

func (m *MockBodyService) NonBody(ctx context.Context, request *emptypb.Empty) (*Response, error) {
	return &Response{Message: "NonBody"}, nil
}

func (m *MockBodyService) StarBody(ctx context.Context, request *BodyRequest) (*Response, error) {
	return &Response{Message: request.GetMessage()}, nil
}

func runServer(server *http.Server, port int) {
	router := http.NewServeMux()
	router = AppendBodyRoute(router, &MockBodyService{})
	server.Addr = fmt.Sprintf(":%d", port)
	server.Handler = router
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

func newClient(port int) BodyService {
	return NewBodyClient(fmt.Sprintf("http://localhost:%d", port))
}

func TestStarBody(t *testing.T) {
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 38081)
	time.Sleep(1 * time.Second)
	client := newClient(38081)
	resp, err := client.StarBody(context.Background(), &BodyRequest{Message: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Message != "hello" {
		t.Fatal("resp is not equal")
	}
}

func TestNamedBody(t *testing.T) {
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 38082)
	client := newClient(38082)
	resp, err := client.NamedBody(context.Background(), &NamedBodyRequest{Body: &NamedBodyRequest_Body{Message: "hello"}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Message != "hello" {
		t.Fatal("resp is not equal")
	}
}

func TestNonBody(t *testing.T) {
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 38083)
	client := newClient(38083)
	resp, err := client.NonBody(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Message != "NonBody" {
		t.Fatal("resp is not equal")
	}
}

func TestHttpBodyStarBody(t *testing.T) {
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 38084)
	client := newClient(38084)
	resp, err := client.HttpBodyStarBody(context.Background(), &httpbody.HttpBody{
		Data: []byte(`{"message": "hello"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Message != "hello" {
		t.Fatal("resp is not equal")
	}
}

func TestHttpBodyNamedBody(t *testing.T) {
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 38085)
	client := newClient(38085)
	resp, err := client.HttpBodyNamedBody(context.Background(),
		&HttpBodyRequest{
			Body: &httpbody.HttpBody{
				Data: []byte(`{"message": "hello"}`),
			},
		})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Message != "hello" {
		t.Fatal("resp is not equal")
	}
}

func TestHttpRequest(t *testing.T) {
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 38086)
	client := newClient(38086)
	resp, err := client.HttpRequest(context.Background(),
		&httprpc.HttpRequest{
			Body: []byte(`{"message": "hello"}`),
		})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Message != "hello" {
		t.Fatal("resp is not equal")
	}
}
