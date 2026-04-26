package grpcrefl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const echoProto = `
syntax = "proto3";
package demo;

service EchoService {
  rpc Echo (EchoRequest) returns (EchoResponse);
}

message EchoRequest  { string message = 1; }
message EchoResponse { string message = 1; }
`

func writeProto(t *testing.T, name, contents string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(contents), 0644); err != nil {
		t.Fatalf("write proto: %v", err)
	}
	return p
}

func TestLoadFromProtoFile_Happy(t *testing.T) {
	path := writeProto(t, "echo.proto", echoProto)

	md, err := LoadFromProtoFile(path, "demo.EchoService", "Echo")
	if err != nil {
		t.Fatalf("LoadFromProtoFile: %v", err)
	}
	if got := string(md.Name()); got != "Echo" {
		t.Fatalf("method name = %q, want Echo", got)
	}
	if got := string(md.Input().FullName()); got != "demo.EchoRequest" {
		t.Fatalf("input = %q, want demo.EchoRequest", got)
	}
	if got := string(md.Output().FullName()); got != "demo.EchoResponse" {
		t.Fatalf("output = %q, want demo.EchoResponse", got)
	}
}

func TestLoadFromProtoFile_ServiceNotFound(t *testing.T) {
	path := writeProto(t, "echo.proto", echoProto)

	_, err := LoadFromProtoFile(path, "demo.WrongService", "Echo")
	if err == nil {
		t.Fatal("expected error for unknown service")
	}
	// Error should list the available service so users can debug.
	if !strings.Contains(err.Error(), "demo.EchoService") {
		t.Fatalf("error %q does not mention available services", err)
	}
}

func TestLoadFromProtoFile_MethodNotFound(t *testing.T) {
	path := writeProto(t, "echo.proto", echoProto)

	_, err := LoadFromProtoFile(path, "demo.EchoService", "Ping")
	if err == nil {
		t.Fatal("expected error for unknown method")
	}
	if !strings.Contains(err.Error(), "Ping") {
		t.Fatalf("error %q should mention the missing method", err)
	}
}

func TestLoadFromProtoFile_BadPath(t *testing.T) {
	_, err := LoadFromProtoFile("/nonexistent/path/missing.proto", "demo.EchoService", "Echo")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
