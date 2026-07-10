package stream

import (
	"github.com/soyacen/goose/cmd/protoc-gen-goose/constant"
	"github.com/soyacen/goose/cmd/protoc-gen-goose/parser"
	"google.golang.org/protobuf/compiler/protogen"
)

type Generator struct{}

func (gen *Generator) GenerateStreamServerInterface(service *parser.Service, g *protogen.GeneratedFile) error {
	g.P("type ", service.StreamServerName(), " interface {")
	for _, endpoint := range service.Endpoints {
		if endpoint.IsClientStreaming() {
			g.P(endpoint.Name(), "(", constant.WsClientStreamingServerIdent, "[*", endpoint.InputGoIdent(), ", *", endpoint.OutputGoIdent(), "]) error")
		} else if endpoint.IsServerStreaming() {
			g.P(endpoint.Name(), "(*", endpoint.InputGoIdent(), ", ", constant.WsServerStreamingServerIdent, "[*", endpoint.OutputGoIdent(), "]) error")
		} else if endpoint.IsBidiStreaming() {
			g.P(endpoint.Name(), "(", constant.WsBidiStreamingServerIdent, "[*", endpoint.InputGoIdent(), ", *", endpoint.OutputGoIdent(), "]) error")
		}
	}
	g.P("}")
	g.P()
	return nil
}

func (gen *Generator) GenerateStreamClientInterface(service *parser.Service, g *protogen.GeneratedFile) error {
	g.P("type ", service.StreamClientName(), " interface {")
	for _, endpoint := range service.Endpoints {
		if endpoint.IsClientStreaming() {
			g.P(endpoint.Name(), "(ctx ", constant.ContextIdent, ") (", constant.WsClientStreamingClientIdent, "[*", endpoint.InputGoIdent(), ", *", endpoint.OutputGoIdent(), "], error)")
		} else if endpoint.IsServerStreaming() {
			g.P(endpoint.Name(), "(ctx ", constant.ContextIdent, ", in *", endpoint.InputGoIdent(), ") (", constant.WsServerStreamingClientIdent, "[*", endpoint.OutputGoIdent(), "], error)")
		} else if endpoint.IsBidiStreaming() {
			g.P(endpoint.Name(), "(ctx ", constant.ContextIdent, ") (", constant.WsBidiStreamingClientIdent, "[*", endpoint.InputGoIdent(), ", *", endpoint.OutputGoIdent(), "], error)")
		}
	}
	g.P("}")
	g.P()
	return nil
}

func (gen *Generator) GenerateAppendStreamRouteFunc(service *parser.Service, g *protogen.GeneratedFile) error {
	g.P("func ", service.AppendStreamRouteName(), "(")
	g.P("router *", constant.RouterIdent, ",")
	g.P("service ", service.StreamServerName(), ",")
	g.P("middleware ", constant.ServerMiddlewareIdent, ",")
	g.P("marshalOpts ", constant.ProtoJsonMarshalOptionsIdent, ",")
	g.P("unmarshalOpts ", constant.ProtoJsonUnmarshalOptionsIdent, ",")
	g.P("acptOpts *", constant.WebsocketAcceptOptionsIdent, ",")
	g.P("cfg *", constant.WsConnConfigIdent, ",")
	g.P("logger *", constant.SlogLoggerIdent, ",")
	g.P(") *", constant.RouterIdent, " {")
	g.P("if router == nil {")
	g.P("router = ", constant.NewServeMuxIndent, "()")
	g.P("}")
	g.P("handler := &", service.StreamHandlerName(), "{")
	g.P("service: service,")
	g.P("middleware: middleware,")
	g.P("marshalOptions: marshalOpts,")
	g.P("unmarshalOptions: unmarshalOpts,")
	g.P("acptOpts: acptOpts,")
	g.P("cfg: cfg,")
	g.P("logger: logger,")
	g.P("}")
	for _, endpoint := range service.Endpoints {
		g.P("router.Handle(", endpoint.DescName(), ".RouteInfo.Pattern, ", constant.HttpHandlerFuncIdent, "(handler.", endpoint.Name(), "))")
	}
	g.P("return router")
	g.P("}")
	g.P()
	return nil
}

func (gen *Generator) GenerateStreamHandlerStruct(service *parser.Service, g *protogen.GeneratedFile) error {
	g.P("type ", service.StreamHandlerName(), " struct {")
	g.P("service ", service.StreamServerName())
	g.P("middleware ", constant.ServerMiddlewareIdent)
	g.P("marshalOptions ", constant.ProtoJsonMarshalOptionsIdent)
	g.P("unmarshalOptions ", constant.ProtoJsonUnmarshalOptionsIdent)
	g.P("acptOpts *", constant.WebsocketAcceptOptionsIdent)
	g.P("cfg *", constant.WsConnConfigIdent)
	g.P("logger *", constant.SlogLoggerIdent)
	g.P("}")
	g.P()
	return nil
}

func (gen *Generator) GenerateStreamHandlerMethods(service *parser.Service, g *protogen.GeneratedFile) error {
	serviceName := service.Name()
	for _, endpoint := range service.Endpoints {
		g.P("func (h ", service.StreamHandlerName(), ") ", endpoint.Name(), "(response ", constant.ResponseWriterIdent, ", request *", constant.RequestIdent, ") {")
		g.P("invoke := func(response ", constant.ResponseWriterIdent, ", request *", constant.RequestIdent, ") {")
		g.P("ctx, conn, cancel, err := ", constant.WsAcceptConnIdent, "(response, request, h.acptOpts, h.cfg, h.logger)")
		g.P("if err != nil {")
		g.P("h.logger.Error(\"failed to accept websocket connection\",")
		g.P("\"service\", ", strconvQuote(serviceName), ", \"method\", ", strconvQuote(endpoint.Name()), ", \"error\", err)")
		g.P("return")
		g.P("}")
		g.P("defer cancel()")

		if endpoint.IsServerStreaming() {
			g.P("var req ", endpoint.InputGoIdent())
			g.P("data, err := conn.Read(ctx)")
			g.P("if err != nil {")
			g.P("if !", constant.WsIsNormalCloseIdent, "(err) {")
			g.P("h.logger.Error(\"failed to read request\",")
			g.P("\"service\", ", strconvQuote(serviceName), ", \"method\", ", strconvQuote(endpoint.Name()), ", \"error\", err)")
			g.P("}")
			g.P("return")
			g.P("}")
			g.P("if err := h.unmarshalOptions.Unmarshal(data, &req); err != nil {")
			g.P("h.logger.Error(\"failed to unmarshal request\",")
			g.P("\"service\", ", strconvQuote(serviceName), ", \"method\", ", strconvQuote(endpoint.Name()), ", \"error\", err)")
			g.P("return")
			g.P("}")
		}

		g.P("stream := ", constant.WsNewServerStreamIdent, "[*", endpoint.InputGoIdent(), ", *", endpoint.OutputGoIdent(), "](ctx, conn, h.marshalOptions, h.unmarshalOptions)")

		if endpoint.IsClientStreaming() {
			g.P("if err := h.service.", endpoint.Name(), "(stream); err != nil && !", constant.WsIsNormalCloseIdent, "(err) {")
			g.P("h.logger.Error(\"failed to handle client stream\",")
			g.P("\"service\", ", strconvQuote(serviceName), ", \"method\", ", strconvQuote(endpoint.Name()), ", \"error\", err)")
			g.P("}")
		} else if endpoint.IsServerStreaming() {
			g.P("if err := h.service.", endpoint.Name(), "(&req, stream); err != nil && !", constant.WsIsNormalCloseIdent, "(err) {")
			g.P("h.logger.Error(\"failed to handle server stream\",")
			g.P("\"service\", ", strconvQuote(serviceName), ", \"method\", ", strconvQuote(endpoint.Name()), ", \"error\", err)")
			g.P("}")
		} else if endpoint.IsBidiStreaming() {
			g.P("if err := h.service.", endpoint.Name(), "(stream); err != nil && !", constant.WsIsNormalCloseIdent, "(err) {")
			g.P("h.logger.Error(\"failed to handle bidi stream\",")
			g.P("\"service\", ", strconvQuote(serviceName), ", \"method\", ", strconvQuote(endpoint.Name()), ", \"error\", err)")
			g.P("}")
		}

		g.P("if err := stream.CloseSend(); err != nil && !", constant.WsIsNormalCloseIdent, "(err) {")
		g.P("h.logger.Error(\"failed to close send stream\",")
		g.P("\"service\", ", strconvQuote(serviceName), ", \"method\", ", strconvQuote(endpoint.Name()), ", \"error\", err)")
		g.P("}")

		g.P("}")
		g.P(constant.ServerInvokeIdent, "(h.middleware, response, request, invoke, ", endpoint.DescName(), ".RouteInfo)")
		g.P("}")
		g.P()
	}
	return nil
}

func (gen *Generator) GenerateStreamClientStruct(service *parser.Service, g *protogen.GeneratedFile) error {
	g.P("var _ ", service.StreamClientName(), " = (*", service.StreamClientStructName(), ")(nil)")
	g.P()
	g.P("type ", service.StreamClientStructName(), " struct {")
	g.P("url string")
	g.P("dialOpts *", constant.WebsocketDialOptionsIdent)
	g.P("connCfg *", constant.WsConnConfigIdent)
	g.P("logger *", constant.SlogLoggerIdent)
	g.P("marshalOptions ", constant.ProtoJsonMarshalOptionsIdent)
	g.P("unmarshalOptions ", constant.ProtoJsonUnmarshalOptionsIdent)
	g.P("}")
	g.P()
	return nil
}

func (gen *Generator) GenerateNewStreamClientFunc(service *parser.Service, g *protogen.GeneratedFile) error {
	g.P("func ", service.NewStreamClientName(), "(url string,")
	g.P("logger *", constant.SlogLoggerIdent, ",")
	g.P("marshalOpts ", constant.ProtoJsonMarshalOptionsIdent, ",")
	g.P("unmarshalOpts ", constant.ProtoJsonUnmarshalOptionsIdent, ",")
	g.P("dialOpts *", constant.WebsocketDialOptionsIdent, ",")
	g.P(") ", service.StreamClientName(), " {")
	g.P("if logger == nil {")
	g.P("logger = ", constant.SlogDefaultIdent, "()")
	g.P("}")
	g.P("return &", service.StreamClientStructName(), "{")
	g.P("url: url,")
	g.P("logger: logger,")
	g.P("connCfg: ", constant.WsDefaultConnConfigIdent, "(),")
	g.P("marshalOptions: marshalOpts,")
	g.P("unmarshalOptions: unmarshalOpts,")
	g.P("dialOpts: dialOpts,")
	g.P("}")
	g.P("}")
	g.P()
	return nil
}

func (gen *Generator) GenerateStreamClientMethods(service *parser.Service, g *protogen.GeneratedFile) error {
	for _, endpoint := range service.Endpoints {
		if endpoint.IsClientStreaming() {
			gen.generateClientStreamingMethod(service, endpoint, g)
		} else if endpoint.IsServerStreaming() {
			gen.generateServerStreamingMethod(service, endpoint, g)
		} else if endpoint.IsBidiStreaming() {
			gen.generateBidiStreamingMethod(service, endpoint, g)
		}
	}
	return nil
}

func (gen *Generator) generateClientStreamingMethod(service *parser.Service, endpoint *parser.Endpoint, g *protogen.GeneratedFile) {
	g.P("func (c *", service.StreamClientStructName(), ") ", endpoint.Name(), "(ctx ", constant.ContextIdent, ") (", constant.WsClientStreamingClientIdent, "[*", endpoint.InputGoIdent(), ", *", endpoint.OutputGoIdent(), "], error) {")
	g.P("u, err := ", constant.JoinPathIndent, "(c.url, ", endpoint.DescName(), ".RouteInfo.Pattern)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("conn, err := ", constant.WsDialAndConnectIdent, "(ctx, u, c.dialOpts, c.connCfg, c.logger)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return ", constant.WsNewClientStreamV2Ident, "[*", endpoint.InputGoIdent(), ", *", endpoint.OutputGoIdent(), "](ctx, conn, c.marshalOptions, c.unmarshalOptions), nil")
	g.P("}")
	g.P()
}

func (gen *Generator) generateServerStreamingMethod(service *parser.Service, endpoint *parser.Endpoint, g *protogen.GeneratedFile) {
	g.P("func (c *", service.StreamClientStructName(), ") ", endpoint.Name(), "(ctx ", constant.ContextIdent, ", in *", endpoint.InputGoIdent(), ") (", constant.WsServerStreamingClientIdent, "[*", endpoint.OutputGoIdent(), "], error) {")
	g.P("u, err := ", constant.JoinPathIndent, "(c.url, ", endpoint.DescName(), ".RouteInfo.Pattern)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("conn, err := ", constant.WsDialAndConnectIdent, "(ctx, u, c.dialOpts, c.connCfg, c.logger)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("cs := ", constant.WsNewClientStreamV2Ident, "[*", endpoint.InputGoIdent(), ", *", endpoint.OutputGoIdent(), "](ctx, conn, c.marshalOptions, c.unmarshalOptions)")
	g.P("if err := cs.SendMsg(in); err != nil {")
	g.P("_ = cs.CloseSend()")
	g.P("return nil, err")
	g.P("}")
	g.P("return cs, nil")
	g.P("}")
	g.P()
}

func (gen *Generator) generateBidiStreamingMethod(service *parser.Service, endpoint *parser.Endpoint, g *protogen.GeneratedFile) {
	g.P("func (c *", service.StreamClientStructName(), ") ", endpoint.Name(), "(ctx ", constant.ContextIdent, ") (", constant.WsBidiStreamingClientIdent, "[*", endpoint.InputGoIdent(), ", *", endpoint.OutputGoIdent(), "], error) {")
	g.P("u, err := ", constant.JoinPathIndent, "(c.url, ", endpoint.DescName(), ".RouteInfo.Pattern)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("conn, err := ", constant.WsDialAndConnectIdent, "(ctx, u, c.dialOpts, c.connCfg, c.logger)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return ", constant.WsNewClientStreamV2Ident, "[*", endpoint.InputGoIdent(), ", *", endpoint.OutputGoIdent(), "](ctx, conn, c.marshalOptions, c.unmarshalOptions), nil")
	g.P("}")
	g.P()
}

func strconvQuote(s string) string {
	return `"` + s + `"`
}
