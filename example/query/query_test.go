package query

import (
	"context"
	errors "errors"
	"net/http"
	"strings"
	"testing"
	"time"

	httpbody "google.golang.org/genproto/googleapis/api/httpbody"
	protojson "google.golang.org/protobuf/encoding/protojson"
	proto "google.golang.org/protobuf/proto"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

// ---- Mock Services ----

type MockBoolQueryService struct{}

func (m *MockBoolQueryService) BoolQuery(ctx context.Context, req *BoolQueryRequest) (*httpbody.HttpBody, error) {
	data, err := protojson.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &httpbody.HttpBody{Data: data}, nil
}

type MockInt32QueryService struct{}

func (m *MockInt32QueryService) Int32Query(ctx context.Context, req *Int32QueryRequest) (*httpbody.HttpBody, error) {
	data, err := protojson.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &httpbody.HttpBody{Data: data}, nil
}

type MockInt64QueryService struct{}

func (m *MockInt64QueryService) Int64Query(ctx context.Context, req *Int64QueryRequest) (*httpbody.HttpBody, error) {
	data, err := protojson.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &httpbody.HttpBody{Data: data}, nil
}

type MockUint32QueryService struct{}

func (m *MockUint32QueryService) Uint32Query(ctx context.Context, req *Uint32QueryRequest) (*httpbody.HttpBody, error) {
	data, err := protojson.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &httpbody.HttpBody{Data: data}, nil
}

type MockUint64QueryService struct{}

func (m *MockUint64QueryService) Uint64Query(ctx context.Context, req *Uint64QueryRequest) (*httpbody.HttpBody, error) {
	data, err := protojson.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &httpbody.HttpBody{Data: data}, nil
}

type MockFloatQueryService struct{}

func (m *MockFloatQueryService) FloatQuery(ctx context.Context, req *FloatQueryRequest) (*httpbody.HttpBody, error) {
	data, err := protojson.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &httpbody.HttpBody{Data: data}, nil
}

type MockDoubleQueryService struct{}

func (m *MockDoubleQueryService) DoubleQuery(ctx context.Context, req *DoubleQueryRequest) (*httpbody.HttpBody, error) {
	data, err := protojson.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &httpbody.HttpBody{Data: data}, nil
}

type MockStringQueryService struct{}

func (m *MockStringQueryService) StringQuery(ctx context.Context, req *StringQueryRequest) (*httpbody.HttpBody, error) {
	data, err := protojson.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &httpbody.HttpBody{Data: data}, nil
}

type MockEnumQueryService struct{}

func (m *MockEnumQueryService) EnumQuery(ctx context.Context, req *EnumQueryRequest) (*httpbody.HttpBody, error) {
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
		router = AppendBoolQueryRoute(router, &MockBoolQueryService{})
		server.Addr = ":28081"
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	defer server.Close()
	time.Sleep(time.Second)
	cli := NewBoolQueryClient("http://localhost:28081")
	resp, err := cli.BoolQuery(context.Background(), &BoolQueryRequest{
		Bool:         true,
		OptBool:      proto.Bool(true),
		WrapBool:     wrapperspb.Bool(true),
		ListBool:     []bool{true, false},
		ListWrapBool: []*wrapperspb.BoolValue{wrapperspb.Bool(true), wrapperspb.Bool(false)},
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"bool":true, "optBool":true, "wrapBool":true, "listBool":[true, false], "listWrapBool":[true, false]}`
	if strings.ReplaceAll(string(resp.GetData()), " ", "") != strings.ReplaceAll(expected, " ", "") {
		t.Fatalf("body is not equal: got %s, want %s", string(resp.GetData()), expected)
	}
}

func TestInt32Path(t *testing.T) {
	server := &http.Server{}
	go func() {
		router := http.NewServeMux()
		router = AppendInt32QueryRoute(router, &MockInt32QueryService{})
		server.Addr = ":28082"
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	defer server.Close()
	time.Sleep(time.Second)
	cli := NewInt32QueryClient("http://localhost:28082")
	resp, err := cli.Int32Query(context.Background(), &Int32QueryRequest{
		Int32:         1,
		Sint32:        2,
		Sfixed32:      3,
		OptInt32:      proto.Int32(4),
		OptSint32:     proto.Int32(5),
		OptSfixed32:   proto.Int32(6),
		WrapInt32:     wrapperspb.Int32(7),
		ListInt32:     []int32{1, 2},
		ListSint32:    []int32{1, 2},
		ListSfixed32:  []int32{1, 2},
		ListWrapInt32: []*wrapperspb.Int32Value{wrapperspb.Int32(1), wrapperspb.Int32(2)},
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"int32":1,"sint32":2,"sfixed32":3,"optInt32":4,"optSint32":5,"optSfixed32":6,"wrapInt32":7,"listInt32":[1,2],"listSint32":[1,2],"listSfixed32":[1,2],"listWrapInt32":[1,2]}`
	if strings.ReplaceAll(string(resp.GetData()), " ", "") != strings.ReplaceAll(expected, " ", "") {
		t.Fatalf("body is not equal: got %s, want %s", string(resp.GetData()), expected)
	}
}

func TestInt64Path(t *testing.T) {
	server := &http.Server{}
	go func() {
		router := http.NewServeMux()
		router = AppendInt64QueryRoute(router, &MockInt64QueryService{})
		server.Addr = ":28083"
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	defer server.Close()
	time.Sleep(time.Second)
	cli := NewInt64QueryClient("http://localhost:28083")
	resp, err := cli.Int64Query(context.Background(), &Int64QueryRequest{
		Int64:         10,
		Sint64:        20,
		Sfixed64:      30,
		OptInt64:      proto.Int64(40),
		OptSint64:     proto.Int64(50),
		OptSfixed64:   proto.Int64(60),
		WrapInt64:     wrapperspb.Int64(70),
		ListInt64:     []int64{1, 2},
		ListSint64:    []int64{1, 2},
		ListSfixed64:  []int64{1, 2},
		ListWrapInt64: []*wrapperspb.Int64Value{wrapperspb.Int64(1), wrapperspb.Int64(2)},
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"int64":"10", "sint64":"20", "sfixed64":"30", "optInt64":"40", "optSint64":"50", "optSfixed64":"60", "wrapInt64":"70", "listInt64":["1", "2"], "listSint64":["1", "2"], "listSfixed64":["1", "2"], "listWrapInt64":["1", "2"]}`
	if strings.ReplaceAll(string(resp.GetData()), " ", "") != strings.ReplaceAll(expected, " ", "") {
		t.Fatalf("body is not equal: got %s, want %s", string(resp.GetData()), expected)
	}
}

func TestUint32Path(t *testing.T) {
	server := &http.Server{}
	go func() {
		router := http.NewServeMux()
		router = AppendUint32QueryRoute(router, &MockUint32QueryService{})
		server.Addr = ":28084"
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	defer server.Close()
	time.Sleep(time.Second)
	cli := NewUint32QueryClient("http://localhost:28084")
	resp, err := cli.Uint32Query(context.Background(), &Uint32QueryRequest{
		Uint32:         1,
		Fixed32:        2,
		OptUint32:      proto.Uint32(3),
		OptFixed32:     proto.Uint32(4),
		WrapUint32:     wrapperspb.UInt32(5),
		ListUint32:     []uint32{1, 2},
		ListFixed32:    []uint32{1, 2},
		ListWrapUint32: []*wrapperspb.UInt32Value{wrapperspb.UInt32(1), wrapperspb.UInt32(2)},
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"uint32":1,"fixed32":2,"optUint32":3,"optFixed32":4,"wrapUint32":5,"listUint32":[1,2],"listFixed32":[1,2],"listWrapUint32":[1,2]}`
	if strings.ReplaceAll(string(resp.GetData()), " ", "") != strings.ReplaceAll(expected, " ", "") {
		t.Fatalf("body is not equal: got %s, want %s", string(resp.GetData()), expected)
	}
}

func TestUint64Path(t *testing.T) {
	server := &http.Server{}
	go func() {
		router := http.NewServeMux()
		router = AppendUint64QueryRoute(router, &MockUint64QueryService{})
		server.Addr = ":28085"
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	defer server.Close()
	time.Sleep(time.Second)
	cli := NewUint64QueryClient("http://localhost:28085")
	resp, err := cli.Uint64Query(context.Background(), &Uint64QueryRequest{
		Uint64:         10,
		Fixed64:        20,
		OptUint64:      proto.Uint64(30),
		OptFixed64:     proto.Uint64(40),
		WrapUint64:     wrapperspb.UInt64(50),
		ListUint64:     []uint64{1, 2},
		ListFixed64:    []uint64{1, 2},
		ListWrapUint64: []*wrapperspb.UInt64Value{wrapperspb.UInt64(1), wrapperspb.UInt64(2)},
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"uint64":"10", "fixed64":"20", "optUint64":"30", "optFixed64":"40", "wrapUint64":"50", "listUint64":["1", "2"], "listFixed64":["1", "2"], "listWrapUint64":["1", "2"]}`
	if strings.ReplaceAll(string(resp.GetData()), " ", "") != strings.ReplaceAll(expected, " ", "") {
		t.Fatalf("body is not equal: got %s, want %s", string(resp.GetData()), expected)
	}
}

func TestFloatPath(t *testing.T) {
	server := &http.Server{}
	go func() {
		router := http.NewServeMux()
		router = AppendFloatQueryRoute(router, &MockFloatQueryService{})
		server.Addr = ":28086"
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	defer server.Close()
	time.Sleep(time.Second)
	cli := NewFloatQueryClient("http://localhost:28086")
	resp, err := cli.FloatQuery(context.Background(), &FloatQueryRequest{
		Float:         1.23,
		OptFloat:      proto.Float32(4.56),
		WrapFloat:     wrapperspb.Float(7.89),
		ListFloat:     []float32{1.23, 3.45},
		ListWrapFloat: []*wrapperspb.FloatValue{wrapperspb.Float(4.32), wrapperspb.Float(5.66)},
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"float":1.23, "optFloat":4.56, "wrapFloat":7.89, "listFloat":[1.23, 3.45], "listWrapFloat":[4.32, 5.66]}`
	if strings.ReplaceAll(string(resp.GetData()), " ", "") != strings.ReplaceAll(expected, " ", "") {
		t.Fatalf("body is not equal: got %s, want %s", string(resp.GetData()), expected)
	}
}

func TestDoublePath(t *testing.T) {
	server := &http.Server{}
	go func() {
		router := http.NewServeMux()
		router = AppendDoubleQueryRoute(router, &MockDoubleQueryService{})
		server.Addr = ":28087"
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	defer server.Close()
	time.Sleep(time.Second)
	cli := NewDoubleQueryClient("http://localhost:28087")
	resp, err := cli.DoubleQuery(context.Background(), &DoubleQueryRequest{
		Double:         1.23,
		OptDouble:      proto.Float64(4.56),
		WrapDouble:     wrapperspb.Double(7.89),
		ListDouble:     []float64{1.23, 3.45},
		ListWrapDouble: []*wrapperspb.DoubleValue{wrapperspb.Double(4.32), wrapperspb.Double(5.66)},
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"double":1.23,"optDouble":4.56,"wrapDouble":7.89,"listDouble":[1.23,3.45],"listWrapDouble":[4.32,5.66]}`
	if strings.ReplaceAll(string(resp.GetData()), " ", "") != strings.ReplaceAll(expected, " ", "") {
		t.Fatalf("body is not equal: got %s, want %s", string(resp.GetData()), expected)
	}
}

func TestStringPath(t *testing.T) {
	server := &http.Server{}
	go func() {
		router := http.NewServeMux()
		router = AppendStringQueryRoute(router, &MockStringQueryService{})
		server.Addr = ":28088"
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	defer server.Close()
	time.Sleep(time.Second)
	cli := NewStringQueryClient("http://localhost:28088")
	resp, err := cli.StringQuery(context.Background(), &StringQueryRequest{
		String_:        "abc",
		OptString:      proto.String("def"),
		WrapString:     wrapperspb.String("ghi"),
		ListString:     []string{"d3d", "lo-"},
		ListWrapString: []*wrapperspb.StringValue{wrapperspb.String("<>d"), wrapperspb.String("{[]}")},
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"string":"abc","optString":"def","wrapString":"ghi","listString":["d3d","lo-"],"listWrapString":["<>d","{[]}"]}`
	if strings.ReplaceAll(string(resp.GetData()), " ", "") != strings.ReplaceAll(expected, " ", "") {
		t.Fatalf("body is not equal: got %s, want %s", string(resp.GetData()), expected)
	}
}

func TestEnumPath(t *testing.T) {
	server := &http.Server{}
	go func() {
		router := http.NewServeMux()
		router = AppendEnumQueryRoute(router, &MockEnumQueryService{})
		server.Addr = ":28089"
		server.Handler = router
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	defer server.Close()
	time.Sleep(time.Second)
	cli := NewEnumQueryClient("http://localhost:28089")
	canceled := EnumQueryRequest_CANCELLED
	resp, err := cli.EnumQuery(context.Background(), &EnumQueryRequest{
		Status:     EnumQueryRequest_OK,
		OptStatus:  &canceled,
		ListStatus: []EnumQueryRequest_Status{EnumQueryRequest_OK, EnumQueryRequest_CANCELLED},
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"status":"OK", "optStatus":"CANCELLED", "listStatus":["OK", "CANCELLED"]}`
	if strings.ReplaceAll(string(resp.GetData()), " ", "") != strings.ReplaceAll(expected, " ", "") {
		t.Fatalf("body is not equal: got %s, want %s", string(resp.GetData()), expected)
	}
}
