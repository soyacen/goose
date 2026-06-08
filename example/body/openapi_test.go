package body

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ---------- Lightweight OpenAPI JSON structs ----------

type openAPIDoc struct {
	OpenAPI    string                 `json:"openapi"`
	Info       *oaInfo                `json:"info"`
	Paths      map[string]*oaPathItem `json:"paths"`
	Components *oaComponents          `json:"components,omitempty"`
}

type oaInfo struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type oaPathItem struct {
	Get     *oaOperation `json:"get,omitempty"`
	Post    *oaOperation `json:"post,omitempty"`
	Put     *oaOperation `json:"put,omitempty"`
	Delete  *oaOperation `json:"delete,omitempty"`
	Patch   *oaOperation `json:"patch,omitempty"`
	Head    *oaOperation `json:"head,omitempty"`
	Options *oaOperation `json:"options,omitempty"`
}

type oaOperation struct {
	OperationID string                `json:"operationId"`
	Summary     string                `json:"summary,omitempty"`
	Parameters  []oaParameter         `json:"parameters,omitempty"`
	RequestBody *oaRequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]oaResponse `json:"responses"`
}

type oaParameter struct {
	Name     string    `json:"name"`
	In       string    `json:"in"`
	Required bool      `json:"required,omitempty"`
	Schema   *oaSchema `json:"schema,omitempty"`
}

type oaRequestBody struct {
	Content map[string]oaMediaType `json:"content"`
}

type oaMediaType struct {
	Schema *oaSchema `json:"schema,omitempty"`
}

type oaResponse struct {
	Description string                 `json:"description"`
	Content     map[string]oaMediaType `json:"content,omitempty"`
}

type oaSchema struct {
	Type                 string            `json:"type,omitempty"`
	Format               string            `json:"format,omitempty"`
	Properties           map[string]*oaSchema `json:"properties,omitempty"`
	Items                *oaSchema           `json:"items,omitempty"`
	Required             []string            `json:"required,omitempty"`
	Ref                  string              `json:"$ref,omitempty"`
	Enum                 []string            `json:"enum,omitempty"`
	Nullable             bool                `json:"nullable,omitempty"`
	AdditionalProperties *oaSchema           `json:"additionalProperties,omitempty"`
}

type oaComponents struct {
	Schemas map[string]*oaSchema `json:"schemas,omitempty"`
}

// ---------- Example data generation ----------

func exampleValue(fieldName string, schema *oaSchema, components *oaComponents) any {
	if schema == nil {
		return nil
	}

	if schema.Ref != "" {
		resolved := resolveRef(schema.Ref, components)
		if resolved != nil {
			return exampleValue(fieldName, resolved, components)
		}
		return nil
	}

	switch schema.Type {
	case "string":
		return exampleString(fieldName, schema)
	case "integer":
		return exampleInt(fieldName, schema)
	case "number":
		return exampleNumber(fieldName, schema)
	case "boolean":
		return exampleBool(fieldName, schema)
	case "array":
		if schema.Items != nil {
			return []any{exampleValue("item", schema.Items, components)}
		}
		return []any{}
	case "object":
		if schema.AdditionalProperties != nil {
			return map[string]any{"key": exampleValue("value", schema.AdditionalProperties, components)}
		}
		return exampleObject(schema, components)
	default:
		return nil
	}
}

func exampleString(fieldName string, schema *oaSchema) string {
	if len(schema.Enum) > 0 {
		return schema.Enum[0]
	}

	switch {
	case strings.Contains(fieldName, "name"):
		return "test_name"
	case strings.Contains(fieldName, "email"):
		return "test@example.com"
	case strings.Contains(fieldName, "url"):
		return "https://example.com"
	case strings.Contains(fieldName, "contentType") || strings.Contains(fieldName, "content_type"):
		return "application/json"
	case schema.Format == "date-time":
		return "2024-01-01T00:00:00Z"
	case schema.Format == "byte" || schema.Format == "binary":
		return "dGVzdA=="
	default:
		return "test"
	}
}

func exampleInt(fieldName string, _ *oaSchema) int64 {
	switch {
	case strings.Contains(fieldName, "pageNum") || strings.Contains(fieldName, "page_num"):
		return 1
	case strings.Contains(fieldName, "pageSize") || strings.Contains(fieldName, "page_size"):
		return 10
	case strings.Contains(fieldName, "id"):
		return 1
	case strings.Contains(fieldName, "count") || strings.Contains(fieldName, "total"):
		return 100
	default:
		return 42
	}
}

func exampleNumber(_ string, _ *oaSchema) float64 {
	return 3.14
}

func exampleBool(_ string, _ *oaSchema) bool {
	return true
}

func exampleObject(schema *oaSchema, components *oaComponents) map[string]any {
	obj := make(map[string]any)
	for propName, propSchema := range schema.Properties {
		obj[propName] = exampleValue(propName, propSchema, components)
	}
	return obj
}

func resolveRef(ref string, components *oaComponents) *oaSchema {
	if components == nil || components.Schemas == nil {
		return nil
	}
	prefix := "#/components/schemas/"
	if strings.HasPrefix(ref, prefix) {
		name := ref[len(prefix):]
		return components.Schemas[name]
	}
	return nil
}

// ---------- Request building ----------

func buildRequest(baseURL, path string, op *oaOperation, components *oaComponents) (*http.Request, string, error) {
	urlPath := buildURLPath(path, op.Parameters)
	fullURL := baseURL + urlPath

	queryStr := buildQueryString(op.Parameters)
	if queryStr != "" {
		fullURL += "?" + queryStr
	}

	var bodyReader io.Reader
	var contentType string
	if op.RequestBody != nil {
		for ct, media := range op.RequestBody.Content {
			contentType = ct
			if media.Schema != nil {
				data := exampleValue("body", media.Schema, components)
				jsonData, err := json.Marshal(data)
				if err != nil {
					return nil, "", fmt.Errorf("marshal request body: %w", err)
				}
				bodyReader = bytes.NewReader(jsonData)
			}
			break
		}
	}

	method := "GET"
	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, "", err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req, contentType, nil
}

func buildURLPath(path string, params []oaParameter) string {
	result := path
	for _, p := range params {
		if p.In == "path" {
			placeholder := "{" + p.Name + "}"
			value := pathParamExample(p.Name, p.Schema)
			result = strings.ReplaceAll(result, placeholder, value)
		}
	}
	return result
}

func pathParamExample(name string, schema *oaSchema) string {
	if schema == nil {
		return "1"
	}
	switch schema.Type {
	case "string":
		return exampleString(name, schema)
	case "integer", "number":
		return strconv.FormatInt(exampleInt(name, schema), 10)
	case "boolean":
		return strconv.FormatBool(exampleBool(name, schema))
	default:
		return "1"
	}
}

func buildQueryString(params []oaParameter) string {
	var parts []string
	for _, p := range params {
		if p.In != "query" {
			continue
		}
		value := queryParamExample(p.Name, p.Schema)
		if value != "" {
			parts = append(parts, fmt.Sprintf("%s=%s", p.Name, value))
		}
	}
	return strings.Join(parts, "&")
}

func queryParamExample(name string, schema *oaSchema) string {
	if schema == nil {
		return ""
	}
	switch schema.Type {
	case "string":
		return exampleString(name, schema)
	case "integer", "number":
		return strconv.FormatInt(exampleInt(name, schema), 10)
	case "boolean":
		return strconv.FormatBool(exampleBool(name, schema))
	case "array":
		if schema.Items != nil {
			return queryParamExample(name, schema.Items)
		}
		return ""
	default:
		return ""
	}
}

// ---------- Response validation ----------

func validateResponse(resp *http.Response, op *oaOperation) error {
	statusCode := strconv.Itoa(resp.StatusCode)

	respSpec, ok := op.Responses[statusCode]
	if !ok {
		respSpec, ok = op.Responses["default"]
		if !ok {
			return fmt.Errorf("unexpected status code %s, not documented in OpenAPI spec", statusCode)
		}
	}

	if resp.StatusCode == http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		if len(body) > 0 {
			return fmt.Errorf("expected empty body for 204 response, got %d bytes", len(body))
		}
		return nil
	}

	if len(respSpec.Content) > 0 {
		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			return fmt.Errorf("missing Content-Type header in response")
		}
		if !strings.Contains(contentType, "application/json") {
			return fmt.Errorf("expected application/json Content-Type, got %s", contentType)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read response body: %w", err)
		}
		var dummy any
		if err := json.Unmarshal(body, &dummy); err != nil {
			return fmt.Errorf("response body is not valid JSON: %w", err)
		}
	}

	return nil
}

// ---------- Test entry point ----------

func getOpenAPIPath() string {
	_, b, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(b)
	return filepath.Join(baseDir, "body_goose.openapi.json")
}

func loadOpenAPIDoc(t *testing.T) *openAPIDoc {
	path := getOpenAPIPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read OpenAPI document at %s: %v", path, err)
	}

	var doc openAPIDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("failed to parse OpenAPI document: %v", err)
	}

	return &doc
}

func TestOpenAPISpec(t *testing.T) {
	doc := loadOpenAPIDoc(t)
	if len(doc.Paths) == 0 {
		t.Fatal("OpenAPI document has no paths")
	}

	port := 38090
	server := &http.Server{}
	defer server.Shutdown(context.Background())
	go func() {
		router := http.NewServeMux()
		router = AppendBodyHttpRoute(router, &MockBodyService{})
		server.Addr = fmt.Sprintf(":%d", port)
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	time.Sleep(500 * time.Millisecond)

	baseURL := fmt.Sprintf("http://localhost:%d", port)
	client := &http.Client{Timeout: 5 * time.Second}

	for path, item := range doc.Paths {
		ops := map[string]*oaOperation{
			"GET":     item.Get,
			"POST":    item.Post,
			"PUT":     item.Put,
			"DELETE":  item.Delete,
			"PATCH":   item.Patch,
			"HEAD":    item.Head,
			"OPTIONS": item.Options,
		}

		for method, op := range ops {
			if op == nil {
				continue
			}

			t.Run(op.OperationID, func(t *testing.T) {
				req, _, err := buildRequest(baseURL, path, op, doc.Components)
				if err != nil {
					t.Fatalf("build request: %v", err)
				}
				req.Method = method

				resp, err := client.Do(req)
				if err != nil {
					t.Fatalf("HTTP request failed: %v", err)
				}
				defer resp.Body.Close()

				if err := validateResponse(resp, op); err != nil {
					t.Fatalf("response validation failed: %v", err)
				}

				t.Logf("✓ %s %s -> %d", method, req.URL.Path, resp.StatusCode)
			})
		}
	}
}

func TestOpenAPIContent(t *testing.T) {
	doc := loadOpenAPIDoc(t)

	requiredPaths := map[string][]string{
		"/v1/star/body":            {"POST"},
		"/v1/named/body":           {"POST"},
		"/v1/user_body":            {"GET"},
		"/v1/http/body/star/body":  {"PUT"},
		"/v1/http/body/named/body": {"PUT"},
		"/v1/http/request":         {"PUT"},
	}

	for requiredPath, methods := range requiredPaths {
		item, ok := doc.Paths[requiredPath]
		if !ok {
			t.Errorf("missing required path: %s", requiredPath)
			continue
		}

		ops := map[string]*oaOperation{
			"GET":  item.Get,
			"POST": item.Post,
			"PUT":  item.Put,
		}

		for _, method := range methods {
			if ops[method] == nil {
				t.Errorf("path %s missing %s operation", requiredPath, method)
			}
		}
	}

	if doc.Components == nil || doc.Components.Schemas == nil {
		t.Fatal("missing components/schemas")
	}

	requiredSchemas := []string{
		"leo.goose.example.body.v1.BodyRequest",
		"leo.goose.example.body.v1.Response",
		"leo.goose.example.body.v1.NamedBodyRequest.Body",
	}
	for _, name := range requiredSchemas {
		if _, ok := doc.Components.Schemas[name]; !ok {
			t.Errorf("missing required schema: %s", name)
		}
	}

	if doc.Paths["/v1/star/body"].Post == nil || doc.Paths["/v1/star/body"].Post.RequestBody == nil {
		t.Error("POST /v1/star/body missing requestBody")
	}

	if doc.Paths["/v1/named/body"].Post == nil || doc.Paths["/v1/named/body"].Post.RequestBody == nil {
		t.Error("POST /v1/named/body missing requestBody")
	}
}
