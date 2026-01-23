package path

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	httpbody "google.golang.org/genproto/googleapis/api/httpbody"
	protojson "google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// ---- Mock Services ----

type MockBoolPathService struct{}

func (m *MockBoolPathService) BoolPath(ctx context.Context, req *BoolPathRequest) (*httpbody.HttpBody, error) {
	data, err := protojson.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &httpbody.HttpBody{Data: data}, nil
}

type MockInt32PathService struct{}

func (m *MockInt32PathService) Int32Path(ctx context.Context, req *Int32PathRequest) (*httpbody.HttpBody, error) {
	data, err := protojson.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &httpbody.HttpBody{Data: data}, nil
}

type MockInt64PathService struct{}

func (m *MockInt64PathService) Int64Path(ctx context.Context, req *Int64PathRequest) (*httpbody.HttpBody, error) {
	data, err := protojson.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &httpbody.HttpBody{Data: data}, nil
}

type MockUint32PathService struct{}

func (m *MockUint32PathService) Uint32Path(ctx context.Context, req *Uint32PathRequest) (*httpbody.HttpBody, error) {
	data, err := protojson.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &httpbody.HttpBody{Data: data}, nil
}

type MockUint64PathService struct{}

func (m *MockUint64PathService) Uint64Path(ctx context.Context, req *Uint64PathRequest) (*httpbody.HttpBody, error) {
	data, err := protojson.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &httpbody.HttpBody{Data: data}, nil
}

type MockFloatPathService struct{}

func (m *MockFloatPathService) FloatPath(ctx context.Context, req *FloatPathRequest) (*httpbody.HttpBody, error) {
	data, err := protojson.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &httpbody.HttpBody{Data: data}, nil
}

type MockDoublePathService struct{}

func (m *MockDoublePathService) DoublePath(ctx context.Context, req *DoublePathRequest) (*httpbody.HttpBody, error) {
	data, err := protojson.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &httpbody.HttpBody{Data: data}, nil
}

type MockStringPathService struct{}

func (m *MockStringPathService) StringPath(ctx context.Context, req *StringPathRequest) (*httpbody.HttpBody, error) {
	data, err := protojson.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &httpbody.HttpBody{Data: data}, nil
}

type MockEnumPathService struct{}

func (m *MockEnumPathService) EnumPath(ctx context.Context, req *EnumPathRequest) (*httpbody.HttpBody, error) {
	data, err := protojson.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &httpbody.HttpBody{Data: data}, nil
}

// ---- Test Cases ----

func TestBoolPath(t *testing.T) {
	server := &http.Server{}
	go func() {
		router := http.NewServeMux()
		router = AppendBoolPathRoute(router, &MockBoolPathService{})
		server.Addr = ":48081"
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	defer server.Close()
	cli := NewBoolPathClient("http://localhost:48081")
	resp, err := cli.BoolPath(context.Background(), &BoolPathRequest{
		Bool:     true,
		OptBool:  proto.Bool(true),
		WrapBool: wrapperspb.Bool(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"bool":true,"optBool":true,"wrapBool":true}`
	if strings.ReplaceAll(string(resp.GetData()), " ", "") != strings.ReplaceAll(expected, " ", "") {
		t.Fatalf("body is not equal: got %s, want %s", string(resp.GetData()), expected)
	}
}

func TestInt32Path(t *testing.T) {
	server := &http.Server{}
	go func() {
		router := http.NewServeMux()
		router = AppendInt32PathRoute(router, &MockInt32PathService{})
		server.Addr = ":48082"
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	defer server.Close()
	cli := NewInt32PathClient("http://localhost:48082")
	resp, err := cli.Int32Path(context.Background(), &Int32PathRequest{
		Int32:       1,
		Sint32:      2,
		Sfixed32:    3,
		OptInt32:    proto.Int32(4),
		OptSint32:   proto.Int32(5),
		OptSfixed32: proto.Int32(6),
		WrapInt32:   wrapperspb.Int32(7),
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"int32":1,"sint32":2,"sfixed32":3,"optInt32":4,"optSint32":5,"optSfixed32":6,"wrapInt32":7}`
	if strings.ReplaceAll(string(resp.GetData()), " ", "") != strings.ReplaceAll(expected, " ", "") {
		t.Fatalf("body is not equal: got %s, want %s", string(resp.GetData()), expected)
	}
}

func TestInt64Path(t *testing.T) {
	server := &http.Server{}
	go func() {
		router := http.NewServeMux()
		router = AppendInt64PathRoute(router, &MockInt64PathService{})
		server.Addr = ":48083"
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	defer server.Close()
	cli := NewInt64PathClient("http://localhost:48083")
	resp, err := cli.Int64Path(context.Background(), &Int64PathRequest{
		Int64:       10,
		Sint64:      20,
		Sfixed64:    30,
		OptInt64:    proto.Int64(40),
		OptSint64:   proto.Int64(50),
		OptSfixed64: proto.Int64(60),
		WrapInt64:   wrapperspb.Int64(70),
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"int64":"10","sint64":"20","sfixed64":"30","optInt64":"40","optSint64":"50","optSfixed64":"60","wrapInt64":"70"}`
	if strings.ReplaceAll(string(resp.GetData()), " ", "") != strings.ReplaceAll(expected, " ", "") {
		t.Fatalf("body is not equal: got %s, want %s", string(resp.GetData()), expected)
	}
}

func TestUint32Path(t *testing.T) {
	server := &http.Server{}
	go func() {
		router := http.NewServeMux()
		router = AppendUint32PathRoute(router, &MockUint32PathService{})
		server.Addr = ":48084"
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	defer server.Close()
	cli := NewUint32PathClient("http://localhost:48084")
	resp, err := cli.Uint32Path(context.Background(), &Uint32PathRequest{
		Uint32:     1,
		Fixed32:    2,
		OptUint32:  proto.Uint32(3),
		OptFixed32: proto.Uint32(4),
		WrapUint32: wrapperspb.UInt32(5),
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"uint32":1,"fixed32":2,"optUint32":3,"optFixed32":4,"wrapUint32":5}`
	if strings.ReplaceAll(string(resp.GetData()), " ", "") != strings.ReplaceAll(expected, " ", "") {
		t.Fatalf("body is not equal: got %s, want %s", string(resp.GetData()), expected)
	}
}

func TestUint64Path(t *testing.T) {
	server := &http.Server{}
	go func() {
		router := http.NewServeMux()
		router = AppendUint64PathRoute(router, &MockUint64PathService{})
		server.Addr = ":48085"
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	defer server.Close()
	cli := NewUint64PathClient("http://localhost:48085")
	resp, err := cli.Uint64Path(context.Background(), &Uint64PathRequest{
		Uint64:     10,
		Fixed64:    20,
		OptUint64:  proto.Uint64(30),
		OptFixed64: proto.Uint64(40),
		WrapUint64: wrapperspb.UInt64(50),
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"uint64":"10","fixed64":"20","optUint64":"30","optFixed64":"40","wrapUint64":"50"}`
	if strings.ReplaceAll(string(resp.GetData()), " ", "") != strings.ReplaceAll(expected, " ", "") {
		t.Fatalf("body is not equal: got %s, want %s", string(resp.GetData()), expected)
	}
}

func TestFloatPath(t *testing.T) {
	server := &http.Server{}
	go func() {
		router := http.NewServeMux()
		router = AppendFloatPathRoute(router, &MockFloatPathService{})
		server.Addr = ":48086"
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	defer server.Close()
	cli := NewFloatPathClient("http://localhost:48086")
	resp, err := cli.FloatPath(context.Background(), &FloatPathRequest{
		Float:     1.23,
		OptFloat:  proto.Float32(4.56),
		WrapFloat: wrapperspb.Float(7.89),
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"float":1.23,"optFloat":4.56,"wrapFloat":7.89}`
	if strings.ReplaceAll(string(resp.GetData()), " ", "") != strings.ReplaceAll(expected, " ", "") {
		t.Fatalf("body is not equal: got %s, want %s", string(resp.GetData()), expected)
	}
}

func TestDoublePath(t *testing.T) {
	server := &http.Server{}
	go func() {
		router := http.NewServeMux()
		router = AppendDoublePathRoute(router, &MockDoublePathService{})
		server.Addr = ":48087"
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	defer server.Close()
	cli := NewDoublePathClient("http://localhost:48087")
	resp, err := cli.DoublePath(context.Background(), &DoublePathRequest{
		Double:     1.23,
		OptDouble:  proto.Float64(4.56),
		WrapDouble: wrapperspb.Double(7.89),
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"double":1.23,"optDouble":4.56,"wrapDouble":7.89}`
	if strings.ReplaceAll(string(resp.GetData()), " ", "") != strings.ReplaceAll(expected, " ", "") {
		t.Fatalf("body is not equal: got %s, want %s", string(resp.GetData()), expected)
	}
}

func TestStringPath(t *testing.T) {
	server := &http.Server{}
	go func() {
		router := http.NewServeMux()
		router = AppendStringPathRoute(router, &MockStringPathService{})
		server.Addr = ":48088"
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	defer server.Close()
	cli := NewStringPathClient("http://localhost:48088")
	resp, err := cli.StringPath(context.Background(), &StringPathRequest{
		String_:     "abc",
		OptString:   proto.String("def"),
		WrapString:  wrapperspb.String("ghi"),
		MultiString: "opq/rst/uv",
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"string":"abc","optString":"def","wrapString":"ghi","multiString":"opq/rst/uv"}`
	if strings.ReplaceAll(string(resp.GetData()), " ", "") != strings.ReplaceAll(expected, " ", "") {
		t.Fatalf("body is not equal: got %s, want %s", string(resp.GetData()), expected)
	}
}

func TestEnumPath(t *testing.T) {
	server := &http.Server{}
	go func() {
		router := http.NewServeMux()
		router = AppendEnumPathRoute(router, &MockEnumPathService{})
		server.Addr = ":48089"
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	defer server.Close()
	cli := NewEnumPathClient("http://localhost:48089")
	canceled := EnumPathRequest_CANCELLED
	resp, err := cli.EnumPath(context.Background(), &EnumPathRequest{
		Status:    EnumPathRequest_OK,
		OptStatus: &canceled,
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"status":"OK","optStatus":"CANCELLED"}`
	if strings.ReplaceAll(string(resp.GetData()), " ", "") != strings.ReplaceAll(expected, " ", "") {
		t.Fatalf("body is not equal: got %s, want %s", string(resp.GetData()), expected)
	}
}
