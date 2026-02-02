package client

import (
	"github.com/soyacen/goose/cmd/protoc-gen-goose/constant"
	"github.com/soyacen/goose/cmd/protoc-gen-goose/parser"
	"google.golang.org/protobuf/compiler/protogen"
)

type Generator struct{}

func (f *Generator) GenerateNewClient(service *parser.Service, g *protogen.GeneratedFile) error {
	g.P("func ", service.NewClientName(), "(target string, opts ...", constant.ClientOptionIdent, ") ", service.ServiceName(), " {")
	g.P("options := ", constant.ClientNewOptionsIdent, "(opts...)")
	g.P("client :=  &", service.Unexported(service.ClientName()), "{")
	g.P("client: options.Client(),")
	g.P("encoder: ", service.Unexported(service.RequestEncoderName()), "{")
	g.P("target: target,")
	g.P("marshalOptions: options.MarshalOptions(),")
	g.P("resolver: options.Resolver(),")
	g.P("},")
	g.P("decoder: ", service.Unexported(service.ResponseDecoderName()), "{")
	g.P("unmarshalOptions: options.UnmarshalOptions(),")
	g.P("errorDecoder: options.ErrorDecoder(),")
	g.P("errorFactory: options.ErrorFactory(),")
	g.P("},")
	g.P("shouldFailFast: options.ShouldFailFast(),")
	g.P("onValidationErrCallback: options.OnValidationErrCallback(),")
	g.P("middleware: ", constant.ClientChainIdent, "(options.Middlewares()...),")
	g.P("}")
	g.P("return client")
	g.P("}")
	g.P()
	return nil
}

func (f *Generator) GenerateClient(service *parser.Service, g *protogen.GeneratedFile) error {
	g.P("type ", service.Unexported(service.ClientName()), " struct {")
	g.P("client *", constant.ClientIdent)
	g.P("encoder ", service.Unexported(service.RequestEncoderName()))
	g.P("decoder ", service.Unexported(service.ResponseDecoderName()))
	g.P("shouldFailFast bool")
	g.P("onValidationErrCallback ", constant.OnErrCallbackIdent)
	g.P("middleware ", constant.ClientMiddlewareIdent)
	g.P("}")
	g.P()
	for _, endpoint := range service.Endpoints {
		g.P("func (c *", service.Unexported(service.ClientName()), ") ", endpoint.Name(), "(ctx ", constant.ContextIdent, ", req *", endpoint.InputGoIdent(), ") (*", endpoint.OutputGoIdent(), ", error){")
		g.P("if err := ", constant.ValidateRequestIdent, "(ctx, req, c.shouldFailFast, c.onValidationErrCallback); err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("request, err := c.encoder.", endpoint.Name(), "(ctx, req)")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("response, err := ", constant.ClientInvokeIdent, "(c.middleware, c.client, request, ", endpoint.DescName(), ".RouteInfo)")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("resp, err := c.decoder.", endpoint.Name(), "(ctx, response)")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("return resp, nil")
		g.P("}")
		g.P()
	}
	return nil
}
