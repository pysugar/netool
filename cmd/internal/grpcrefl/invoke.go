package grpcrefl

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// InvokeJSON marshals jsonRequest into the message type described by
// methodDesc.Input(), calls the RPC, and returns the JSON-encoded response.
func InvokeJSON(
	ctx context.Context,
	conn *grpc.ClientConn,
	methodDesc protoreflect.MethodDescriptor,
	jsonRequest []byte,
	opts ...grpc.CallOption,
) ([]byte, error) {
	req := dynamicpb.NewMessage(methodDesc.Input())
	if len(jsonRequest) == 0 {
		jsonRequest = []byte("{}")
	}
	if err := protojson.Unmarshal(jsonRequest, req); err != nil {
		return nil, fmt.Errorf("parse request JSON: %w", err)
	}

	res := dynamicpb.NewMessage(methodDesc.Output())
	service := string(methodDesc.Parent().FullName())
	rpc := fmt.Sprintf("/%s/%s", service, methodDesc.Name())
	if err := conn.Invoke(ctx, rpc, req, res, opts...); err != nil {
		return nil, fmt.Errorf("grpc call %s: %w", rpc, err)
	}

	out, err := protojson.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal response JSON: %w", err)
	}
	return out, nil
}
