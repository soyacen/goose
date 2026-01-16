package server

import (
	"fmt"

	"github.com/soyacen/goose/cmd/protoc-gen-goose/constant"
	"github.com/soyacen/goose/cmd/protoc-gen-goose/parser"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func (generator *Generator) GenerateEncodeResponse(service *parser.Service, g *protogen.GeneratedFile) error {
	g.P("type ", service.Unexported(service.ResponseEncoderName()), " struct {")
	g.P("marshalOptions ", constant.ProtoJsonMarshalOptionsIdent)
	g.P("unmarshalOptions ", constant.ProtoJsonUnmarshalOptionsIdent)
	g.P("}")
	for _, endpoint := range service.Endpoints {
		g.P("func (encoder ", service.Unexported(service.ResponseEncoderName()), ")", endpoint.Name(), "(ctx ", constant.ContextIdent, ", w ", constant.ResponseWriterIdent, ", resp *", endpoint.OutputGoIdent(), ") error {")
		bodyParameter := endpoint.ResponseBody()
		switch bodyParameter {
		case "", "*":
			message := endpoint.Output()
			switch message.Desc.FullName() {
			case "google.api.HttpBody":
				srcValue := []any{"resp"}
				generator.PrintHttpBodyEncodeBlock(g, srcValue)
			case "google.rpc.HttpResponse":
				srcValue := []any{"resp"}
				generator.PrintHttpResponseEncodeBlock(g, srcValue)
			default:
				srcValue := []any{"resp"}
				generator.PrintResponseEncodeBlock(g, srcValue)
			}
		default:
			bodyField := parser.FindField(bodyParameter, endpoint.Output())
			if bodyField == nil {
				return fmt.Errorf("%s, failed to find body response field %s", endpoint.FullName(), bodyParameter)
			}
			switch bodyField.Desc.Kind() {
			case protoreflect.MessageKind:
				switch bodyField.Message.Desc.FullName() {
				case "google.api.HttpBody":
					srcValue := []any{"resp.Get", bodyField.GoName, "()"}
					generator.PrintHttpBodyEncodeBlock(g, srcValue)
				default:
					srcValue := []any{"resp.Get", bodyField.GoName, "()"}
					generator.PrintResponseEncodeBlock(g, srcValue)
				}
			}
		}
		g.P("}")
	}
	g.P()
	return nil
}

func (generator *Generator) PrintHttpBodyEncodeBlock(g *protogen.GeneratedFile, srcValue []any) {
	g.P(append(append([]any{"return ", constant.EncodeHttpBodyIdent, "(ctx, w, "}, srcValue...), ")")...)
}

func (generator *Generator) PrintHttpResponseEncodeBlock(g *protogen.GeneratedFile, srcValue []any) {
	g.P(append(append([]any{"return ", constant.EncodeHttpResponseIdent, "(ctx, w, "}, srcValue...), ")")...)
}

func (generator *Generator) PrintResponseEncodeBlock(g *protogen.GeneratedFile, srcValue []any) {
	g.P(append(append([]any{"return ", constant.EncodeResponseIdent, "(ctx, w, "}, srcValue...), ", encoder.marshalOptions)")...)
}
