package openapi

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestWellKnownTypeToSchema(t *testing.T) {
	tests := []struct {
		name     protoreflect.FullName
		wantType string
		wantFmt  string
		nullable bool
	}{
		{"google.protobuf.Timestamp", "string", "date-time", false},
		{"google.protobuf.Duration", "string", "", false},
		{"google.protobuf.Empty", "object", "", false},
		{"google.protobuf.StringValue", "string", "", true},
		{"google.protobuf.Int32Value", "integer", "int32", true},
		{"google.protobuf.Int64Value", "integer", "int64", true},
		{"google.protobuf.BoolValue", "boolean", "", true},
		{"google.protobuf.FloatValue", "number", "float", true},
		{"google.protobuf.DoubleValue", "number", "double", true},
		{"google.protobuf.UInt32Value", "integer", "int32", true},
		{"google.protobuf.UInt64Value", "integer", "int64", true},
		{"google.api.HttpBody", "string", "binary", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			got := wellKnownTypeToSchema(tt.name)
			if got == nil {
				t.Fatalf("wellKnownTypeToSchema(%q) = nil, want non-nil", tt.name)
			}
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
			if got.Format != tt.wantFmt {
				t.Errorf("Format = %q, want %q", got.Format, tt.wantFmt)
			}
			if got.Nullable != tt.nullable {
				t.Errorf("Nullable = %v, want %v", got.Nullable, tt.nullable)
			}
		})
	}
}

func TestWellKnownTypeToSchema_Unknown(t *testing.T) {
	got := wellKnownTypeToSchema("unknown.Type")
	if got != nil {
		t.Errorf("wellKnownTypeToSchema(\"unknown.Type\") = %+v, want nil", got)
	}
}

func TestIsWellKnownType(t *testing.T) {
	if !isWellKnownType("google.protobuf.Timestamp") {
		t.Error("isWellKnownType(google.protobuf.Timestamp) = false, want true")
	}
	if isWellKnownType("my.custom.Message") {
		t.Error("isWellKnownType(my.custom.Message) = true, want false")
	}
}
