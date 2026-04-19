// Package reflection centralizes gRPC method descriptor resolution used by
// both the netool grpc and fetch commands.
//
// Three descriptor sources are supported:
//
//   - ParseFullMethod: split "service/method" or URL paths into parts.
//   - LoadFromProtoFile: parse a local .proto file via jhump/protoreflect.
//   - LoadViaReflection: talk to a running server's reflection service.
//
// Each returns a protoreflect.MethodDescriptor that can be combined with the
// Invoke helper to dispatch a JSON request through grpc.ClientConn.
package grpcrefl

import (
	"fmt"
	"net/url"
	"strings"
)

// ParseFullMethod splits "service/method" (the gRPC shorthand) into its
// two components. Leading/trailing slashes are tolerated.
func ParseFullMethod(s string) (service, method string, err error) {
	s = strings.Trim(s, "/")
	parts := strings.Split(s, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid method format %q: want service/method", s)
	}
	return parts[0], parts[1], nil
}

// ParseURLPath extracts service and method from the last two segments of
// the URL path — the shape used by `netool fetch --grpc`.
func ParseURLPath(u *url.URL) (service, method string, err error) {
	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) < 2 {
		return "", "", fmt.Errorf("path %q missing service or method", u.Path)
	}
	return segments[len(segments)-2], segments[len(segments)-1], nil
}
