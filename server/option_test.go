package server

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/soyacen/goose"
	"google.golang.org/protobuf/encoding/protojson"
)

func dummyErrorEncoder(ctx context.Context, err error, w http.ResponseWriter) {
}

func TestOptions_Defaults(t *testing.T) {
	opts := NewOptions()

	// 检查默认值
	if !reflect.DeepEqual(opts.UnmarshalOptions(), protojson.UnmarshalOptions{}) {
		t.Errorf("default UnmarshalOptions not empty")
	}
	if !reflect.DeepEqual(opts.MarshalOptions(), protojson.MarshalOptions{}) {
		t.Errorf("default MarshalOptions not empty")
	}
	if opts.ErrorEncoder() == nil {
		t.Errorf("default ErrorEncoder is nil")
	}
}

func TestOptions_WithOptions(t *testing.T) {
	unmarshalOpt := protojson.UnmarshalOptions{AllowPartial: true}
	marshalOpt := protojson.MarshalOptions{EmitUnpopulated: true}
	var customErrorEncoder goose.ErrorEncoder = dummyErrorEncoder

	opts := NewOptions(
		UnmarshalOptions(unmarshalOpt),
		MarshalOptions(marshalOpt),
		ErrorEncoder(customErrorEncoder),
	)

	if !reflect.DeepEqual(opts.UnmarshalOptions(), unmarshalOpt) {
		t.Errorf("UnmarshalOptions not set correctly")
	}
	if !reflect.DeepEqual(opts.MarshalOptions(), marshalOpt) {
		t.Errorf("MarshalOptions not set correctly")
	}
	// if opts.ErrorEncoder() != customErrorEncoder {
	// 	t.Errorf("ErrorEncoder not set correctly")
	// }
}

func TestOptions_Apply(t *testing.T) {
	o := &options{}
	opt1 := UnmarshalOptions(protojson.UnmarshalOptions{DiscardUnknown: true})
	opt2 := MarshalOptions(protojson.MarshalOptions{UseProtoNames: true})
	o.apply(opt1, opt2)

	if !o.unmarshalOptions.DiscardUnknown {
		t.Errorf("Apply did not set UnmarshalOptions.DiscardUnknown")
	}
	if !o.marshalOptions.UseProtoNames {
		t.Errorf("Apply did not set MarshalOptions.UseProtoNames")
	}
}
