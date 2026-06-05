package openapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/soyacen/goose/cmd/protoc-gen-goose/parser"
	"google.golang.org/protobuf/compiler/protogen"
)

// Generate creates an OpenAPI document for the given proto file and services,
// writing it as a generated JSON file.
func Generate(plugin *protogen.Plugin, file *protogen.File, services []*parser.Service) error {
	// Collect all schemas
	collector := NewSchemaCollector()
	schemas := collector.Collect(services)

	// Generate paths
	paths := make(map[string]*PathItem)
	for _, service := range services {
		servicePaths := GeneratePaths(service)
		for path, item := range servicePaths {
			if existing, ok := paths[path]; ok {
				mergePathItem(existing, item)
			} else {
				paths[path] = item
			}
		}
	}

	// Build document
	doc := &Document{
		OpenAPI: "3.0.3",
		Info: &Info{
			Title:   fmt.Sprintf("%s API", file.Desc.Package()),
			Version: "1.0.0",
		},
		Paths: paths,
	}

	if len(schemas) > 0 {
		doc.Components = &Components{
			Schemas: schemas,
		}
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal OpenAPI document: %w", err)
	}

	// Write generated file
	filename := file.GeneratedFilenamePrefix + "_goose.openapi.json"
	g := plugin.NewGeneratedFile(filename, "")
	g.Write(data)

	return nil
}

// GeneratePaths generates OpenAPI path items for all endpoints in a service.
func GeneratePaths(service *parser.Service) map[string]*PathItem {
	paths := make(map[string]*PathItem)
	for _, endpoint := range service.Endpoints {
		path := normalizePath(endpoint.Path())
		if _, ok := paths[path]; !ok {
			paths[path] = &PathItem{}
		}

		operation := generateOperation(endpoint)
		setOperation(paths[path], endpoint.Method(), operation)
	}
	return paths
}

// normalizePath converts goose path patterns to OpenAPI compatible paths.
func normalizePath(path string) string {
	// Go 1.22 ServeMux uses {$} for matching trailing slash; OpenAPI doesn't need this.
	path = strings.ReplaceAll(path, "{$}", "")
	return path
}

// generateOperation creates an OpenAPI Operation for a single endpoint.
func generateOperation(endpoint *parser.Endpoint) *Operation {
	operation := &Operation{
		OperationID: fmt.Sprintf("%s_%s", endpoint.Input().GoIdent.GoName, endpoint.Name()),
		Responses:   make(map[string]*Response),
	}

	// Extract summary from proto comments if available
	if endpoint.Input() != nil {
		operation.Summary = fmt.Sprintf("%s", endpoint.Name())
	}

	// Parse parameters (path and query)
	_, _, pathFields, queryFields, _ := endpoint.ParseParameters()

	// Path parameters
	for _, field := range pathFields {
		param := &Parameter{
			Name:     field.Desc.JSONName(),
			In:       "path",
			Required: true,
			Schema:   GetFieldSchema(field),
		}
		operation.Parameters = append(operation.Parameters, param)
	}

	// Query parameters
	for _, field := range queryFields {
		param := &Parameter{
			Name:   field.Desc.JSONName(),
			In:     "query",
			Schema: GetFieldSchema(field),
		}
		operation.Parameters = append(operation.Parameters, param)
	}

	// Request body
	if body := generateRequestBody(endpoint); body != nil {
		operation.RequestBody = body
	}

	// Response
	operation.Responses = generateResponses(endpoint)

	return operation
}

// generateRequestBody creates a RequestBody for an endpoint if applicable.
func generateRequestBody(endpoint *parser.Endpoint) *RequestBody {
	method := endpoint.Method()
	// GET, HEAD, DELETE typically don't have request bodies
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodDelete {
		return nil
	}

	bodyParam := endpoint.Body()
	if bodyParam == "" {
		return nil
	}

	var schema *Schema
	if bodyParam == "*" {
		schema = GetSchema(endpoint.Input())
	} else {
		// Named body field
		field := parser.FindField(bodyParam, endpoint.Input())
		if field != nil {
			schema = GetFieldSchema(field)
		}
	}

	if schema == nil {
		return nil
	}

	return &RequestBody{
		Content: map[string]*MediaType{
			"application/json": {Schema: schema},
		},
	}
}

// generateResponses creates response definitions for an endpoint.
func generateResponses(endpoint *parser.Endpoint) map[string]*Response {
	responses := make(map[string]*Response)

	// Determine success status code
	statusCode := "200"
	switch endpoint.Method() {
	case http.MethodPost:
		statusCode = "201"
	case http.MethodDelete:
		statusCode = "204"
	}

	var schema *Schema
	responseBody := endpoint.ResponseBody()
	if responseBody == "" || responseBody == "*" {
		schema = GetSchema(endpoint.Output())
	} else {
		field := parser.FindField(responseBody, endpoint.Output())
		if field != nil {
			schema = GetFieldSchema(field)
		}
	}

	description := "Success"
	if statusCode == "201" {
		description = "Created"
	} else if statusCode == "204" {
		description = "No Content"
	}

	resp := &Response{
		Description: description,
	}
	if schema != nil {
		resp.Content = map[string]*MediaType{
			"application/json": {Schema: schema},
		}
	}
	responses[statusCode] = resp

	// Add default error response
	responses["default"] = &Response{
		Description: "Error response",
	}

	return responses
}

// setOperation sets the operation on the path item based on HTTP method.
func setOperation(item *PathItem, method string, operation *Operation) {
	switch method {
	case http.MethodGet:
		item.Get = operation
	case http.MethodPost:
		item.Post = operation
	case http.MethodPut:
		item.Put = operation
	case http.MethodDelete:
		item.Delete = operation
	case http.MethodPatch:
		item.Patch = operation
	case http.MethodHead:
		item.Head = operation
	case http.MethodOptions:
		item.Options = operation
	}
}

// mergePathItem merges operations from source into dest.
func mergePathItem(dest, src *PathItem) {
	if src.Get != nil {
		dest.Get = src.Get
	}
	if src.Post != nil {
		dest.Post = src.Post
	}
	if src.Put != nil {
		dest.Put = src.Put
	}
	if src.Delete != nil {
		dest.Delete = src.Delete
	}
	if src.Patch != nil {
		dest.Patch = src.Patch
	}
	if src.Head != nil {
		dest.Head = src.Head
	}
	if src.Options != nil {
		dest.Options = src.Options
	}
}
