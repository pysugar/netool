package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/pysugar/netool/grpc/interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// BuildServer assembles the standard netool gRPC server (health, reflection,
// channelz, logging interceptor, keepalive) and lets the caller register
// application services on it.
func BuildServer(serviceName string, serviceRegistry func(*grpc.Server)) *grpc.Server {
	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle:     5 * time.Minute,
		MaxConnectionAge:      2 * time.Hour,
		MaxConnectionAgeGrace: 5 * time.Minute,
		Time:                  1 * time.Hour,
		Timeout:               20 * time.Second,
	}

	s := grpc.NewServer(
		grpc.KeepaliveParams(kaParams),
		grpc.ChainUnaryInterceptor(interceptors.LoggingUnaryServerInterceptor),
	)

	healthServer := health.NewServer()
	healthServer.SetServingStatus(serviceName, grpc_health_v1.HealthCheckResponse_SERVING)

	grpc_health_v1.RegisterHealthServer(s, healthServer)
	reflection.RegisterV1(s)
	service.RegisterChannelzServiceToServer(s)
	if serviceRegistry != nil {
		serviceRegistry(s)
	}
	return s
}

// Serve binds the listener on :port and serves until ctx is cancelled, at
// which point it performs GracefulStop. Blocks until the server exits.
func Serve(ctx context.Context, port int, server *grpc.Server) error {
	logger := grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr)
	grpclog.SetLoggerV2(logger)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("listen :%d: %w", port, err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- server.Serve(lis) }()

	logger.Infof("gRPC server listening on :%d", port)

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		server.GracefulStop()
		return <-errCh
	}
}

// StartGrpcServer is the blocking convenience wrapper retained for callers
// that don't need a context handle. It composes BuildServer + Serve with a
// background context.
func StartGrpcServer(port int, serviceName string, serviceRegistry func(*grpc.Server)) error {
	return Serve(context.Background(), port, BuildServer(serviceName, serviceRegistry))
}
