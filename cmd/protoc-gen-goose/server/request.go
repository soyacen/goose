package server

import (
	"strconv"
	"strings"

	"github.com/soyacen/goose/cmd/protoc-gen-goose/constant"
	"github.com/soyacen/goose/cmd/protoc-gen-goose/parser"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func (generator *Generator) GenerateDecodeRequest(service *parser.Service, g *protogen.GeneratedFile) error {
	g.P("type ", service.Unexported(service.RequestDecoderName()), " struct {")
	g.P("unmarshalOptions ", constant.ProtoJsonUnmarshalOptionsIdent)
	g.P("}")
	for _, endpoint := range service.Endpoints {
		g.P("func (decoder ", service.Unexported(service.RequestDecoderName()), ")", endpoint.Name(), "(ctx ", constant.ContextIdent, ", request *", constant.RequestIdent, ") (*", endpoint.InputGoIdent(), ", error){")
		g.P("req := &", endpoint.InputGoIdent(), "{}")
		g.P("ok, err := ", constant.CustomDecodeRequestIdent, "(ctx, request, req)")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("if ok {")
		g.P("return req, nil")
		g.P("}")

		bodyMessage, bodyField, pathFields, queryFields, err := endpoint.ParseParameters()
		if err != nil {
			return err
		}

		if bodyMessage != nil {
			switch bodyMessage.Desc.FullName() {
			case "google.api.HttpBody":
				generator.PrintHttpBodyDecodeBlock(g, []any{"req"})
			case "google.rpc.HttpRequest":
				generator.PrintHttpRequestEncodeBlock(g, []any{"req"})
			default:
				generator.PrintRequestDecodeBlock(g, []any{"req"})
			}
		} else if bodyField != nil {
			tgtValue := []any{"req.", bodyField.GoName}
			g.P(append(append([]any{"if "}, tgtValue...), " == nil {")...)
			g.P(append(tgtValue, " = &", bodyField.Message.GoIdent, "{}")...)
			g.P("}")
			switch bodyField.Desc.Kind() {
			case protoreflect.MessageKind:
				switch bodyField.Message.Desc.FullName() {
				case "google.api.HttpBody":
					generator.PrintHttpBodyDecodeBlock(g, tgtValue)
				default:
					generator.PrintRequestDecodeBlock(g, []any{"req.", bodyField.GoName})
				}
			}
		}

		if len(pathFields) > 0 {
			fields := make([]string, 0, len(pathFields))
			for _, field := range pathFields {
				fields = append(fields, strconv.Quote(string(field.Desc.Name())))
			}
			g.P("vars := ", constant.FormFromPathIdent, "(request, ", strings.Join(fields, ", "), ")")
			generator.PrintPathField(g, pathFields)
		}

		if len(queryFields) > 0 {
			g.P("queries := request.URL.Query()")
			g.P("var queryErr error")
			generator.PrintQueryField(g, queryFields)
			g.P("if queryErr != nil {")
			g.P("return nil, queryErr")
			g.P("}")
		}

		g.P("return req, nil")
		g.P("}")
	}
	g.P()
	return nil
}

func (generator *Generator) PrintHttpBodyDecodeBlock(g *protogen.GeneratedFile, tgtValue []any) {
	g.P(append(append([]any{"if err := ", constant.DecodeHttpBodyIdent, "(ctx, request, "}, tgtValue...), "); err != nil {")...)
	g.P("return nil, err")
	g.P("}")
}

func (generator *Generator) PrintHttpRequestEncodeBlock(g *protogen.GeneratedFile, tgtValue []any) {
	g.P(append(append([]any{"if err := ", constant.DecodeHttpRequestIdent, "(ctx, request, "}, tgtValue...), "); err != nil {")...)
	g.P("return nil, err")
	g.P("}")
}

func (generator *Generator) PrintRequestDecodeBlock(g *protogen.GeneratedFile, tgtValue []any) {
	g.P(append(append([]any{"if err := ", constant.DecodeRequestIdent, "(ctx, request, "}, tgtValue...), ", decoder.unmarshalOptions); err != nil {")...)
	g.P("return nil, err")
	g.P("}")
}

func (generator *Generator) PrintPathField(g *protogen.GeneratedFile, pathFields []*protogen.Field) {
	if len(pathFields) <= 0 {
		return
	}
	form := "vars"
	errName := "varErr"
	g.P("var ", errName, " error")
	for _, field := range pathFields {
		fieldName := string(field.Desc.Name())

		tgtValue := []any{"req.", field.GoName, " = "}
		tgtErrValue := []any{"req.", field.GoName, ", ", errName, " = "}
		srcValue := []any{"vars.Get(", strconv.Quote(fieldName), ")"}

		goType, pointer := parser.FieldGoType(g, field)
		if pointer {
			goType = append([]any{"*"}, goType...)
		}

		switch field.Desc.Kind() {
		case protoreflect.BoolKind: // bool
			if pointer {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetBoolPtrIdent, fieldName, form, errName)
			} else {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetBoolIdent, fieldName, form, errName)
			}
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind: // int32
			if pointer {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetIntPtrIdent, fieldName, form, errName)
			} else {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetIntIdent, fieldName, form, errName)
			}
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind: // uint32
			if pointer {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetUintPtrIdent, fieldName, form, errName)
			} else {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetUintIdent, fieldName, form, errName)
			}
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind: // int64
			if pointer {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetIntPtrIdent, fieldName, form, errName)
			} else {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetIntIdent, fieldName, form, errName)
			}
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind: // uint64
			if pointer {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetUintPtrIdent, fieldName, form, errName)
			} else {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetUintIdent, fieldName, form, errName)
			}
		case protoreflect.FloatKind: // float32
			if pointer {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetFloatPtrIdent, fieldName, form, errName)
			} else {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetFloatIdent, fieldName, form, errName)
			}
		case protoreflect.DoubleKind: // float64
			if pointer {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetFloatPtrIdent, fieldName, form, errName)
			} else {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetFloatIdent, fieldName, form, errName)
			}
		case protoreflect.StringKind: // string
			generator.PrintStringValueAssign(g, tgtValue, srcValue, pointer)
		case protoreflect.EnumKind: // enum int32
			if pointer {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetEnumPtrIdent(g, goType[1].(protogen.GoIdent)), fieldName, form, errName)
			} else {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetEnumIdent(g, goType[0].(protogen.GoIdent)), fieldName, form, errName)
			}
		case protoreflect.MessageKind:
			switch field.Message.Desc.FullName() {
			case "google.protobuf.BoolValue":
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetBoolValueIdent, fieldName, form, errName)
			case "google.protobuf.Int32Value":
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetInt32ValueIdent, fieldName, form, errName)
			case "google.protobuf.UInt32Value":
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetUint32ValueIdent, fieldName, form, errName)
			case "google.protobuf.Int64Value":
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetInt64ValueIdent, fieldName, form, errName)
			case "google.protobuf.UInt64Value":
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetUint64ValueIdent, fieldName, form, errName)
			case "google.protobuf.FloatValue":
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetFloat32ValueIdent, fieldName, form, errName)
			case "google.protobuf.DoubleValue":
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetFloat64ValueIdent, fieldName, form, errName)
			case "google.protobuf.StringValue":
				generator.PrintWrapStringValueAssign(g, tgtValue, srcValue)
			}
		}
	}
	g.P("if ", errName, " != nil {")
	g.P("return nil, ", errName)
	g.P("}")
}

func (generator *Generator) PrintQueryField(g *protogen.GeneratedFile, queryFields []*protogen.Field) {
	for _, field := range queryFields {
		fieldName := string(field.Desc.Name())

		tgtValue := []any{"req.", field.GoName, " = "}
		tgtErrValue := []any{"req.", field.GoName, ", queryErr = "}
		srcValue := []any{"queries.Get(", strconv.Quote(fieldName), ")"}
		if field.Desc.IsList() {
			srcValue = []any{"queries[", strconv.Quote(fieldName), "]"}
		}

		goType, pointer := parser.FieldGoType(g, field)
		if pointer {
			goType = append([]any{"*"}, goType...)
		}

		form := "queries"
		errName := "queryErr"

		switch field.Desc.Kind() {
		case protoreflect.BoolKind: // bool
			if field.Desc.IsList() {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetBoolSliceIdent, fieldName, form, errName)
			} else {
				if pointer {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetBoolPtrIdent, fieldName, form, errName)
				} else {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetBoolIdent, fieldName, form, errName)
				}
			}
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind: // int32
			if field.Desc.IsList() {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetIntSliceIdent, fieldName, form, errName)
			} else {
				if pointer {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetIntPtrIdent, fieldName, form, errName)
				} else {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetIntIdent, fieldName, form, errName)
				}
			}
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind: // uint32
			if field.Desc.IsList() {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetUintSliceIdent, fieldName, form, errName)
			} else {
				if pointer {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetUintPtrIdent, fieldName, form, errName)
				} else {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetUintIdent, fieldName, form, errName)
				}
			}
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind: // int64
			if field.Desc.IsList() {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetIntSliceIdent, fieldName, form, errName)
			} else {
				if pointer {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetIntPtrIdent, fieldName, form, errName)
				} else {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetIntIdent, fieldName, form, errName)
				}
			}
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind: // uint64
			if field.Desc.IsList() {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetUintSliceIdent, fieldName, form, errName)
			} else {
				if pointer {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetUintPtrIdent, fieldName, form, errName)
				} else {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetUintIdent, fieldName, form, errName)
				}
			}
		case protoreflect.FloatKind: // float32
			if field.Desc.IsList() {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetFloatSliceIdent, fieldName, form, errName)
			} else {
				if pointer {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetFloatPtrIdent, fieldName, form, errName)
				} else {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetFloatIdent, fieldName, form, errName)
				}
			}
		case protoreflect.DoubleKind: // float64
			if field.Desc.IsList() {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetFloatSliceIdent, fieldName, form, errName)
			} else {
				if pointer {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetFloatPtrIdent, fieldName, form, errName)
				} else {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetFloatIdent, fieldName, form, errName)
				}
			}
		case protoreflect.StringKind: // string
			if field.Desc.IsList() {
				generator.PrintStringListAssign(g, tgtValue, srcValue)
			} else {
				generator.PrintStringValueAssign(g, tgtValue, srcValue, pointer)
			}
		case protoreflect.EnumKind: // enum int32
			if field.Desc.IsList() {
				generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetEnumSliceIdent(g, goType[1].(protogen.GoIdent)), fieldName, form, errName)
			} else {
				if pointer {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetEnumPtrIdent(g, goType[1].(protogen.GoIdent)), fieldName, form, errName)
				} else {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetEnumIdent(g, goType[0].(protogen.GoIdent)), fieldName, form, errName)
				}
			}
		case protoreflect.MessageKind:
			switch field.Message.Desc.FullName() {
			case "google.protobuf.BoolValue":
				if field.Desc.IsList() {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetBoolValueSliceIdent, fieldName, form, errName)
				} else {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetBoolValueIdent, fieldName, form, errName)
				}
			case "google.protobuf.Int32Value":
				if field.Desc.IsList() {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetInt32ValueSliceIdent, fieldName, form, errName)
				} else {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetInt32ValueIdent, fieldName, form, errName)
				}
			case "google.protobuf.UInt32Value":
				if field.Desc.IsList() {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetUint32ValueSliceIdent, fieldName, form, errName)
				} else {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetUint32ValueIdent, fieldName, form, errName)
				}
			case "google.protobuf.Int64Value":
				if field.Desc.IsList() {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetInt64ValueSliceIdent, fieldName, form, errName)
				} else {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetInt64ValueIdent, fieldName, form, errName)
				}
			case "google.protobuf.UInt64Value":
				if field.Desc.IsList() {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetUint64ValueSliceIdent, fieldName, form, errName)
				} else {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetUint64ValueIdent, fieldName, form, errName)
				}
			case "google.protobuf.FloatValue":
				if field.Desc.IsList() {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetFloat32ValueSliceIdent, fieldName, form, errName)
				} else {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetFloat32ValueIdent, fieldName, form, errName)
				}

			case "google.protobuf.DoubleValue":
				if field.Desc.IsList() {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetFloat64ValueSliceIdent, fieldName, form, errName)
				} else {
					generator.PrintFieldAssign(g, tgtErrValue, goType, constant.GetFloat64ValueIdent, fieldName, form, errName)
				}
			case "google.protobuf.StringValue":
				if field.Desc.IsList() {
					generator.PrintWrapStringListAssign(g, tgtValue, srcValue)
				} else {
					generator.PrintWrapStringValueAssign(g, tgtValue, srcValue)
				}
			}
		}
	}
}

func (generator *Generator) PrintFieldAssign(g *protogen.GeneratedFile, tgtValue []any, goType []any, getter protogen.GoIdent, key string, form string, errName string) {
	g.P(append(append([]any{}, tgtValue...), append(append([]any{constant.GetFormIdent, "["}, goType...), append([]any{"](", errName, ", ", form, ", ", strconv.Quote(key), ", ", getter}, ")")...)...)...)
}

func (generator *Generator) PrintStringValueAssign(g *protogen.GeneratedFile, tgtValue []any, srcValue []any, hasPresence bool) {
	if hasPresence {
		g.P(append(tgtValue, append(append([]any{constant.ProtoStringIdent, "("}, srcValue...), ")")...)...)
	} else {
		g.P(append(tgtValue, srcValue...)...)
	}
}

func (generator *Generator) PrintWrapStringValueAssign(g *protogen.GeneratedFile, tgtValue []any, srcValue []any) {
	g.P(append(tgtValue, append(append([]any{constant.WrapperspbStringIdent, "("}, srcValue...), ")")...)...)
}

func (generator *Generator) PrintStringListAssign(g *protogen.GeneratedFile, tgtValue []any, srcValue []any) {
	g.P(append(tgtValue, srcValue...)...)
}

func (generator *Generator) PrintWrapStringListAssign(g *protogen.GeneratedFile, tgtValue []any, srcValue []any) {
	g.P(append(tgtValue, append(append([]any{constant.WrapStringSliceIdent, "("}, srcValue...), ")")...)...)
}
