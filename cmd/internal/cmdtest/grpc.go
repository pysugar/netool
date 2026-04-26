package cmdtest

import (
	"net"
	"testing"
	"time"

	"github.com/pysugar/netool/grpc/server"
	"google.golang.org/grpc"
)

// StartGRPC builds a netool standard gRPC server (health + reflection +
// channelz + interceptors) on a random port, registers caller-supplied
// services, and starts serving. The bound "host:port" string is returned;
// t.Cleanup performs GracefulStop.
//
// serviceName is what BuildServer reports as healthy via grpc_health_v1.
func StartGRPC(t *testing.T, serviceName string, register func(*grpc.Server)) string {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	s := server.BuildServer(serviceName, register)

	done := make(chan error, 1)
	go func() { done <- s.Serve(lis) }()

	t.Cleanup(func() {
		s.GracefulStop()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Logf("grpc server did not stop in time")
		}
	})

	return lis.Addr().String()
}
