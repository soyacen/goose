package openapi

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// isWellKnownType returns true if the message is a well-known protobuf type
// that has special OpenAPI representation.
func isWellKnownType(name protoreflect.FullName) bool {
	switch name {
	case "google.protobuf.Timestamp",
		"google.protobuf.Duration",
		"google.protobuf.Empty",
		"google.protobuf.StringValue",
		"google.protobuf.Int32Value",
		"google.protobuf.Int64Value",
		"google.protobuf.UInt32Value",
		"google.protobuf.UInt64Value",
		"google.protobuf.BoolValue",
		"google.protobuf.FloatValue",
		"google.protobuf.DoubleValue",
		"google.protobuf.BytesValue",
		"google.api.HttpBody":
		return true
	}
	return false
}

// wellKnownTypeToSchema returns the OpenAPI Schema for a well-known protobuf type.
// Returns nil if the type is not a well-known type.
func wellKnownTypeToSchema(name protoreflect.FullName) *Schema {
	switch name {
	case "google.protobuf.Timestamp":
		return &Schema{Type: "string", Format: "date-time"}
	case "google.protobuf.Duration":
		return &Schema{Type: "string"}
	case "google.protobuf.Empty":
		return &Schema{Type: "object"}
	case "google.protobuf.StringValue":
		return &Schema{Type: "string", Nullable: true}
	case "google.protobuf.Int32Value":
		return &Schema{Type: "integer", Format: "int32", Nullable: true}
	case "google.protobuf.Int64Value":
		return &Schema{Type: "integer", Format: "int64", Nullable: true}
	case "google.protobuf.UInt32Value":
		return &Schema{Type: "integer", Format: "int32", Nullable: true}
	case "google.protobuf.UInt64Value":
		return &Schema{Type: "integer", Format: "int64", Nullable: true}
	case "google.protobuf.BoolValue":
		return &Schema{Type: "boolean", Nullable: true}
	case "google.protobuf.FloatValue":
		return &Schema{Type: "number", Format: "float", Nullable: true}
	case "google.protobuf.DoubleValue":
		return &Schema{Type: "number", Format: "double", Nullable: true}
	case "google.protobuf.BytesValue":
		return &Schema{Type: "string", Format: "byte", Nullable: true}
	case "google.api.HttpBody":
		return &Schema{Type: "string", Format: "binary"}
	}
	return nil
}

// protoFieldToSchema converts a protobuf field descriptor to an OpenAPI Schema.
func protoFieldToSchema(field *protogen.Field, schemaResolver func(*protogen.Message) string) *Schema {
	if field.Desc.IsList() {
		elemSchema := protoKindToSchema(field, schemaResolver)
		return &Schema{
			Type:  "array",
			Items: elemSchema,
		}
	}

	if field.Desc.IsMap() {
		// Map fields have a synthetic message with key and value sub-fields.
		valueField := field.Message.Fields[1]
		valueSchema := protoFieldToSchema(valueField, schemaResolver)
		return &Schema{
			Type:                 "object",
			AdditionalProperties: valueSchema,
		}
	}

	return protoKindToSchema(field, schemaResolver)
}

// protoKindToSchema converts a protobuf field's kind to an OpenAPI Schema.
// This function handles scalar, enum, and message types (but not list/map).
func protoKindToSchema(field *protogen.Field, schemaResolver func(*protogen.Message) string) *Schema {
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		return &Schema{Type: "boolean", Nullable: field.Desc.HasPresence()}
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return &Schema{Type: "integer", Format: "int32", Nullable: field.Desc.HasPresence()}
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return &Schema{Type: "integer", Format: "int32", Nullable: field.Desc.HasPresence()}
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return &Schema{Type: "integer", Format: "int64", Nullable: field.Desc.HasPresence()}
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return &Schema{Type: "integer", Format: "int64", Nullable: field.Desc.HasPresence()}
	case protoreflect.FloatKind:
		return &Schema{Type: "number", Format: "float", Nullable: field.Desc.HasPresence()}
	case protoreflect.DoubleKind:
		return &Schema{Type: "number", Format: "double", Nullable: field.Desc.HasPresence()}
	case protoreflect.StringKind:
		return &Schema{Type: "string", Nullable: field.Desc.HasPresence()}
	case protoreflect.BytesKind:
		return &Schema{Type: "string", Format: "byte", Nullable: field.Desc.HasPresence()}
	case protoreflect.EnumKind:
		values := make([]string, 0, len(field.Enum.Values))
		for i := 0; i < len(field.Enum.Values); i++ {
			values = append(values, string(field.Enum.Values[i].Desc.Name()))
		}
		return &Schema{Type: "string", Enum: values, Nullable: field.Desc.HasPresence()}
	case protoreflect.MessageKind, protoreflect.GroupKind:
		msg := field.Message
		if msg == nil {
			return &Schema{Type: "object"}
		}
		name := msg.Desc.FullName()
		if isWellKnownType(name) {
			return wellKnownTypeToSchema(name)
		}
		ref := ""
		if schemaResolver != nil {
			ref = schemaResolver(msg)
		}
		if ref != "" {
			return &Schema{Ref: ref}
		}
		return &Schema{Type: "object"}
	default:
		return &Schema{Type: "object"}
	}
}
