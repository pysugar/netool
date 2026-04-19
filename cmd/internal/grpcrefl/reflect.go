package grpcrefl

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type reflectStream = grpc.BidiStreamingClient[reflectionpb.ServerReflectionRequest, reflectionpb.ServerReflectionResponse]

// ListServices returns the fully-qualified service names exposed by the
// remote server's reflection endpoint.
func ListServices(ctx context.Context, conn *grpc.ClientConn) ([]string, error) {
	stream, err := openStream(ctx, conn)
	if err != nil {
		return nil, err
	}
	if err := stream.Send(&reflectionpb.ServerReflectionRequest{
		MessageRequest: &reflectionpb.ServerReflectionRequest_ListServices{ListServices: "*"},
	}); err != nil {
		return nil, fmt.Errorf("send list services: %w", err)
	}
	resp, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("recv list services: %w", err)
	}
	if er := resp.GetErrorResponse(); er != nil {
		return nil, fmt.Errorf("server error: code=%d %s", er.ErrorCode, er.ErrorMessage)
	}
	lr := resp.GetListServicesResponse()
	if lr == nil {
		return nil, nil
	}
	names := make([]string, 0, len(lr.GetService()))
	for _, s := range lr.GetService() {
		names = append(names, s.GetName())
	}
	return names, nil
}

// FileDescriptorsForSymbol fetches every FileDescriptorProto whose transitive
// closure contains the requested symbol.
func FileDescriptorsForSymbol(ctx context.Context, conn *grpc.ClientConn, symbol string) ([]protoreflect.FileDescriptor, error) {
	stream, err := openStream(ctx, conn)
	if err != nil {
		return nil, err
	}
	if err := stream.Send(&reflectionpb.ServerReflectionRequest{
		MessageRequest: &reflectionpb.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: symbol},
	}); err != nil {
		return nil, fmt.Errorf("send reflection request: %w", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("recv reflection response: %w", err)
	}
	if er := resp.GetErrorResponse(); er != nil {
		return nil, fmt.Errorf("server error: code=%d %s", er.ErrorCode, er.ErrorMessage)
	}
	fdResp := resp.GetFileDescriptorResponse()
	if fdResp == nil {
		return nil, fmt.Errorf("unexpected response type")
	}
	out := make([]protoreflect.FileDescriptor, 0, len(fdResp.FileDescriptorProto))
	for _, raw := range fdResp.FileDescriptorProto {
		fdProto := &descriptorpb.FileDescriptorProto{}
		if err := proto.Unmarshal(raw, fdProto); err != nil {
			return nil, fmt.Errorf("unmarshal file descriptor: %w", err)
		}
		fd, err := protodesc.NewFile(fdProto, protoregistry.GlobalFiles)
		if err != nil {
			return nil, fmt.Errorf("build file descriptor: %w", err)
		}
		out = append(out, fd)
	}
	return out, nil
}

// LoadViaReflection resolves the given service/method by asking the remote
// server for its file descriptor set.
func LoadViaReflection(ctx context.Context, conn *grpc.ClientConn, service, method string) (protoreflect.MethodDescriptor, error) {
	fds, err := FileDescriptorsForSymbol(ctx, conn, service)
	if err != nil {
		return nil, err
	}
	for _, fd := range fds {
		for i := 0; i < fd.Services().Len(); i++ {
			srv := fd.Services().Get(i)
			if !strings.EqualFold(string(srv.FullName()), service) {
				continue
			}
			for j := 0; j < srv.Methods().Len(); j++ {
				m := srv.Methods().Get(j)
				if strings.EqualFold(string(m.Name()), method) {
					return m, nil
				}
			}
			return nil, fmt.Errorf("method %q not found in service %q", method, service)
		}
	}
	return nil, fmt.Errorf("service %q not found via reflection", service)
}

func openStream(ctx context.Context, conn *grpc.ClientConn) (reflectStream, error) {
	client := reflectionpb.NewServerReflectionClient(conn)
	return client.ServerReflectionInfo(ctx)
}
