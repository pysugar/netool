package distro

import (
	"context"
	"log/slog"

	"github.com/pysugar/netool/cmd/base"
	"github.com/pysugar/netool/cmd/internal/cli"
	pb "github.com/pysugar/netool/grpc/proto"
	"github.com/pysugar/netool/grpc/server"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

type echoServer struct {
	pb.UnimplementedEchoServiceServer
}

func (s *echoServer) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	slog.Debug("echo", "message", req.Message)
	return &pb.EchoResponse{Message: req.Message}, nil
}

var echoServiceCmd = &cobra.Command{
	Use:   "echoservice [-p 8080]",
	Short: "Start a gRPC echo service",
	Long: `
Start a gRPC echo service.

Start a gRPC echo service: netool echoservice --port=8080
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port := cli.Port(cmd)
		grpcSrv := server.BuildServer("echoservice", func(s *grpc.Server) {
			pb.RegisterEchoServiceServer(s, &echoServer{})
		})
		return cli.RunServer(cmd.Context(), "echoservice",
			func(ctx context.Context) error {
				return server.Serve(ctx, port, grpcSrv)
			},
			func(ctx context.Context) error {
				grpcSrv.GracefulStop()
				return nil
			},
		)
	},
}

func init() {
	cli.AddPort(echoServiceCmd, 8080)
	base.AddSubCommands(echoServiceCmd)
}
