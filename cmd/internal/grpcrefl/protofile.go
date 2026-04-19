package grpcrefl

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jhump/protoreflect/desc/protoparse"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// LoadFromProtoFile parses a local .proto file and resolves the given
// service + method into a MethodDescriptor. Returns the discovered service
// descriptor alongside so callers can list alternatives on failure.
func LoadFromProtoFile(protoPath, service, method string) (protoreflect.MethodDescriptor, error) {
	parser := protoparse.Parser{
		ImportPaths: []string{filepath.Dir(protoPath)},
	}
	fds, err := parser.ParseFiles(filepath.Base(protoPath))
	if err != nil {
		return nil, fmt.Errorf("parse proto %s: %w", protoPath, err)
	}
	if len(fds) == 0 {
		return nil, fmt.Errorf("no file descriptors parsed from %s", protoPath)
	}

	fd := fds[0]
	srv := fd.FindService(service)
	if srv == nil {
		return nil, fmt.Errorf("service %q not found in %s (available: %s)",
			service, protoPath, availableServices(fd.UnwrapFile()))
	}
	m := srv.FindMethodByName(method)
	if m == nil {
		return nil, fmt.Errorf("method %q not found in service %q", method, service)
	}
	return m.UnwrapMethod(), nil
}

func availableServices(fd protoreflect.FileDescriptor) string {
	var names []string
	for i := 0; i < fd.Services().Len(); i++ {
		names = append(names, string(fd.Services().Get(i).FullName()))
	}
	return strings.Join(names, ", ")
}
