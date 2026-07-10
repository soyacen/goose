package parser

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

type Service struct {
	ProtoService *protogen.Service
	Endpoints    []*Endpoint
}

func (s *Service) Unexported(name string) string {
	return strings.ToLower(name[:1]) + name[1:]
}

func (s *Service) FullName() string {
	return string(s.ProtoService.Desc.FullName())
}

func (s *Service) Name() string {
	return s.ProtoService.GoName
}

func (s *Service) ServiceName() string {
	return s.Name() + "Service"
}

func (s *Service) AppendRouteName() string {
	return "Append" + s.Name() + "HttpRoute"
}

func (s *Service) HandlerName() string {
	return s.Name() + "Handler"
}

func (s *Service) RequestDecoderName() string {
	return s.Name() + "RequestDecoder"
}

func (s *Service) ResponseEncoderName() string {
	return s.Name() + "ResponseEncoder"
}

func (s *Service) NewClientName() string {
	return "New" + s.Name() + "HttpClient"
}

func (s *Service) ClientName() string {
	return s.Name() + "HttpClient"
}

func (s *Service) RequestEncoderName() string {
	return s.Name() + "RequestEncoder"
}

func (s *Service) ResponseDecoderName() string {
	return s.Name() + "ResponseDecoder"
}

func (s *Service) IsStreamingService() bool {
	for _, endpoint := range s.Endpoints {
		if endpoint.IsStreaming() {
			return true
		}
	}
	return false
}

func (s *Service) StreamServerName() string {
	return s.Name() + "StreamServer"
}

func (s *Service) StreamClientName() string {
	return s.Name() + "StreamClient"
}

func (s *Service) AppendStreamRouteName() string {
	return "Append" + s.Name() + "WebsocketRoute"
}

func (s *Service) StreamHandlerName() string {
	return s.Unexported(s.Name()) + "StreamHandler"
}

func (s *Service) StreamClientStructName() string {
	return s.Unexported(s.Name()) + "StreamClient"
}

func (s *Service) NewStreamClientName() string {
	return "New" + s.Name() + "StreamClient"
}

func NewServices(file *protogen.File) ([]*Service, error) {
	var services []*Service
	for _, pbService := range file.Services {
		service := &Service{
			ProtoService: pbService,
		}
		var endpoints []*Endpoint
		for _, pbMethod := range pbService.Methods {
			endpoint := &Endpoint{
				protoMethod: pbMethod,
			}
			endpoint.SetHttpRule()
			pattern, err := ParsePattern(endpoint.Path())
			if err != nil {
				return nil, fmt.Errorf("goose: %s", err)
			}
			endpoint.SetPattern(pattern)
			endpoints = append(endpoints, endpoint)
		}
		service.Endpoints = endpoints
		services = append(services, service)
	}
	return services, nil
}
