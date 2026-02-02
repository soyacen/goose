package server

import (
	"strconv"

	"github.com/soyacen/goose/cmd/protoc-gen-goose/constant"
	"github.com/soyacen/goose/cmd/protoc-gen-goose/parser"
	"google.golang.org/protobuf/compiler/protogen"
)

type Generator struct{}



func (generator *Generator) GenerateAppendServerFunc(service *parser.Service, g *protogen.GeneratedFile) error {
	g.P("func ", service.AppendRouteName(), "(router *", constant.RouterIdent, ", service ", service.ServiceName(), ", opts ...", constant.ServerOptionIdent, ") ", "*", constant.RouterIdent, " {")
	g.P("options := ", constant.ServerNewOptionsIdent, "(opts...)")
	g.P("handler :=  ", service.Unexported(service.HandlerName()), "{")
	g.P("service: service,")
	g.P("decoder: ", service.Unexported(service.RequestDecoderName()), "{")
	g.P("unmarshalOptions: options.UnmarshalOptions(),")
	g.P("},")
	g.P("encoder: ", service.Unexported(service.ResponseEncoderName()), "{")
	g.P("marshalOptions: options.MarshalOptions(),")
	g.P("unmarshalOptions: options.UnmarshalOptions(),")
	g.P("},")
	g.P("errorEncoder: options.ErrorEncoder(),")
	g.P("shouldFailFast: options.ShouldFailFast(),")
	g.P("onValidationErrCallback: options.OnValidationErrCallback(),")
	g.P("middleware: ", constant.ServerChainIdent, "(options.Middlewares()...),")
	g.P("}")
	for _, endpoint := range service.Endpoints {
		g.P("router.Handle(", strconv.Quote(endpoint.Method()+" "+endpoint.Path()), ", ", constant.HttpHandlerFuncIdent, "(handler.", endpoint.Name(), "))")
	}
	g.P("return router")
	g.P("}")
	g.P()
	return nil
}

func (generator *Generator) GenerateHandlers(service *parser.Service, g *protogen.GeneratedFile) error {
	g.P("type ", service.Unexported(service.HandlerName()), " struct {")
	g.P("service ", service.ServiceName())
	g.P("decoder ", service.Unexported(service.RequestDecoderName()))
	g.P("encoder ", service.Unexported(service.ResponseEncoderName()))
	g.P("errorEncoder ", constant.ErrorEncoderIdent)
	g.P("shouldFailFast bool")
	g.P("onValidationErrCallback ", constant.OnErrCallbackIdent)
	g.P("middleware ", constant.ServerMiddlewareIdent)
	g.P("}")
	g.P()
	for _, endpoint := range service.Endpoints {
		g.P("func (h ", service.Unexported(service.HandlerName()), ")", endpoint.Name(), "(response ", constant.ResponseWriterIdent, ", request *", constant.RequestIdent, ") {")
		g.P("invoke := func(response ", constant.ResponseWriterIdent, ", request *", constant.RequestIdent, ") {")
		g.P("ctx := request.Context()")
		g.P("req, err := h.decoder.", endpoint.Name(), "(ctx, request)")
		g.P("if err != nil {")
		g.P("h.errorEncoder(ctx, err, response)")
		g.P("return")
		g.P("}")
		g.P("if err := ", constant.ValidateRequestIdent, "(ctx, req, h.shouldFailFast, h.onValidationErrCallback)", "; err != nil {")
		g.P("h.errorEncoder(ctx, err, response)")
		g.P("return")
		g.P("}")
		g.P("resp, err := h.service.", endpoint.Name(), "(ctx, req)")
		g.P("if err != nil {")
		g.P("h.errorEncoder(ctx, err, response)")
		g.P("return")
		g.P("}")
		g.P("if err := h.encoder.", endpoint.Name(), "(ctx, response, resp); err != nil {")
		g.P("h.errorEncoder(ctx, err, response)")
		g.P("return")
		g.P("}")
		g.P("}")
		g.P(constant.ServerInvokeIdent, "(h.middleware, response, request, invoke, ", endpoint.DescName(), ".RouteInfo)")
		g.P("}")
		g.P()
	}
	return nil
}
