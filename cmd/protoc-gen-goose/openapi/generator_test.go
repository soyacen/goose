package openapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/v1/user/{id}", "/v1/user/{id}"},
		{"/v1/user/{$}", "/v1/user/"},
		{"/v1/files/{rest...}", "/v1/files/{rest...}"},
		{"/", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizePath(tt.input)
			if got != tt.want {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSetOperation(t *testing.T) {
	item := &PathItem{}
	ops := map[string]*Operation{
		"Get":     {OperationID: "get-op"},
		"Post":    {OperationID: "post-op"},
		"Put":     {OperationID: "put-op"},
		"Delete":  {OperationID: "delete-op"},
		"Patch":   {OperationID: "patch-op"},
		"Head":    {OperationID: "head-op"},
		"Options": {OperationID: "options-op"},
	}

	setOperation(item, http.MethodGet, ops["Get"])
	setOperation(item, http.MethodPost, ops["Post"])
	setOperation(item, http.MethodPut, ops["Put"])
	setOperation(item, http.MethodDelete, ops["Delete"])
	setOperation(item, http.MethodPatch, ops["Patch"])
	setOperation(item, http.MethodHead, ops["Head"])
	setOperation(item, http.MethodOptions, ops["Options"])

	if item.Get != ops["Get"] {
		t.Error("Get not set correctly")
	}
	if item.Post != ops["Post"] {
		t.Error("Post not set correctly")
	}
	if item.Put != ops["Put"] {
		t.Error("Put not set correctly")
	}
	if item.Delete != ops["Delete"] {
		t.Error("Delete not set correctly")
	}
	if item.Patch != ops["Patch"] {
		t.Error("Patch not set correctly")
	}
	if item.Head != ops["Head"] {
		t.Error("Head not set correctly")
	}
	if item.Options != ops["Options"] {
		t.Error("Options not set correctly")
	}
}

func TestMergePathItem(t *testing.T) {
	dest := &PathItem{
		Get:  &Operation{OperationID: "original-get"},
		Post: &Operation{OperationID: "original-post"},
	}
	src := &PathItem{
		Post:   &Operation{OperationID: "new-post"},
		Delete: &Operation{OperationID: "new-delete"},
	}

	mergePathItem(dest, src)

	if dest.Get.OperationID != "original-get" {
		t.Error("Get should remain unchanged")
	}
	if dest.Post.OperationID != "new-post" {
		t.Error("Post should be overwritten")
	}
	if dest.Delete == nil || dest.Delete.OperationID != "new-delete" {
		t.Error("Delete should be added from src")
	}
}

func TestDocumentJSONMarshal(t *testing.T) {
	doc := &Document{
		OpenAPI: "3.0.3",
		Info: &Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]*PathItem{
			"/test": {
				Get: &Operation{
					OperationID: "Test_Get",
					Responses: map[string]*Response{
						"200": {
							Description: "Success",
							Content: map[string]*MediaType{
								"application/json": {
									Schema: &Schema{Type: "object"},
								},
							},
						},
					},
				},
			},
		},
		Components: &Components{
			Schemas: map[string]*Schema{
				"TestMessage": {
					Type: "object",
					Properties: map[string]*Schema{
						"id": {Type: "integer", Format: "int64"},
					},
					Required: []string{"id"},
				},
			},
		},
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("marshaled document is empty")
	}

	str := string(data)
	if str == "" {
		t.Error("marshaled document string is empty")
	}
}
