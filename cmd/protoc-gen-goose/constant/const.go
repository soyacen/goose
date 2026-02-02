package constant

import (
	"google.golang.org/protobuf/compiler/protogen"
)

var (
	StatusOK  = HttpPackage.Ident("StatusOK")
	StatusOK2 = HttpPackage.Ident("StatusOK")
)

var (
	ErrorsPackage = protogen.GoImportPath("errors")
	NewErrorIdent = ErrorsPackage.Ident("New")
)

var (
	UrlPackage      = protogen.GoImportPath("net/url")
	URLIndent       = UrlPackage.Ident("URL")
	URLParseIndent  = UrlPackage.Ident("Parse")
	URLValuesIndent = UrlPackage.Ident("Values")
	JoinPathIndent  = UrlPackage.Ident("JoinPath")
)

var (
	BytesPackage = protogen.GoImportPath("bytes")
	Buffer       = BytesPackage.Ident("Buffer")
)

var (
	ProtoJsonPackage               = protogen.GoImportPath("google.golang.org/protobuf/encoding/protojson")
	ProtoJsonMarshalOptionsIdent   = ProtoJsonPackage.Ident("MarshalOptions")
	ProtoJsonUnmarshalOptionsIdent = ProtoJsonPackage.Ident("UnmarshalOptions")
)

var (
	ContextPackage = protogen.GoImportPath("context")
	ContextIdent   = ContextPackage.Ident("Context")
)

var (
	HttpPackage                 = protogen.GoImportPath("net/http")
	RouterIdent                 = HttpPackage.Ident("ServeMux")
	ClientIdent                 = HttpPackage.Ident("Client")
	HttpHandlerIdent            = HttpPackage.Ident("Handler")
	HttpHandlerFuncIdent        = HttpPackage.Ident("HandlerFunc")
	ResponseWriterIdent         = HttpPackage.Ident("ResponseWriter")
	RequestIdent                = HttpPackage.Ident("Request")
	ResponseIdent               = HttpPackage.Ident("Response")
	Handler                     = HttpPackage.Ident("Handler")
	Header                      = HttpPackage.Ident("Header")
	NewRequestWithContextIndent = HttpPackage.Ident("NewRequestWithContext")
)

var (
	FmtPackage   = protogen.GoImportPath("fmt")
	SprintfIdent = FmtPackage.Ident("Sprintf")
)

var (
	ProtoPackage     = protogen.GoImportPath("google.golang.org/protobuf/proto")
	ProtoStringIdent = ProtoPackage.Ident("String")
)

var (
	GoosePackage = protogen.GoImportPath("github.com/soyacen/goose")

	ValidateRequestIdent = GoosePackage.Ident("ValidateRequest")
	OnErrCallbackIdent   = GoosePackage.Ident("OnValidationErrCallback")

	ErrorEncoderIdent = GoosePackage.Ident("ErrorEncoder")
	ErrorDecoderIdent = GoosePackage.Ident("ErrorDecoder")
	ErrorFactoryIdent = GoosePackage.Ident("ErrorFactory")

	URLPathIdent = GoosePackage.Ident("URLPath")

	CopyHeaderIdent = GoosePackage.Ident("CopyHeader")

	FormFromPathIdent = GoosePackage.Ident("FormFromPath")

	FormatBoolIdent       = GoosePackage.Ident("FormatBool")
	FormatBoolSliceIdent  = GoosePackage.Ident("FormatBoolSlice")
	FormatIntIdent        = GoosePackage.Ident("FormatInt")
	FormatIntSliceIdent   = GoosePackage.Ident("FormatIntSlice")
	FormatUintIdent       = GoosePackage.Ident("FormatUint")
	FormatUintSliceIdent  = GoosePackage.Ident("FormatUintSlice")
	FormatFloatIdent      = GoosePackage.Ident("FormatFloat")
	FormatFloatSliceIdent = GoosePackage.Ident("FormatFloatSlice")

	UnwrapBoolSliceIdent    = GoosePackage.Ident("UnwrapBoolSlice")
	UnwrapInt32SliceIdent   = GoosePackage.Ident("UnwrapInt32Slice")
	UnwrapUint32SliceIdent  = GoosePackage.Ident("UnwrapUint32Slice")
	UnwrapInt64SliceIdent   = GoosePackage.Ident("UnwrapInt64Slice")
	UnwrapUint64SliceIdent  = GoosePackage.Ident("UnwrapUint64Slice")
	UnwrapFloat32SliceIdent = GoosePackage.Ident("UnwrapFloat32Slice")
	UnwrapFloat64SliceIdent = GoosePackage.Ident("UnwrapFloat64Slice")
	UnwrapStringSliceIdent  = GoosePackage.Ident("UnwrapStringSlice")

	GetBoolIdent           = GoosePackage.Ident("GetBool")
	GetBoolPtrIdent        = GoosePackage.Ident("GetBoolPtr")
	GetBoolSliceIdent      = GoosePackage.Ident("GetBoolSlice")
	GetBoolValueIdent      = GoosePackage.Ident("GetBoolValue")
	GetBoolValueSliceIdent = GoosePackage.Ident("GetBoolValueSlice")

	GetIntIdent      = GoosePackage.Ident("GetInt")
	GetIntPtrIdent   = GoosePackage.Ident("GetIntPtr")
	GetIntSliceIdent = GoosePackage.Ident("GetIntSlice")

	GetInt32ValueIdent      = GoosePackage.Ident("GetInt32Value")
	GetInt32ValueSliceIdent = GoosePackage.Ident("GetInt32ValueSlice")

	GetInt64ValueIdent      = GoosePackage.Ident("GetInt64Value")
	GetInt64ValueSliceIdent = GoosePackage.Ident("GetInt64ValueSlice")

	GetUintIdent      = GoosePackage.Ident("GetUint")
	GetUintPtrIdent   = GoosePackage.Ident("GetUintPtr")
	GetUintSliceIdent = GoosePackage.Ident("GetUintSlice")

	GetUint32ValueIdent      = GoosePackage.Ident("GetUint32Value")
	GetUint32ValueSliceIdent = GoosePackage.Ident("GetUint32ValueSlice")

	GetUint64ValueIdent      = GoosePackage.Ident("GetUint64Value")
	GetUint64ValueSliceIdent = GoosePackage.Ident("GetUint64ValueSlice")

	GetFloatIdent      = GoosePackage.Ident("GetFloat")
	GetFloatPtrIdent   = GoosePackage.Ident("GetFloatPtr")
	GetFloatSliceIdent = GoosePackage.Ident("GetFloatSlice")

	GetFloat32ValueIdent      = GoosePackage.Ident("GetFloat32Value")
	GetFloat32ValueSliceIdent = GoosePackage.Ident("GetFloat32ValueSlice")

	GetFloat64ValueIdent      = GoosePackage.Ident("GetFloat64Value")
	GetFloat64ValueSliceIdent = GoosePackage.Ident("GetFloat64ValueSlice")

	WrapStringSliceIdent = GoosePackage.Ident("WrapStringSlice")

	GetFormIdent = GoosePackage.Ident("GetForm")

	RouteInfoIdent = GoosePackage.Ident("RouteInfo")
	DescIdent      = GoosePackage.Ident("Desc")
)

func GetEnumIdent(g *protogen.GeneratedFile, ident protogen.GoIdent) protogen.GoIdent {
	return GoosePackage.Ident("GetInt[" + g.QualifiedGoIdent(ident) + "]")
}

func GetEnumPtrIdent(g *protogen.GeneratedFile, ident protogen.GoIdent) protogen.GoIdent {
	return GoosePackage.Ident("GetIntPtr[" + g.QualifiedGoIdent(ident) + "]")
}

func GetEnumSliceIdent(g *protogen.GeneratedFile, ident protogen.GoIdent) protogen.GoIdent {
	return GoosePackage.Ident("GetIntSlice[" + g.QualifiedGoIdent(ident) + "]")
}

var (
	GooseServerPackage       = protogen.GoImportPath("github.com/soyacen/goose/server")
	EncodeResponseIdent      = GooseServerPackage.Ident("EncodeResponse")
	EncodeHttpBodyIdent      = GooseServerPackage.Ident("EncodeHttpBody")
	EncodeHttpResponseIdent  = GooseServerPackage.Ident("EncodeHttpResponse")
	DecodeRequestIdent       = GooseServerPackage.Ident("DecodeRequest")
	DecodeHttpBodyIdent      = GooseServerPackage.Ident("DecodeHttpBody")
	DecodeHttpRequestIdent   = GooseServerPackage.Ident("DecodeHttpRequest")
	CustomDecodeRequestIdent = GooseServerPackage.Ident("CustomDecodeRequest")

	ServerOptionIdent     = GooseServerPackage.Ident("Option")
	ServerNewOptionsIdent = GooseServerPackage.Ident("NewOptions")

	ServerChainIdent      = GooseServerPackage.Ident("Chain")
	ServerInvokeIdent     = GooseServerPackage.Ident("Invoke")
	ServerMiddlewareIdent = GooseServerPackage.Ident("Middleware")
)

var (
	ClientPackage = protogen.GoImportPath("github.com/soyacen/goose/client")

	DecodeMessageIdent              = ClientPackage.Ident("DecodeMessage")
	DecodeHttpBodyFromResponseIdent = ClientPackage.Ident("DecodeHttpBody")
	DecodeHttpResponseIdent         = ClientPackage.Ident("DecodeHttpResponse")
	EncodeHttpBodyToRequestIdent    = ClientPackage.Ident("EncodeHttpBody")
	EncodeHttpRequestIdent          = ClientPackage.Ident("EncodeHttpRequest")
	EncodeMessageIdent              = ClientPackage.Ident("EncodeMessage")

	ClientOptionIdent     = ClientPackage.Ident("Option")
	ClientNewOptionsIdent = ClientPackage.Ident("NewOptions")

	ClientChainIdent      = ClientPackage.Ident("Chain")
	ClientMiddlewareIdent = ClientPackage.Ident("Middleware")
	ClientInvokeIdent     = ClientPackage.Ident("Invoke")
)

var (
	ClientResolverPackage = protogen.GoImportPath("github.com/soyacen/goose/client/resolver")

	ResolverIdent = ClientResolverPackage.Ident("Resolver")
	ResolveIdent  = ClientResolverPackage.Ident("Resolve")
)

var (
	WrapperspbPackage     = protogen.GoImportPath("google.golang.org/protobuf/types/known/wrapperspb")
	WrapperspbStringIdent = WrapperspbPackage.Ident("String")
)
