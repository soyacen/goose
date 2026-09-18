package openapi

import (
	"fmt"

	"github.com/soyacen/goose/cmd/protoc-gen-goose/parser"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// SchemaCollector collects and generates OpenAPI schemas for protobuf messages.
type SchemaCollector struct {
	schemas map[string]*Schema
	visited map[string]bool
}

// NewSchemaCollector creates a new SchemaCollector.
func NewSchemaCollector() *SchemaCollector {
	return &SchemaCollector{
		schemas: make(map[string]*Schema),
		visited: make(map[string]bool),
	}
}

// Collect collects schemas for all messages referenced by the given services.
func (c *SchemaCollector) Collect(services []*parser.Service) map[string]*Schema {
	for _, service := range services {
		for _, endpoint := range service.Endpoints {
			// Collect request message schema
			if endpoint.Input() != nil {
				c.collectMessage(endpoint.Input())
			}
			// Collect response message schema
			if endpoint.Output() != nil {
				c.collectMessage(endpoint.Output())
			}
		}
	}
	return c.schemas
}

// schemaName returns the OpenAPI schema reference name for a message.
func schemaName(msg *protogen.Message) string {
	return string(msg.Desc.FullName())
}

// schemaRef returns the $ref string for a message schema.
func schemaRef(msg *protogen.Message) string {
	return fmt.Sprintf("#/components/schemas/%s", schemaName(msg))
}

// collectMessage recursively collects a message and all its nested messages.
func (c *SchemaCollector) collectMessage(msg *protogen.Message) {
	if msg == nil {
		return
	}
	name := schemaName(msg)
	if c.visited[name] {
		return
	}
	c.visited[name] = true

	// Skip well-known types - they don't need schema definitions
	if isWellKnownType(msg.Desc.FullName()) {
		return
	}

	// Generate schema for this message
	schema := c.generateMessageSchema(msg)
	c.schemas[name] = schema
}

// generateMessageSchema generates an OpenAPI Schema for a single protobuf message.
func (c *SchemaCollector) generateMessageSchema(msg *protogen.Message) *Schema {
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	for _, field := range msg.Fields {
		fieldSchema := c.fieldToSchema(field)
		schema.Properties[field.Desc.JSONName()] = fieldSchema

		// Add to required if not optional and not a list/map
		if !field.Desc.HasPresence() && !field.Desc.IsList() && !field.Desc.IsMap() {
			schema.Required = append(schema.Required, field.Desc.JSONName())
		}
	}

	return schema
}

// fieldToSchema converts a protobuf field to an OpenAPI Schema,
// recursively collecting nested message types.
func (c *SchemaCollector) fieldToSchema(field *protogen.Field) *Schema {
	// For message types (non-well-known), collect the nested message first
	if field.Desc.Kind() == protoreflect.MessageKind || field.Desc.Kind() == protoreflect.GroupKind {
		msg := field.Message
		if msg != nil && !isWellKnownType(msg.Desc.FullName()) {
			c.collectMessage(msg)
		}
	}

	return protoFieldToSchema(field, func(msg *protogen.Message) string {
		if isWellKnownType(msg.Desc.FullName()) {
			return ""
		}
		return schemaRef(msg)
	})
}

// GetSchema returns a Schema reference for a message.
// If the message is a well-known type, returns the inline schema.
// Otherwise, returns a $ref to the components/schemas entry.
func GetSchema(msg *protogen.Message) *Schema {
	if msg == nil {
		return &Schema{Type: "object"}
	}
	if isWellKnownType(msg.Desc.FullName()) {
		return wellKnownTypeToSchema(msg.Desc.FullName())
	}
	return &Schema{Ref: schemaRef(msg)}
}

// GetFieldSchema returns a Schema for a specific field within a message.
func GetFieldSchema(field *protogen.Field) *Schema {
	return protoFieldToSchema(field, func(msg *protogen.Message) string {
		if isWellKnownType(msg.Desc.FullName()) {
			return ""
		}
		return schemaRef(msg)
	})
}
