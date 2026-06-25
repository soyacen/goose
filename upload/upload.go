// Package upload provides full-featured multipart/form-data parsing and file saving.
//
// Features:
//   - File parts: saved to disk with content-type, filename, field-name metadata
//   - Form fields: collected with support for repeated names (array aggregation)
//   - Empty file parts: detected and marked with IsEmpty=true
//   - Size limits: per-file and total upload size enforcement
//   - Same field name multiple files: all saved independently (e.g. files[])
//   - Extension inference: from Content-Type or original filename
package upload

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"
)

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

var (
	// ErrFileTooLarge is returned when a single file exceeds MaxFileSize.
	ErrFileTooLarge = errors.New("file exceeds max file size limit")
	// ErrTotalTooLarge is returned when total upload size exceeds MaxTotalSize.
	ErrTotalTooLarge = errors.New("upload exceeds max total size limit")
)

// ---------------------------------------------------------------------------
// Data types
// ---------------------------------------------------------------------------

// SavedFile records a successfully saved file with full multipart metadata.
type SavedFile struct {
	FieldName   string `json:"field_name"`   // form field name (e.g. "files[]", "avatar")
	OrigName    string `json:"orig_name"`    // original filename from client
	ContentType string `json:"content_type"` // MIME type from part Content-Type header
	SavedAs     string `json:"saved_as"`     // filename on disk
	Size        int64  `json:"size"`         // bytes written
	IsEmpty     bool   `json:"is_empty"`     // true if uploaded file was 0 bytes
}

// FormField records a regular (non-file) form field.
// Same field name appearing multiple times → Values is a slice.
type FormField struct {
	Name   string   `json:"name"`
	Values []string `json:"values"` // supports repeated field names
}

// Result holds both files and form fields from a multipart request.
type Result struct {
	Files      []SavedFile `json:"files,omitempty"`
	Fields     []FormField `json:"fields,omitempty"`
	TotalSize  int64       `json:"total_size"`
	FileCount  int         `json:"file_count"`
	FieldCount int         `json:"field_count"`
}

// JSON returns the result serialized as a JSON string.
func (r *Result) JSON() string {
	b, err := json.Marshal(r)
	if err != nil {
		return fmt.Sprintf("{\"error\": %q}", err)
	}
	return string(b)
}

// ---------------------------------------------------------------------------
// Config & Options
// ---------------------------------------------------------------------------

// Options controls multipart parsing and file saving behavior.
type Options struct {
	UploadDir    string // directory to save uploaded files
	MaxFileSize  int64  // per-file limit in bytes (0 = unlimited)
	MaxTotalSize int64  // total upload limit in bytes (0 = unlimited)
}

// Option is a functional option that mutates a Config.
type Option func(*Options)

// WithUploadDir sets the directory where uploaded files are saved.
func WithUploadDir(dir string) Option {
	return func(c *Options) {
		c.UploadDir = dir
	}
}

// WithMaxFileSize sets the per-file size limit in bytes (0 = unlimited).
func WithMaxFileSize(n int64) Option {
	return func(c *Options) {
		c.MaxFileSize = n
	}
}

// WithMaxTotalSize sets the total upload size limit in bytes (0 = unlimited).
func WithMaxTotalSize(n int64) Option {
	return func(c *Options) {
		c.MaxTotalSize = n
	}
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// Handler processes multipart/form-data and saves files to disk.
type Handler struct {
	cfg Options
}

// NewHandler creates an UploadHandler, applying the given options.
// The upload directory is created automatically if it does not exist.
func NewHandler(opts ...Option) (*Handler, error) {
	cfg := Options{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.UploadDir == "" {
		cfg.UploadDir = "./uploads"
	}
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}
	return &Handler{cfg: cfg}, nil
}

// Handle is the main entry point. It auto-detects multipart vs raw body:
//   - multipart/form-data or multipart/mixed → ParseMultipart
//   - otherwise → SaveSingleFile
func (h *Handler) Handle(data []byte, contentType string) (*Result, error) {
	mediaType, params, _ := mime.ParseMediaType(contentType)

	if mediaType == "multipart/form-data" || mediaType == "multipart/mixed" {
		boundary := params["boundary"]
		if boundary == "" {
			return nil, fmt.Errorf("multipart missing boundary")
		}
		return h.ParseMultipart(data, boundary)
	}

	// Single file (raw body)
	if h.cfg.MaxTotalSize > 0 && int64(len(data)) > h.cfg.MaxTotalSize {
		return nil, fmt.Errorf("%w: %d > %d", ErrTotalTooLarge, len(data), h.cfg.MaxTotalSize)
	}
	ext := ExtensionFromContentType(contentType)
	f, err := h.SaveSingleFile(data, ext, "", "", contentType)
	if err != nil {
		return nil, err
	}
	return &Result{
		Files:     []SavedFile{f},
		TotalSize: f.Size,
		FileCount: 1,
	}, nil
}

// ParseMultipart parses multipart/form-data, saves file parts, and collects form fields.
func (h *Handler) ParseMultipart(data []byte, boundary string) (*Result, error) {
	if h.cfg.MaxTotalSize > 0 && int64(len(data)) > h.cfg.MaxTotalSize {
		return nil, fmt.Errorf("%w: %d > %d", ErrTotalTooLarge, len(data), h.cfg.MaxTotalSize)
	}

	reader := multipart.NewReader(bytes.NewReader(data), boundary)
	result := &Result{}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return result, fmt.Errorf("reading next part: %w", err)
		}

		// Read part data with optional per-file size limit
		var r io.Reader = part
		if h.cfg.MaxFileSize > 0 {
			r = io.LimitReader(part, h.cfg.MaxFileSize+1) // +1 to detect overflow
		}
		partData, err := io.ReadAll(r)
		if err != nil {
			return result, fmt.Errorf("reading part body: %w", err)
		}
		if h.cfg.MaxFileSize > 0 && int64(len(partData)) > h.cfg.MaxFileSize {
			return result, fmt.Errorf("%w: part %q has %d bytes (limit %d)",
				ErrFileTooLarge, part.FormName(), len(partData), h.cfg.MaxFileSize)
		}

		fieldName := part.FormName()
		origName := part.FileName()
		partContentType := part.Header.Get("Content-Type")

		if origName != "" {
			// File part — determine extension
			ext := ExtensionFromContentType(partContentType)
			if ext == ".bin" {
				ext = ExtensionFromFilename(origName)
			}
			f, err := h.SaveSingleFile(partData, ext, fieldName, origName, partContentType)
			if err != nil {
				return result, fmt.Errorf("saving file part: %w", err)
			}
			result.Files = append(result.Files, f)
			result.TotalSize += f.Size
			result.FileCount++
		} else {
			// Regular form field (aggregate same-name)
			result.Fields = addField(result.Fields, fieldName, string(partData))
			result.FieldCount++
		}
	}
	return result, nil
}

// SaveSingleFile writes data to a timestamped file on disk.
func (h *Handler) SaveSingleFile(data []byte, ext string, fieldName, origName, contentType string) (SavedFile, error) {
	if h.cfg.MaxFileSize > 0 && int64(len(data)) > h.cfg.MaxFileSize {
		return SavedFile{}, fmt.Errorf("%w: %d > %d", ErrFileTooLarge, len(data), h.cfg.MaxFileSize)
	}
	diskName := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	path := filepath.Join(h.cfg.UploadDir, diskName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return SavedFile{}, err
	}
	return SavedFile{
		FieldName:   fieldName,
		OrigName:    origName,
		ContentType: contentType,
		SavedAs:     diskName,
		Size:        int64(len(data)),
		IsEmpty:     len(data) == 0,
	}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// addField appends a value to an existing FormField or creates a new one.
func addField(fields []FormField, name, value string) []FormField {
	for i := range fields {
		if fields[i].Name == name {
			fields[i].Values = append(fields[i].Values, value)
			return fields
		}
	}
	return append(fields, FormField{Name: name, Values: []string{value}})
}

// ExtensionFromContentType returns a file extension for the given MIME content type.
func ExtensionFromContentType(contentType string) string {
	if mediaType, _, err := mime.ParseMediaType(contentType); err == nil {
		contentType = mediaType
	}
	switch contentType {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	case "application/zip":
		return ".zip"
	case "application/gzip":
		return ".gz"
	case "application/octet-stream":
		return ".bin"
	case "text/plain":
		return ".txt"
	case "text/csv":
		return ".csv"
	case "text/html":
		return ".html"
	case "application/json":
		return ".json"
	case "application/xml":
		return ".xml"
	case "application/msword":
		return ".doc"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ".docx"
	case "application/vnd.ms-excel":
		return ".xls"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return ".xlsx"
	default:
		return ".bin"
	}
}

// ExtensionFromFilename returns the file extension from a filename.
func ExtensionFromFilename(name string) string {
	ext := filepath.Ext(name)
	if ext != "" {
		return ext
	}
	return ".bin"
}

// FormatBytes returns a human-readable byte size string.
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
