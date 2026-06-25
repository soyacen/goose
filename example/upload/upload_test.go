package upload

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/soyacen/goose/upload"
	"google.golang.org/genproto/googleapis/api/httpbody"
	rpchttp "google.golang.org/genproto/googleapis/rpc/http"
)

// UploadServiceImpl implements upload.UploadService using multipartx.Handler.
type UploadServiceImpl struct {
	handler *upload.Handler
}

// Upload handles PUT /v1/upload/api — receives raw HttpBody (body: "*")
func (s *UploadServiceImpl) Upload(ctx context.Context, req *httpbody.HttpBody) (*Response, error) {
	result, err := s.handler.Handle(req.GetData(), req.GetContentType())
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	log.Printf("[Upload]          %d file(s), %d field(s), %d bytes total", result.FileCount, result.FieldCount, result.TotalSize)
	return &Response{Message: result.JSON()}, nil
}

// UploadEmbed handles PUT /v1/upload/embd — receives HttpBodyRequest with embedded HttpBody (body: "body")
func (s *UploadServiceImpl) UploadEmbed(ctx context.Context, req *HttpBodyRequest) (*Response, error) {
	body := req.GetBody()
	result, err := s.handler.Handle(body.GetData(), body.GetContentType())
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	log.Printf("[UploadEmbed]     %d file(s), %d field(s), %d bytes total", result.FileCount, result.FieldCount, result.TotalSize)
	return &Response{Message: result.JSON()}, nil
}

// UploadForRPC handles PUT /v1/upload/rpc — receives google.rpc.HttpRequest (body: "*")
func (s *UploadServiceImpl) UploadForRPC(ctx context.Context, req *rpchttp.HttpRequest) (*Response, error) {
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
	log.Printf("[UploadForRPC]    %d file(s), %d field(s), %d bytes total", result.FileCount, result.FieldCount, result.TotalSize)
	return &Response{Message: result.JSON()}, nil
}

func main() {
	port := flag.Int("port", 8080, "server listen port")
	uploadDir := flag.String("dir", "./uploads", "directory to save uploaded files")
	maxFileSize := flag.Int64("max-file-size", 32<<20, "max size per file in bytes (default 32MB)")
	maxTotalSize := flag.Int64("max-total-size", 128<<20, "max total upload size in bytes (default 128MB)")
	flag.Parse()

	handler, err := upload.NewHandler(
		upload.WithUploadDir(*uploadDir),
		upload.WithMaxFileSize(*maxFileSize),
		upload.WithMaxTotalSize(*maxTotalSize),
	)
	if err != nil {
		log.Fatalf("failed to init multipart handler: %v", err)
	}

	service := &UploadServiceImpl{handler: handler}
	router := http.NewServeMux()
	router = AppendUploadHttpRoute(router, service)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Upload server starting on %s", addr)
	log.Printf("  PUT /v1/upload/api   — single / multi-file / mixed fields (multipart/form-data)")
	log.Printf("  PUT /v1/upload/embd  — single / multi-file / mixed fields (embedded HttpBody)")
	log.Printf("  PUT /v1/upload/rpc   — single / multi-file / mixed fields (RPC HttpRequest)")
	log.Printf("  Upload directory : %s", *uploadDir)
	log.Printf("  Max file size    : %s", upload.FormatBytes(*maxFileSize))
	log.Printf("  Max total size   : %s", upload.FormatBytes(*maxTotalSize))

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
