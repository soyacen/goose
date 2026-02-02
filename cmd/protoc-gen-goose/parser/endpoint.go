package parser

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/soyacen/goose/internal/strconvx"
	"golang.org/x/exp/slices"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Endpoint struct {
	protoMethod *protogen.Method
	httpRule    *annotations.HttpRule
	pattern     *Pattern
}

func (e *Endpoint) Name() string {
	return e.protoMethod.GoName
}

func (e *Endpoint) Unexported(s string) string {
	return strings.ToLower(s[:1]) + s[1:]
}

func (e *Endpoint) FullName() string {
	return fmt.Sprintf("/%s/%s", e.protoMethod.Parent.Desc.FullName(), e.protoMethod.Desc.Name())
}

func (e *Endpoint) DescName() string {
	return fmt.Sprintf("_%s_%s_Desc", strconvx.GoSanitized(string(e.protoMethod.Parent.Desc.FullName())), e.protoMethod.Desc.Name())
}

func (e *Endpoint) IsStreaming() bool {
	return e.protoMethod.Desc.IsStreamingServer() || e.protoMethod.Desc.IsStreamingClient()
}

func (e *Endpoint) Input() *protogen.Message {
	return e.protoMethod.Input
}

func (e *Endpoint) Output() *protogen.Message {
	return e.protoMethod.Output
}

func (e *Endpoint) InputGoIdent() protogen.GoIdent {
	return e.Input().GoIdent
}

func (e *Endpoint) OutputGoIdent() protogen.GoIdent {
	return e.Output().GoIdent
}

func (e *Endpoint) ParseParameters() (*protogen.Message, *protogen.Field, []*protogen.Field, []*protogen.Field, error) {
	// body arguments
	var bodyMessage *protogen.Message
	var bodyField *protogen.Field
	bodyParameter := e.Body()
	switch bodyParameter {
	case "":
		// ignore
	case "*":
		bodyMessage = e.Input()
	default:
		bodyField = FindField(bodyParameter, e.Input())
		if bodyField == nil {
			return nil, nil, nil, nil, fmt.Errorf("%s, failed to find body field %s", e.FullName(), bodyParameter)
		}
	}

	var pathFields []*protogen.Field
	pathParameters, _ := e.PathParameters()
	for _, pathParameter := range pathParameters {
		field := FindField(pathParameter, e.Input())
		if field == nil {
			return nil, nil, nil, nil, fmt.Errorf("%s, failed to find path field %s", e.FullName(), pathParameter)
		}
		if field.Desc.IsList() || field.Desc.IsMap() {
			return nil, nil, nil, nil, fmt.Errorf("%s, path parameters do not support list or map", e.FullName())
		}

		switch field.Desc.Kind() {
		case protoreflect.BoolKind: // bool
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind: // int32
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind: // uint32
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind: // int64
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind: // uint64
		case protoreflect.FloatKind: // float32
		case protoreflect.DoubleKind: // float64
		case protoreflect.StringKind: // string
		case protoreflect.EnumKind: // enum
		case protoreflect.MessageKind:
			message := field.Message
			switch message.Desc.FullName() {
			case "google.protobuf.DoubleValue":
			case "google.protobuf.FloatValue":
			case "google.protobuf.Int64Value":
			case "google.protobuf.UInt64Value":
			case "google.protobuf.Int32Value":
			case "google.protobuf.UInt32Value":
			case "google.protobuf.BoolValue":
			case "google.protobuf.StringValue":
			default:
				return nil, nil, nil, nil, fmt.Errorf("%s, path parameters do not support %s", e.FullName(), message.Desc.FullName())
			}
		default:
			return nil, nil, nil, nil, fmt.Errorf("%s, path parameters do not support %s", e.FullName(), field.Desc.Kind())
		}

		pathFields = append(pathFields, field)
	}

	var queryFields []*protogen.Field
	if bodyMessage != nil {
		return bodyMessage, bodyField, pathFields, queryFields, nil
	}
	for _, field := range e.Input().Fields {
		if field == bodyField {
			continue
		}
		if slices.Contains(pathFields, field) {
			continue
		}
		if field.Desc.IsMap() {
			continue
		}
		switch field.Desc.Kind() {
		case protoreflect.BoolKind: // bool
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind: // int32
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind: // uint32
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind: // int64
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind: // uint64
		case protoreflect.FloatKind: // float32
		case protoreflect.DoubleKind: // float64
		case protoreflect.StringKind: // string
		case protoreflect.EnumKind: // enum
		case protoreflect.MessageKind:
			message := field.Message
			switch message.Desc.FullName() {
			case "google.protobuf.DoubleValue":
			case "google.protobuf.FloatValue":
			case "google.protobuf.Int64Value":
			case "google.protobuf.UInt64Value":
			case "google.protobuf.Int32Value":
			case "google.protobuf.UInt32Value":
			case "google.protobuf.BoolValue":
			case "google.protobuf.StringValue":
			default:
				continue
			}
		default:
			continue
		}
		queryFields = append(queryFields, field)
	}
	return bodyMessage, bodyField, pathFields, queryFields, nil
}

func (e *Endpoint) PathParameters() ([]string, error) {
	values := make([]string, 0, len(e.pattern.segments))
	for _, segment := range e.pattern.segments {
		if segment.wild {
			values = append(values, segment.s)
		}
	}
	return values, nil
}

func (e *Endpoint) SetPattern(pattern *Pattern) {
	e.pattern = pattern
}

func (e *Endpoint) Pattern() *Pattern {
	return e.pattern
}

func (e *Endpoint) SetHttpRule() {
	httpRule := proto.GetExtension(e.protoMethod.Desc.Options(), annotations.E_Http)
	if httpRule == nil || httpRule == annotations.E_Http.InterfaceOf(annotations.E_Http.Zero()) {
		// 默认为方法全称
		e.httpRule = &annotations.HttpRule{
			Pattern: &annotations.HttpRule_Post{Post: e.FullName()},
			Body:    "*",
		}
		return
	}
	e.httpRule = httpRule.(*annotations.HttpRule)
}

func (e *Endpoint) HttpRule() *annotations.HttpRule {
	return e.httpRule
}

func (e *Endpoint) Method() string {
	switch pattern := e.httpRule.GetPattern().(type) {
	case *annotations.HttpRule_Get:
		return http.MethodGet
	case *annotations.HttpRule_Post:
		return http.MethodPost
	case *annotations.HttpRule_Put:
		return http.MethodPut
	case *annotations.HttpRule_Delete:
		return http.MethodDelete
	case *annotations.HttpRule_Patch:
		return http.MethodPatch
	case *annotations.HttpRule_Custom:
		return pattern.Custom.GetKind()
	default:
		return ""
	}
}

func (e *Endpoint) Path() string {
	switch pattern := e.httpRule.GetPattern().(type) {
	case *annotations.HttpRule_Get:
		return pattern.Get
	case *annotations.HttpRule_Post:
		return pattern.Post
	case *annotations.HttpRule_Put:
		return pattern.Put
	case *annotations.HttpRule_Delete:
		return pattern.Delete
	case *annotations.HttpRule_Patch:
		return pattern.Patch
	case *annotations.HttpRule_Custom:
		return pattern.Custom.GetPath()
	default:
		return ""
	}
}

func (e *Endpoint) Body() string {
	return e.httpRule.GetBody()
}

func (e *Endpoint) ResponseBody() string {
	return e.httpRule.GetResponseBody()
}
