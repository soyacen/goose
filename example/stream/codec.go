package main

import "encoding/json"

// Codec defines the interface for message serialization and deserialization.
// In a gRPC-like system, proto.Marshal/Unmarshal is typically used.
// For this WebSocket example, JSON serves as the default codec.
//
// protoc-gen-goose would generate code that accepts any Codec implementation,
// allowing users to swap JSON for protobuf, msgpack, or any other format.
type Codec interface {
	// Marshal serializes v into a byte slice.
	Marshal(v any) ([]byte, error)
	// Unmarshal deserializes data into v.
	Unmarshal(data []byte, v any) error
}

// JSONCodec is the default Codec implementation that uses encoding/json.
type JSONCodec struct{}

// Marshal implements Codec.
func (JSONCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal implements Codec.
func (JSONCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
