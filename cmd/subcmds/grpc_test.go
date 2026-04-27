package subcmds

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/pysugar/netool/cmd/internal/cli"
	"github.com/pysugar/netool/cmd/internal/cmdtest"
	pb "github.com/pysugar/netool/grpc/proto"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

type echoEchoServer struct {
	pb.UnimplementedEchoServiceServer
}

func (s *echoEchoServer) Echo(_ context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	return &pb.EchoResponse{Message: req.Message}, nil
}

func newGRPCRoot(t *testing.T) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "netool"}
	root.PersistentFlags().BoolP(cli.FlagVerbose, "V", false, "")
	root.PersistentFlags().StringP(cli.FlagOutput, "o", string(cli.FormatText), "")
	root.AddCommand(grpcCmd)
	return root
}

func startEchoServer(t *testing.T) string {
	t.Helper()
	return cmdtest.StartGRPC(t, "echoservice", func(s *grpc.Server) {
		pb.RegisterEchoServiceServer(s, &echoEchoServer{})
	})
}

func TestGRPCListServicesText(t *testing.T) {
	addr := startEchoServer(t)
	res := cmdtest.Run(t, newGRPCRoot(t), "grpc", addr, "list")
	if res.Err != nil {
		t.Fatalf("run: %v (stderr=%q)", res.Err, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "proto.EchoService") {
		t.Fatalf("expected proto.EchoService in list output:\n%s", res.Stdout)
	}
	// BuildServer also registers health + channelz + reflection.
	for _, want := range []string{
		"grpc.health.v1.Health",
		"grpc.reflection.v1.ServerReflection",
	} {
		if !strings.Contains(res.Stdout, want) {
			t.Fatalf("missing %q in list output:\n%s", want, res.Stdout)
		}
	}
}

func TestGRPCListServicesJSON(t *testing.T) {
	addr := startEchoServer(t)
	res := cmdtest.Run(t, newGRPCRoot(t), "--output", "json", "grpc", addr, "list")
	if res.Err != nil {
		t.Fatalf("run: %v", res.Err)
	}
	var got map[string][]string
	if err := json.Unmarshal([]byte(res.Stdout), &got); err != nil {
		t.Fatalf("not valid JSON: %v\noutput:%s", err, res.Stdout)
	}
	found := false
	for _, s := range got["services"] {
		if s == "proto.EchoService" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("proto.EchoService missing from JSON: %+v", got)
	}
}

func TestGRPCListServiceMethodsText(t *testing.T) {
	addr := startEchoServer(t)
	res := cmdtest.Run(t, newGRPCRoot(t), "grpc", addr, "list", "proto.EchoService")
	if res.Err != nil {
		t.Fatalf("run: %v (stderr=%q)", res.Err, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "proto.EchoService") {
		t.Fatalf("missing service name:\n%s", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "Echo(proto.EchoRequest) returns (proto.EchoResponse)") {
		t.Fatalf("missing Echo method line:\n%s", res.Stdout)
	}
}

func TestGRPCListServiceMethodsJSON(t *testing.T) {
	addr := startEchoServer(t)
	res := cmdtest.Run(t, newGRPCRoot(t),
		"--output", "json", "grpc", addr, "list", "proto.EchoService")
	if res.Err != nil {
		t.Fatalf("run: %v", res.Err)
	}
	var got struct {
		Services []serviceSummary `json:"services"`
	}
	if err := json.Unmarshal([]byte(res.Stdout), &got); err != nil {
		t.Fatalf("not valid JSON: %v\noutput:%s", err, res.Stdout)
	}
	var echoSvc *serviceSummary
	for i := range got.Services {
		if got.Services[i].Service == "proto.EchoService" {
			echoSvc = &got.Services[i]
			break
		}
	}
	if echoSvc == nil {
		t.Fatalf("proto.EchoService not in JSON: %+v", got)
	}
	if len(echoSvc.Methods) == 0 || echoSvc.Methods[0].Name != "Echo" {
		t.Fatalf("expected Echo method, got %+v", echoSvc.Methods)
	}
}

func TestGRPCInvokeEchoViaReflection(t *testing.T) {
	addr := startEchoServer(t)
	res := cmdtest.Run(t, newGRPCRoot(t),
		"grpc", "-d", `{"message":"hello"}`,
		addr, "proto.EchoService/Echo")
	if res.Err != nil {
		t.Fatalf("run: %v (stderr=%q)", res.Err, res.Stderr)
	}
	// Output should be the JSON-encoded EchoResponse.
	var got map[string]string
	body := strings.TrimSpace(res.Stdout)
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("response not JSON: %v\nstdout=%q", err, res.Stdout)
	}
	if got["message"] != "hello" {
		t.Fatalf("echo round-trip mismatch: %+v", got)
	}
}

// The response from a successful gRPC call is already JSON, so --output json
// must not drop it the way Output.Text would.
func TestGRPCInvokeEchoPreservesJSONOutput(t *testing.T) {
	addr := startEchoServer(t)
	res := cmdtest.Run(t, newGRPCRoot(t),
		"--output", "json", "grpc", "-d", `{"message":"world"}`,
		addr, "proto.EchoService/Echo")
	if res.Err != nil {
		t.Fatalf("run: %v (stderr=%q)", res.Err, res.Stderr)
	}
	body := strings.TrimSpace(res.Stdout)
	if body == "" {
		t.Fatal("--output json produced empty stdout")
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("response not JSON: %v\nstdout=%q", err, res.Stdout)
	}
	if got["message"] != "world" {
		t.Fatalf("echo round-trip mismatch: %+v", got)
	}
}

func TestGRPCMissingArgs(t *testing.T) {
	res := cmdtest.Run(t, newGRPCRoot(t), "grpc", "127.0.0.1:0")
	if res.Err == nil {
		t.Fatalf("expected error for missing SERVICE/METHOD argument")
	}
}
