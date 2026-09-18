package upload

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestHandler creates a Handler that saves to a temporary directory.
func newTestHandler(t *testing.T, opts ...Option) (*Handler, string) {
	t.Helper()
	dir := t.TempDir()
	allOpts := append([]Option{WithUploadDir(dir)}, opts...)
	h, err := NewHandler(allOpts...)
	if err != nil {
		t.Fatalf("NewHandler failed: %v", err)
	}
	return h, dir
}

// buildMultipart constructs a multipart/form-data body from parts.
func buildMultipart(t *testing.T, parts []multipartPart) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, p := range parts {
		hdr := textproto.MIMEHeader{}
		if p.contentDisposition != "" {
			hdr.Set("Content-Disposition", p.contentDisposition)
		}
		if p.contentType != "" {
			hdr.Set("Content-Type", p.contentType)
		}
		part, err := w.CreatePart(hdr)
		if err != nil {
			t.Fatalf("CreatePart: %v", err)
		}
		part.Write(p.data)
	}
	w.Close()
	return buf.Bytes(), w.Boundary()
}

type multipartPart struct {
	contentDisposition string
	contentType        string
	data               []byte
}

// ---------------------------------------------------------------------------
// ExtensionFromContentType
// ---------------------------------------------------------------------------

func TestExtensionFromContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        string
	}{
		{"image/png", "image/png", ".png"},
		{"image/jpeg", "image/jpeg", ".jpg"},
		{"image/jpg", "image/jpg", ".jpg"},
		{"image/gif", "image/gif", ".gif"},
		{"image/webp", "image/webp", ".webp"},
		{"application/pdf", "application/pdf", ".pdf"},
		{"application/zip", "application/zip", ".zip"},
		{"application/gzip", "application/gzip", ".gz"},
		{"application/octet-stream", "application/octet-stream", ".bin"},
		{"text/plain", "text/plain", ".txt"},
		{"text/csv", "text/csv", ".csv"},
		{"text/html", "text/html", ".html"},
		{"application/json", "application/json", ".json"},
		{"application/xml", "application/xml", ".xml"},
		{"application/msword", "application/msword", ".doc"},
		{"docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", ".docx"},
		{"xls", "application/vnd.ms-excel", ".xls"},
		{"xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", ".xlsx"},
		{"unknown type", "application/x-unknown", ".bin"},
		{"with charset param", "image/png; charset=utf-8", ".png"},
		{"empty string", "", ".bin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtensionFromContentType(tt.contentType)
			if got != tt.want {
				t.Errorf("ExtensionFromContentType(%q) = %q, want %q", tt.contentType, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ExtensionFromFilename
// ---------------------------------------------------------------------------

func TestExtensionFromFilename(t *testing.T) {
	tests := []struct {
		name string
		file string
		want string
	}{
		{"with extension", "photo.jpg", ".jpg"},
		{"double extension", "archive.tar.gz", ".gz"},
		{"no extension", "README", ".bin"},
		{"empty string", "", ".bin"},
		{"hidden file", ".gitignore", ".gitignore"},
		{"pdf", "report.pdf", ".pdf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtensionFromFilename(tt.file)
			if got != tt.want {
				t.Errorf("ExtensionFromFilename(%q) = %q, want %q", tt.file, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FormatBytes
// ---------------------------------------------------------------------------

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name string
		b    int64
		want string
	}{
		{"zero", 0, "0 B"},
		{"bytes", 512, "512 B"},
		{"1 KB", 1024, "1.0 KB"},
		{"1.5 KB", 1536, "1.5 KB"},
		{"1 MB", 1024 * 1024, "1.0 MB"},
		{"2.5 GB", int64(2.5 * 1024 * 1024 * 1024), "2.5 GB"},
		{"1 TB", 1024 * 1024 * 1024 * 1024, "1.0 TB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatBytes(tt.b)
			if got != tt.want {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.b, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NewHandler / Options
// ---------------------------------------------------------------------------

func TestNewHandler_Defaults(t *testing.T) {
	h, err := NewHandler()
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	if h.cfg.UploadDir != "./uploads" {
		t.Errorf("default UploadDir = %q, want %q", h.cfg.UploadDir, "./uploads")
	}
	// Clean up the created directory
	os.Remove("./uploads")
}

func TestNewHandler_WithOptions(t *testing.T) {
	dir := t.TempDir()
	h, err := NewHandler(
		WithUploadDir(dir),
		WithMaxFileSize(1024),
		WithMaxTotalSize(2048),
	)
	if err != nil {
		t.Fatalf("NewHandler error = %v", err)
	}
	if h.cfg.UploadDir != dir {
		t.Errorf("UploadDir = %q, want %q", h.cfg.UploadDir, dir)
	}
	if h.cfg.MaxFileSize != 1024 {
		t.Errorf("MaxFileSize = %d, want 1024", h.cfg.MaxFileSize)
	}
	if h.cfg.MaxTotalSize != 2048 {
		t.Errorf("MaxTotalSize = %d, want 2048", h.cfg.MaxTotalSize)
	}
}

func TestNewHandler_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "upload_dir")
	h, err := NewHandler(WithUploadDir(dir))
	if err != nil {
		t.Fatalf("NewHandler error = %v", err)
	}
	info, err := os.Stat(h.cfg.UploadDir)
	if err != nil {
		t.Fatalf("upload directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("upload dir exists but is not a directory")
	}
}

// ---------------------------------------------------------------------------
// Result.JSON
// ---------------------------------------------------------------------------

func TestResult_JSON(t *testing.T) {
	r := &Result{
		Files: []SavedFile{
			{FieldName: "avatar", OrigName: "photo.jpg", ContentType: "image/jpeg", SavedAs: "123.jpg", Size: 1024, IsEmpty: false},
		},
		Fields: []FormField{
			{Name: "name", Values: []string{"alice"}},
		},
		TotalSize:  1024,
		FileCount:  1,
		FieldCount: 1,
	}
	s := r.JSON()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(s), &parsed); err != nil {
		t.Fatalf("JSON output is not valid JSON: %v\nraw: %s", err, s)
	}
	if int(parsed["file_count"].(float64)) != 1 {
		t.Errorf("file_count = %v, want 1", parsed["file_count"])
	}
}

// ---------------------------------------------------------------------------
// SaveSingleFile
// ---------------------------------------------------------------------------

func TestSaveSingleFile_Normal(t *testing.T) {
	h, dir := newTestHandler(t)
	data := []byte("hello world")
	f, err := h.SaveSingleFile(data, ".txt", "field1", "hello.txt", "text/plain")
	if err != nil {
		t.Fatalf("SaveSingleFile error = %v", err)
	}
	if f.FieldName != "field1" {
		t.Errorf("FieldName = %q, want %q", f.FieldName, "field1")
	}
	if f.OrigName != "hello.txt" {
		t.Errorf("OrigName = %q, want %q", f.OrigName, "hello.txt")
	}
	if f.Size != int64(len(data)) {
		t.Errorf("Size = %d, want %d", f.Size, len(data))
	}
	if f.IsEmpty {
		t.Error("IsEmpty = true, want false")
	}
	// Verify file was actually written to disk
	saved, err := os.ReadFile(filepath.Join(dir, f.SavedAs))
	if err != nil {
		t.Fatalf("reading saved file: %v", err)
	}
	if !bytes.Equal(saved, data) {
		t.Errorf("saved file content = %q, want %q", saved, data)
	}
}

func TestSaveSingleFile_EmptyFile(t *testing.T) {
	h, _ := newTestHandler(t)
	f, err := h.SaveSingleFile([]byte{}, ".txt", "", "", "text/plain")
	if err != nil {
		t.Fatalf("SaveSingleFile error = %v", err)
	}
	if !f.IsEmpty {
		t.Error("IsEmpty = false, want true for empty file")
	}
	if f.Size != 0 {
		t.Errorf("Size = %d, want 0", f.Size)
	}
}

func TestSaveSingleFile_ExceedsMaxFileSize(t *testing.T) {
	h, _ := newTestHandler(t, WithMaxFileSize(5))
	_, err := h.SaveSingleFile([]byte("too long data"), ".txt", "", "", "text/plain")
	if !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("expected ErrFileTooLarge, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Handle — raw body (single file)
// ---------------------------------------------------------------------------

func TestHandle_RawBody(t *testing.T) {
	h, dir := newTestHandler(t)
	data := []byte("raw file content")
	result, err := h.Handle(data, "text/plain")
	if err != nil {
		t.Fatalf("Handle error = %v", err)
	}
	if result.FileCount != 1 {
		t.Errorf("FileCount = %d, want 1", result.FileCount)
	}
	if result.FieldCount != 0 {
		t.Errorf("FieldCount = %d, want 0", result.FieldCount)
	}
	if result.TotalSize != int64(len(data)) {
		t.Errorf("TotalSize = %d, want %d", result.TotalSize, len(data))
	}
	// Verify file on disk
	saved, _ := os.ReadFile(filepath.Join(dir, result.Files[0].SavedAs))
	if !bytes.Equal(saved, data) {
		t.Errorf("saved content mismatch")
	}
}

func TestHandle_RawBody_ExceedsMaxTotalSize(t *testing.T) {
	h, _ := newTestHandler(t, WithMaxTotalSize(5))
	_, err := h.Handle([]byte("too much data"), "text/plain")
	if !errors.Is(err, ErrTotalTooLarge) {
		t.Errorf("expected ErrTotalTooLarge, got: %v", err)
	}
}

func TestHandle_MissingBoundary(t *testing.T) {
	h, _ := newTestHandler(t)
	// multipart type without boundary parameter
	_, err := h.Handle([]byte("data"), "multipart/form-data")
	if err == nil {
		t.Fatal("expected error for missing boundary, got nil")
	}
	if !strings.Contains(err.Error(), "missing boundary") {
		t.Errorf("error should mention 'missing boundary', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Handle — multipart/form-data
// ---------------------------------------------------------------------------

func TestHandle_Multipart_SingleFile(t *testing.T) {
	h, _ := newTestHandler(t)
	parts := []multipartPart{
		{
			contentDisposition: `form-data; name="file"; filename="test.txt"`,
			contentType:        "text/plain",
			data:               []byte("file content"),
		},
	}
	body, boundary := buildMultipart(t, parts)
	result, err := h.Handle(body, fmt.Sprintf("multipart/form-data; boundary=%s", boundary))
	if err != nil {
		t.Fatalf("Handle error = %v", err)
	}
	if result.FileCount != 1 {
		t.Errorf("FileCount = %d, want 1", result.FileCount)
	}
	if result.Files[0].OrigName != "test.txt" {
		t.Errorf("OrigName = %q, want %q", result.Files[0].OrigName, "test.txt")
	}
}

func TestHandle_Multipart_MixedFields(t *testing.T) {
	h, _ := newTestHandler(t)
	parts := []multipartPart{
		{
			contentDisposition: `form-data; name="file"; filename="photo.png"`,
			contentType:        "image/png",
			data:               []byte{0x89, 0x50, 0x4e, 0x47}, // PNG header stub
		},
		{
			contentDisposition: `form-data; name="description"`,
			data:               []byte("a nice photo"),
		},
		{
			contentDisposition: `form-data; name="tag"`,
			data:               []byte("nature"),
		},
	}
	body, boundary := buildMultipart(t, parts)
	result, err := h.Handle(body, fmt.Sprintf("multipart/form-data; boundary=%s", boundary))
	if err != nil {
		t.Fatalf("Handle error = %v", err)
	}
	if result.FileCount != 1 {
		t.Errorf("FileCount = %d, want 1", result.FileCount)
	}
	if result.FieldCount != 2 {
		t.Errorf("FieldCount = %d, want 2", result.FieldCount)
	}
	// Check field values
	var descField *FormField
	for i := range result.Fields {
		if result.Fields[i].Name == "description" {
			descField = &result.Fields[i]
			break
		}
	}
	if descField == nil || descField.Values[0] != "a nice photo" {
		t.Errorf("description field = %v, want 'a nice photo'", descField)
	}
}

// ---------------------------------------------------------------------------
// ParseMultipart
// ---------------------------------------------------------------------------

func TestParseMultipart_RepeatedFieldNames(t *testing.T) {
	h, _ := newTestHandler(t)
	parts := []multipartPart{
		{
			contentDisposition: `form-data; name="tag"`,
			data:               []byte("alpha"),
		},
		{
			contentDisposition: `form-data; name="tag"`,
			data:               []byte("beta"),
		},
		{
			contentDisposition: `form-data; name="tag"`,
			data:               []byte("gamma"),
		},
	}
	body, boundary := buildMultipart(t, parts)
	result, err := h.ParseMultipart(body, boundary)
	if err != nil {
		t.Fatalf("ParseMultipart error = %v", err)
	}
	if len(result.Fields) != 1 {
		t.Fatalf("expected 1 field (aggregated), got %d", len(result.Fields))
	}
	if len(result.Fields[0].Values) != 3 {
		t.Errorf("expected 3 values for 'tag', got %d", len(result.Fields[0].Values))
	}
	want := []string{"alpha", "beta", "gamma"}
	for i, v := range want {
		if result.Fields[0].Values[i] != v {
			t.Errorf("tag[%d] = %q, want %q", i, result.Fields[0].Values[i], v)
		}
	}
}

func TestParseMultipart_MultipleFiles(t *testing.T) {
	h, _ := newTestHandler(t)
	parts := []multipartPart{
		{
			contentDisposition: `form-data; name="files[]"; filename="a.txt"`,
			contentType:        "text/plain",
			data:               []byte("aaa"),
		},
		{
			contentDisposition: `form-data; name="files[]"; filename="b.txt"`,
			contentType:        "text/plain",
			data:               []byte("bbb"),
		},
	}
	body, boundary := buildMultipart(t, parts)
	result, err := h.ParseMultipart(body, boundary)
	if err != nil {
		t.Fatalf("ParseMultipart error = %v", err)
	}
	if result.FileCount != 2 {
		t.Errorf("FileCount = %d, want 2", result.FileCount)
	}
	if result.TotalSize != 6 {
		t.Errorf("TotalSize = %d, want 6", result.TotalSize)
	}
}

func TestParseMultipart_EmptyFile(t *testing.T) {
	h, _ := newTestHandler(t)
	parts := []multipartPart{
		{
			contentDisposition: `form-data; name="empty"; filename="empty.txt"`,
			contentType:        "text/plain",
			data:               []byte{},
		},
	}
	body, boundary := buildMultipart(t, parts)
	result, err := h.ParseMultipart(body, boundary)
	if err != nil {
		t.Fatalf("ParseMultipart error = %v", err)
	}
	if result.FileCount != 1 {
		t.Fatalf("FileCount = %d, want 1", result.FileCount)
	}
	if !result.Files[0].IsEmpty {
		t.Error("IsEmpty = false, want true for empty file")
	}
}

func TestParseMultipart_MaxFileSizeExceeded(t *testing.T) {
	h, _ := newTestHandler(t, WithMaxFileSize(5))
	parts := []multipartPart{
		{
			contentDisposition: `form-data; name="file"; filename="big.txt"`,
			contentType:        "text/plain",
			data:               []byte("this is too large"),
		},
	}
	body, boundary := buildMultipart(t, parts)
	_, err := h.ParseMultipart(body, boundary)
	if !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("expected ErrFileTooLarge, got: %v", err)
	}
}

func TestParseMultipart_MaxTotalSizeExceeded(t *testing.T) {
	h, _ := newTestHandler(t, WithMaxTotalSize(10))
	parts := []multipartPart{
		{
			contentDisposition: `form-data; name="field"`,
			data:               []byte("this body is larger than limit"),
		},
	}
	body, boundary := buildMultipart(t, parts)
	_, err := h.ParseMultipart(body, boundary)
	if !errors.Is(err, ErrTotalTooLarge) {
		t.Errorf("expected ErrTotalTooLarge, got: %v", err)
	}
}

func TestParseMultipart_ExtensionFallbackToFilename(t *testing.T) {
	h, _ := newTestHandler(t)
	// Use application/octet-stream so content-type extension is ".bin",
	// but the filename has ".csv" — the handler should prefer ".csv".
	parts := []multipartPart{
		{
			contentDisposition: `form-data; name="data"; filename="report.csv"`,
			contentType:        "application/octet-stream",
			data:               []byte("col1,col2\n1,2\n"),
		},
	}
	body, boundary := buildMultipart(t, parts)
	result, err := h.ParseMultipart(body, boundary)
	if err != nil {
		t.Fatalf("ParseMultipart error = %v", err)
	}
	if !strings.HasSuffix(result.Files[0].SavedAs, ".csv") {
		t.Errorf("SavedAs = %q, expected .csv extension from filename", result.Files[0].SavedAs)
	}
}

// ---------------------------------------------------------------------------
// addField (indirectly tested via ParseMultipart, but also directly)
// ---------------------------------------------------------------------------

func TestAddField_NewField(t *testing.T) {
	fields := addField(nil, "name", "alice")
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
	if fields[0].Name != "name" || fields[0].Values[0] != "alice" {
		t.Errorf("field = %+v, want name=[alice]", fields[0])
	}
}

func TestAddField_AppendExisting(t *testing.T) {
	fields := []FormField{{Name: "tag", Values: []string{"a"}}}
	fields = addField(fields, "tag", "b")
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
	if len(fields[0].Values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(fields[0].Values))
	}
	if fields[0].Values[1] != "b" {
		t.Errorf("Values[1] = %q, want %q", fields[0].Values[1], "b")
	}
}

func TestAddField_MultipleDistinctFields(t *testing.T) {
	fields := addField(nil, "a", "1")
	fields = addField(fields, "b", "2")
	fields = addField(fields, "a", "3")
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
	if len(fields[0].Values) != 2 {
		t.Errorf("field 'a' should have 2 values, got %d", len(fields[0].Values))
	}
}

// ---------------------------------------------------------------------------
// Handle — multipart/mixed
// ---------------------------------------------------------------------------

func TestHandle_MultipartMixed(t *testing.T) {
	h, _ := newTestHandler(t)
	parts := []multipartPart{
		{
			contentDisposition: `form-data; name="file"; filename="doc.pdf"`,
			contentType:        "application/pdf",
			data:               []byte("pdf content"),
		},
	}
	body, boundary := buildMultipart(t, parts)
	// Use multipart/mixed content type
	result, err := h.Handle(body, fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	if err != nil {
		t.Fatalf("Handle multipart/mixed error = %v", err)
	}
	if result.FileCount != 1 {
		t.Errorf("FileCount = %d, want 1", result.FileCount)
	}
}

// ---------------------------------------------------------------------------
// Handle — unknown content type (falls back to raw body)
// ---------------------------------------------------------------------------

func TestHandle_UnknownContentType(t *testing.T) {
	h, _ := newTestHandler(t)
	data := []byte("binary data")
	result, err := h.Handle(data, "application/x-custom")
	if err != nil {
		t.Fatalf("Handle error = %v", err)
	}
	if result.FileCount != 1 {
		t.Errorf("FileCount = %d, want 1", result.FileCount)
	}
	// Extension should be ".bin" for unknown types
	if !strings.HasSuffix(result.Files[0].SavedAs, ".bin") {
		t.Errorf("SavedAs = %q, expected .bin extension", result.Files[0].SavedAs)
	}
}

// ---------------------------------------------------------------------------
// Result.JSON — edge case: nil/empty result
// ---------------------------------------------------------------------------

func TestResult_JSON_Empty(t *testing.T) {
	r := &Result{}
	s := r.JSON()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(s), &parsed); err != nil {
		t.Fatalf("empty result JSON is not valid JSON: %v\nraw: %s", err, s)
	}
	if int(parsed["file_count"].(float64)) != 0 {
		t.Errorf("file_count = %v, want 0", parsed["file_count"])
	}
}

// ---------------------------------------------------------------------------
// Option functional options
// ---------------------------------------------------------------------------

func TestWithUploadDir(t *testing.T) {
	var cfg Options
	WithUploadDir("/tmp/custom")(&cfg)
	if cfg.UploadDir != "/tmp/custom" {
		t.Errorf("UploadDir = %q, want %q", cfg.UploadDir, "/tmp/custom")
	}
}

func TestWithMaxFileSize(t *testing.T) {
	var cfg Options
	WithMaxFileSize(1024 * 1024)(&cfg)
	if cfg.MaxFileSize != 1024*1024 {
		t.Errorf("MaxFileSize = %d, want %d", cfg.MaxFileSize, 1024*1024)
	}
}

func TestWithMaxTotalSize(t *testing.T) {
	var cfg Options
	WithMaxTotalSize(10 * 1024 * 1024)(&cfg)
	if cfg.MaxTotalSize != 10*1024*1024 {
		t.Errorf("MaxTotalSize = %d, want %d", cfg.MaxTotalSize, 10*1024*1024)
	}
}
