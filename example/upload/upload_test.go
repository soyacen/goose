package upload

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"testing"
	"time"

	gupload "github.com/soyacen/goose/upload"
	"google.golang.org/genproto/googleapis/api/httpbody"
	rpchttp "google.golang.org/genproto/googleapis/rpc/http"
)

// ---- Mock Service ----

// MockUploadService implements UploadService using upload.Handler for integration testing.
type MockUploadService struct {
	handler *gupload.Handler
}

// Upload handles PUT /v1/upload/api — receives raw HttpBody (body: "*")
//
// Parameters:
//   - ctx: Request context
//   - req: HttpBody containing raw multipart/form-data bytes
//
// Returns:
//   - *Response: Response with upload result JSON
//   - error: Error if upload processing fails
func (s *MockUploadService) Upload(ctx context.Context, req *httpbody.HttpBody) (*Response, error) {
	result, err := s.handler.Handle(req.GetData(), req.GetContentType())
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	return &Response{Message: result.JSON()}, nil
}

// UploadEmbed handles PUT /v1/upload/embd — receives HttpBodyRequest with embedded HttpBody (body: "body")
//
// Parameters:
//   - ctx: Request context
//   - req: UploadEmbedRequest containing embedded HttpBody
//
// Returns:
//   - *Response: Response with upload result JSON
//   - error: Error if upload processing fails
func (s *MockUploadService) UploadEmbed(ctx context.Context, req *UploadEmbedRequest) (*Response, error) {
	body := req.GetBody()
	result, err := s.handler.Handle(body.GetData(), body.GetContentType())
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	return &Response{Message: result.JSON()}, nil
}

// UploadForRPC handles PUT /v1/upload/rpc — receives google.rpc.HttpRequest (body: "*")
//
// Parameters:
//   - ctx: Request context
//   - req: HttpRequest containing raw body bytes and headers
//
// Returns:
//   - *Response: Response with upload result JSON
//   - error: Error if upload processing fails
func (s *MockUploadService) UploadForRPC(ctx context.Context, req *rpchttp.HttpRequest) (*Response, error) {
	contentType := ""
	for _, h := range req.GetHeaders() {
		if h.GetKey() == "Content-Type" {
			contentType = h.GetValue()
			break
		}
	}
	result, err := s.handler.Handle(req.GetBody(), contentType)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	return &Response{Message: result.JSON()}, nil
}

func runServer(server *http.Server, port int, uploadDir string) {
	handler, err := gupload.NewHandler(gupload.WithUploadDir(uploadDir))
	if err != nil {
		panic(err)
	}
	router := http.NewServeMux()
	router = AppendUploadHttpRoute(router, &MockUploadService{handler: handler})
	server.Addr = fmt.Sprintf(":%d", port)
	server.Handler = router
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

func newClient(port int) UploadService {
	return NewUploadHttpClient(fmt.Sprintf("http://localhost:%d", port))
}

// buildMultipartForm constructs a multipart/form-data body with one file part and one form field.
func buildMultipartForm() ([]byte, string) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("name", "test_upload")
	fileWriter, _ := writer.CreateFormFile("file", "test.txt")
	_, _ = fileWriter.Write([]byte("hello file content"))
	_ = writer.Close()
	return buf.Bytes(), writer.FormDataContentType()
}

// ---- Test Cases ----

func TestUpload(t *testing.T) {
	uploadDir := t.TempDir()
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 58081, uploadDir)
	time.Sleep(1 * time.Second)

	client := newClient(58081)
	data, contentType := buildMultipartForm()
	resp, err := client.Upload(context.Background(), &httpbody.HttpBody{
		ContentType: contentType,
		Data:        data,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetMessage() == "" {
		t.Fatal("resp message should not be empty")
	}
}

func TestUploadEmbed(t *testing.T) {
	uploadDir := t.TempDir()
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 58082, uploadDir)
	time.Sleep(1 * time.Second)

	client := newClient(58082)
	data, contentType := buildMultipartForm()
	resp, err := client.UploadEmbed(context.Background(), &UploadEmbedRequest{
		Body: &httpbody.HttpBody{
			ContentType: contentType,
			Data:        data,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetMessage() == "" {
		t.Fatal("resp message should not be empty")
	}
}

func TestUploadForRPC(t *testing.T) {
	uploadDir := t.TempDir()
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 58083, uploadDir)
	time.Sleep(1 * time.Second)

	client := newClient(58083)
	data, contentType := buildMultipartForm()
	resp, err := client.UploadForRPC(context.Background(), &rpchttp.HttpRequest{
		Headers: []*rpchttp.HttpHeader{
			{Key: "Content-Type", Value: contentType},
		},
		Body: data,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetMessage() == "" {
		t.Fatal("resp message should not be empty")
	}
}
