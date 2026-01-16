package client

import (
	"strconv"

	"github.com/soyacen/goose/cmd/protoc-gen-goose/constant"
	"github.com/soyacen/goose/cmd/protoc-gen-goose/parser"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func (f *Generator) GenerateRequestEncoder(service *parser.Service, g *protogen.GeneratedFile) error {
	g.P("type ", service.Unexported(service.RequestEncoderName()), " struct {")
	g.P("target string")
	g.P("marshalOptions ", constant.ProtoJsonMarshalOptionsIdent)
	g.P("resolver ", constant.ResolverIdent)
	g.P("}")
	for _, endpoint := range service.Endpoints {
		g.P("func (encoder *", service.Unexported(service.RequestEncoderName()), ") ", endpoint.Name(), "(ctx ", constant.ContextIdent, ", req *", endpoint.InputGoIdent(), ") (*", constant.RequestIdent, ", error){")
		g.P("if req == nil {")
		g.P("return nil, ", constant.NewErrorIdent, "(", strconv.Quote("request is nil"), ")")
		g.P("}")
		g.P("target, err := ", constant.ResolveIdent, "(ctx, encoder.resolver, encoder.target)")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("method := ", strconv.Quote(endpoint.Method()))
		g.P("header := ", constant.Header, "{}")
		g.P("var body ", constant.Buffer)
		bodyMessage, bodyField, pathFields, queryFields, err := endpoint.ParseParameters()
		if err != nil {
			return err
		}
		_ = pathFields
		_ = queryFields
		if bodyMessage != nil {
			srcValue := []any{"req"}
			switch bodyMessage.Desc.FullName() {
			case "google.api.HttpBody":
				f.PrintEncodeHttpBodyToRequest(g, srcValue)
			case "google.rpc.HttpRequest":
				f.PrintEncodeHttpRequestToRequest(g, srcValue)
			default:
				f.PrintEncodeMessageToRequest(g, srcValue)
			}
		} else if bodyField != nil {
			switch bodyField.Desc.Kind() {
			case protoreflect.MessageKind:
				srcValue := []any{"req.Get", bodyField.GoName, "()"}
				switch bodyField.Message.Desc.FullName() {
				case "google.api.HttpBody":
					f.PrintEncodeHttpBodyToRequest(g, srcValue)
				default:
					f.PrintEncodeMessageToRequest(g, srcValue)
				}
			}
		}

		g.P("path := ", strconv.Quote(endpoint.Path()))
		f.PrintPathField(g, pathFields)
		g.P("target.Path = path")

		f.PrintQueryField(g, queryFields)

		g.P("request, err := ", constant.NewRequestWithContextIndent, "(ctx, method, target.String(), &body)")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P(constant.CopyHeaderIdent, "(request.Header, header)")
		g.P("return request, nil")
		g.P("}")
		g.P()
	}
	g.P()
	return nil
}

func (f *Generator) PrintEncodeHttpBodyToRequest(g *protogen.GeneratedFile, srcValue []any) {
	g.P(append(append([]any{"if err := ", constant.EncodeHttpBodyToRequestIdent, "(ctx, "}, srcValue...), ", header, &body); err!= nil {")...)
	g.P("return nil, err")
	g.P("}")
}

func (f *Generator) PrintEncodeHttpRequestToRequest(g *protogen.GeneratedFile, srcValue []any) {
	g.P(append(append([]any{"if err := ", constant.EncodeHttpRequestIdent, "(ctx, "}, srcValue...), ", header, &body); err!= nil {")...)
	g.P("return nil, err")
	g.P("}")
}

func (f *Generator) PrintEncodeMessageToRequest(g *protogen.GeneratedFile, srcValue []any) {
	g.P(append(append([]any{"if err := ", constant.EncodeMessageIdent, "(ctx, "}, srcValue...), ", header, &body, encoder.marshalOptions); err!= nil {")...)
	g.P("return nil, err")
	g.P("}")
}

func (f *Generator) PrintPathField(g *protogen.GeneratedFile, pathFields []*protogen.Field) {
	if len(pathFields) <= 0 {
		return
	}
	g.P("pairs := map[string]string{")
	for _, field := range pathFields {
		g.P(append(append([]any{strconv.Quote(string(field.Desc.Name())), ": "}, f.PathFieldFormat(field)...), ",")...)
	}
	g.P("}")
	g.P("path = ", constant.URLPathIdent, "(path, pairs)")
	g.P("path, err = ", constant.JoinPathIndent, "(target.Path, path)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
}

func (f *Generator) PathFieldFormat(field *protogen.Field) []any {
	srcValue := []any{"req.Get", field.GoName, "()"}
	switch field.Desc.Kind() {
	case protoreflect.BoolKind: // bool
		return f.BoolValueFormat(srcValue)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind: // int32
		return f.IntValueFormat(srcValue)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind: // uint32
		return f.UintValueFormat(srcValue)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind: // int64
		return f.IntValueFormat(srcValue)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind: // uint64
		return f.UintValueFormat(srcValue)
	case protoreflect.FloatKind: // float32
		return f.FloatValueFormat(srcValue, "32")
	case protoreflect.DoubleKind: // float64
		return f.FloatValueFormat(srcValue, "64")
	case protoreflect.StringKind: // string
		return f.StringKindFormat(srcValue)
	case protoreflect.EnumKind: //  enum int32
		return f.IntValueFormat(srcValue)
	case protoreflect.MessageKind:
		switch field.Message.Desc.FullName() {
		case "google.protobuf.BoolValue":
			return f.UnwrapBoolValueFormat(srcValue)
		case "google.protobuf.Int64Value":
			return f.UnwrapIntValueFormat(srcValue)
		case "google.protobuf.UInt64Value":
			return f.UnwrapUintValueFormat(srcValue)
		case "google.protobuf.Int32Value":
			return f.UnwrapIntValueFormat(srcValue)
		case "google.protobuf.UInt32Value":
			return f.UnwrapUintValueFormat(srcValue)
		case "google.protobuf.FloatValue":
			return f.UnwrapFloatValueFormat(srcValue, "32")
		case "google.protobuf.DoubleValue":
			return f.UnwrapFloatValueFormat(srcValue, "64")
		case "google.protobuf.StringValue":
			return f.UnwrapStringValueFormat(srcValue)
		}
	}
	return nil
}

func (f *Generator) PrintQueryField(g *protogen.GeneratedFile, queryFields []*protogen.Field) {
	if len(queryFields) <= 0 {
		return
	}
	g.P("queries := ", constant.URLValuesIndent, "{}")
	for _, field := range queryFields {
		srcValue := []any{"req.Get", field.GoName, "()"}
		fieldName := string(field.Desc.Name())
		switch field.Desc.Kind() {
		case protoreflect.BoolKind: // bool
			if field.Desc.IsList() {
				f.PrintQuery(g, fieldName, append(f.BoolSliceFormat(srcValue), []any{"..."}...))
			} else {
				f.PrintQuery(g, fieldName, f.BoolValueFormat(srcValue))
			}
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind: // int32
			if field.Desc.IsList() {
				f.PrintQuery(g, fieldName, append(f.IntSliceFormat(srcValue), []any{"..."}...))
			} else {
				f.PrintQuery(g, fieldName, f.IntValueFormat(srcValue))
			}
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind: // uint32
			if field.Desc.IsList() {
				f.PrintQuery(g, fieldName, append(f.UintSliceFormat(srcValue), []any{"..."}...))
			} else {
				f.PrintQuery(g, fieldName, f.UintValueFormat(srcValue))
			}
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind: // int64
			if field.Desc.IsList() {
				f.PrintQuery(g, fieldName, append(f.IntSliceFormat(srcValue), []any{"..."}...))
			} else {
				f.PrintQuery(g, fieldName, f.IntValueFormat(srcValue))
			}
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind: // uint64
			if field.Desc.IsList() {
				f.PrintQuery(g, fieldName, append(f.UintSliceFormat(srcValue), []any{"..."}...))
			} else {
				f.PrintQuery(g, fieldName, f.UintValueFormat(srcValue))
			}
		case protoreflect.FloatKind: // float32
			if field.Desc.IsList() {
				f.PrintQuery(g, fieldName, append(f.FloatSliceFormat(srcValue, "32"), []any{"..."}...))
			} else {
				f.PrintQuery(g, fieldName, f.FloatValueFormat(srcValue, "32"))
			}
		case protoreflect.DoubleKind: // float64
			if field.Desc.IsList() {
				f.PrintQuery(g, fieldName, append(f.FloatSliceFormat(srcValue, "64"), []any{"..."}...))
			} else {
				f.PrintQuery(g, fieldName, f.FloatValueFormat(srcValue, "64"))
			}
		case protoreflect.StringKind: // string
			if field.Desc.IsList() {
				f.PrintQuery(g, fieldName, append(f.StringKindFormat(srcValue), []any{"..."}...))
			} else {
				f.PrintQuery(g, fieldName, f.StringKindFormat(srcValue))
			}
		case protoreflect.EnumKind: // enum int32
			if field.Desc.IsList() {
				f.PrintQuery(g, fieldName, append(f.IntSliceFormat(srcValue), []any{"..."}...))
			} else {
				f.PrintQuery(g, fieldName, f.IntValueFormat(srcValue))
			}
		case protoreflect.MessageKind:
			switch field.Message.Desc.FullName() {
			case "google.protobuf.BoolValue":
				if field.Desc.IsList() {
					f.PrintQuery(g, fieldName, append(f.UnwrapBoolSliceFormat(srcValue), []any{"..."}...))
				} else {
					f.PrintQuery(g, fieldName, f.UnwrapBoolValueFormat(srcValue))
				}
			case "google.protobuf.Int32Value":
				if field.Desc.IsList() {
					f.PrintQuery(g, fieldName, append(f.UnwrapIntSliceFormat(srcValue, constant.UnwrapInt32SliceIdent, "32"), []any{"..."}...))
				} else {
					f.PrintQuery(g, fieldName, f.UnwrapIntValueFormat(srcValue))
				}
			case "google.protobuf.UInt32Value":
				if field.Desc.IsList() {
					f.PrintQuery(g, fieldName, append(f.UnwrapUintSliceFormat(srcValue, constant.UnwrapUint32SliceIdent, "32"), []any{"..."}...))
				} else {
					f.PrintQuery(g, fieldName, f.UnwrapUintValueFormat(srcValue))
				}
			case "google.protobuf.Int64Value":
				if field.Desc.IsList() {
					f.PrintQuery(g, fieldName, append(f.UnwrapIntSliceFormat(srcValue, constant.UnwrapInt64SliceIdent, "64"), []any{"..."}...))
				} else {
					f.PrintQuery(g, fieldName, f.UnwrapIntValueFormat(srcValue))
				}
			case "google.protobuf.UInt64Value":
				if field.Desc.IsList() {
					f.PrintQuery(g, fieldName, append(f.UnwrapUintSliceFormat(srcValue, constant.UnwrapUint64SliceIdent, "64"), []any{"..."}...))
				} else {
					f.PrintQuery(g, fieldName, f.UnwrapUintValueFormat(srcValue))
				}

			case "google.protobuf.FloatValue":
				if field.Desc.IsList() {
					f.PrintQuery(g, fieldName, append(f.UnwrapFloatSliceFormat(srcValue, constant.UnwrapFloat32SliceIdent, "32"), []any{"..."}...))
				} else {
					f.PrintQuery(g, fieldName, f.UnwrapFloatValueFormat(srcValue, "32"))
				}
			case "google.protobuf.DoubleValue":
				if field.Desc.IsList() {
					f.PrintQuery(g, fieldName, append(f.UnwrapFloatSliceFormat(srcValue, constant.UnwrapFloat64SliceIdent, "64"), []any{"..."}...))
				} else {
					f.PrintQuery(g, fieldName, f.UnwrapFloatValueFormat(srcValue, "64"))
				}
			case "google.protobuf.StringValue":
				if field.Desc.IsList() {
					f.PrintQuery(g, fieldName, append(f.UnwrapStringSliceFormat(srcValue), []any{"..."}...))
				} else {
					f.PrintQuery(g, fieldName, f.UnwrapStringValueFormat(srcValue))
				}
			}
		}
	}
	g.P("target.RawQuery = queries.Encode()")
}

func (f *Generator) PrintQuery(g *protogen.GeneratedFile, fieldName string, srcValue []any) {
	g.P(append(append([]any{"queries[", strconv.Quote(fieldName), "] = append(queries[", strconv.Quote(fieldName), "], "}, srcValue...), []any{")"}...)...)
}

func (f *Generator) BoolValueFormat(srcValue []any) []any {
	return append(append([]any{constant.FormatBoolIdent, "("}, srcValue...), []any{")"}...)
}

func (f *Generator) BoolSliceFormat(srcValue []any) []any {
	return append(append([]any{constant.FormatBoolSliceIdent, "("}, srcValue...), []any{")"}...)
}

func (f *Generator) UnwrapBoolValueFormat(srcValue []any) []any {
	return append(append([]any{constant.FormatBoolIdent, "("}, srcValue...), []any{".GetValue()", ")"}...)
}

func (f *Generator) UnwrapBoolSliceFormat(srcValue []any) []any {
	return append(append([]any{constant.FormatBoolSliceIdent, "(", constant.UnwrapBoolSliceIdent, "("}, srcValue...), []any{"))"}...)
}

func (f *Generator) IntValueFormat(srcValue []any) []any {
	return append(append([]any{constant.FormatIntIdent, "("}, srcValue...), []any{", 10)"}...)
}

func (f *Generator) IntSliceFormat(srcValue []any) []any {
	return append(append([]any{constant.FormatIntSliceIdent, "("}, srcValue...), []any{", 10)"}...)
}

func (f *Generator) UnwrapIntValueFormat(srcValue []any) []any {
	return append(append([]any{constant.FormatIntIdent, "("}, srcValue...), []any{".GetValue()", ", 10)"}...)
}

func (f *Generator) UnwrapIntSliceFormat(srcValue []any, unwrapper any, bitSize string) []any {
	return append(append([]any{constant.FormatIntSliceIdent, "(", unwrapper, "("}, srcValue...), []any{"), 10)"}...)
}

func (f *Generator) UintValueFormat(srcValue []any) []any {
	return append(append([]any{constant.FormatUintIdent, "("}, srcValue...), []any{", 10)"}...)
}

func (f *Generator) UintSliceFormat(srcValue []any) []any {
	return append(append([]any{constant.FormatUintSliceIdent, "("}, srcValue...), []any{", 10)"}...)
}

func (f *Generator) UnwrapUintValueFormat(srcValue []any) []any {
	return append(append([]any{constant.FormatUintIdent, "("}, srcValue...), []any{".GetValue()", ", 10)"}...)
}

func (f *Generator) UnwrapUintSliceFormat(srcValue []any, unwrapper any, bitSize string) []any {
	return append(append([]any{constant.FormatUintSliceIdent, "(", unwrapper, "("}, srcValue...), []any{"), 10)"}...)
}

func (f *Generator) FloatValueFormat(srcValue []any, bitSize string) []any {
	return append(append([]any{constant.FormatFloatIdent, "("}, srcValue...), []any{", 'f', -1, ", bitSize, ")"}...)
}

func (f *Generator) FloatSliceFormat(srcValue []any, bitSize string) []any {
	return append(append([]any{constant.FormatFloatSliceIdent, "("}, srcValue...), []any{", 'f', -1, ", bitSize, ")"}...)
}

func (f *Generator) UnwrapFloatValueFormat(srcValue []any, bitSize string) []any {
	return append(append([]any{constant.FormatFloatIdent, "("}, srcValue...), []any{".GetValue()", ", 'f', -1, ", bitSize, ")"}...)
}

func (f *Generator) UnwrapFloatSliceFormat(srcValue []any, unwrapper any, bitSize string) []any {
	return append(append([]any{constant.FormatFloatSliceIdent, "(", unwrapper, "("}, srcValue...), []any{"), 'f', -1, ", bitSize, ")"}...)
}

func (f *Generator) StringKindFormat(srcValue []any) []any {
	return append(append([]any{}, srcValue...), []any{}...)
}

func (f *Generator) UnwrapStringValueFormat(srcValue []any) []any {
	return append(append([]any{}, srcValue...), []any{".GetValue()"}...)
}

func (f *Generator) UnwrapStringSliceFormat(srcValue []any) []any {
	return append(append([]any{constant.UnwrapStringSliceIdent, "("}, srcValue...), []any{")"}...)
}
