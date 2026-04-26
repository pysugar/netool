package subcmds

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pysugar/netool/binproto/grpc/codec"
	"github.com/pysugar/netool/cmd/base"
	"github.com/pysugar/netool/cmd/internal/cli"
	grpcrefl "github.com/pysugar/netool/cmd/internal/grpcrefl"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func parseGRPCMetadata(raw []string) metadata.MD {
	md := metadata.MD{}
	for k, vs := range cli.ParseHeaders(raw) {
		md.Append(strings.ToLower(k), vs...)
	}
	return md
}

const contextPathKey = "contextPath"

var grpcTLS *cli.TLSFlags

var grpcCmd = &cobra.Command{
	Use:   "grpc TARGET SERVICE/METHOD [flags]",
	Short: "Call a gRPC service (JSON in, JSON out)",
	Long: `
Call a gRPC service.

Send an empty request:                     netool grpc grpc.server.com:443 my.custom.server.Service/Method
Send a request with a header and a body:   netool grpc -H "Authorization: Bearer $token" -d '{"foo":"bar"}' grpc.server.com:443 my.custom.server.Service/Method
List all services exposed by a server:     netool grpc grpc.server.com:443 list
List all methods in a particular service:  netool grpc grpc.server.com:443 list my.custom.server.Service
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("usage: netool grpc TARGET SERVICE/METHOD")
		}

		cred, err := grpcCredentials(cmd)
		if err != nil {
			return err
		}

		target := args[0]
		op := args[1]

		ctx, cancel := cli.RunContext(cmd, 10*time.Second)
		defer cancel()

		contextPath, _ := cmd.Flags().GetString("context-path")
		if contextPath != "" {
			ctx = context.WithValue(ctx, contextPathKey, contextPath)
		}
		headers, _ := cmd.Flags().GetStringArray("header")
		ctx = metadata.NewOutgoingContext(ctx, parseGRPCMetadata(headers))

		opts := []grpc.DialOption{grpc.WithTransportCredentials(cred)}

		switch {
		case strings.EqualFold(op, "list") && len(args) > 2:
			return listServiceSymbols(cmd, ctx, target, args[2], opts...)
		case strings.EqualFold(op, "list"):
			return listServerServices(cmd, ctx, target, opts...)
		default:
			data, _ := cmd.Flags().GetString("data")
			return invokeByReflection(cmd, ctx, target, op, []byte(data), opts...)
		}
	},
}

func init() {
	grpcCmd.Flags().Bool("plaintext", false, "force plaintext HTTP/2 even when TLS flags are set")
	grpcCmd.Flags().StringP("data", "d", "{}", "request data")
	grpcCmd.Flags().StringP("context-path", "c", "", "context path")
	grpcCmd.Flags().StringArrayP("header", "H", []string{}, "extra header to include with the request (repeatable)")
	grpcTLS = cli.AddTLS(grpcCmd)
	base.AddSubCommands(grpcCmd)
}

// grpcCredentials picks transport credentials in this order:
//   - --plaintext     -> always plaintext, ignore TLS flags
//   - any --tls-*     -> credentials.NewTLS(<config from cli.TLSFlags>)
//   - default         -> plaintext (matches the prior default of insecure.NewCredentials)
func grpcCredentials(cmd *cobra.Command) (credentials.TransportCredentials, error) {
	plaintext, _ := cmd.Flags().GetBool("plaintext")
	if plaintext {
		return insecure.NewCredentials(), nil
	}
	cfg, err := grpcTLS.Config()
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return insecure.NewCredentials(), nil
	}
	return credentials.NewTLS(cfg), nil
}

func listServerServices(cmd *cobra.Command, ctx context.Context, target string, opts ...grpc.DialOption) error {
	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return err
	}
	defer conn.Close()

	services, err := grpcrefl.ListServices(ctx, conn)
	if err != nil {
		return err
	}
	out := cli.NewOutput(cmd)
	if out.Format() == cli.FormatJSON {
		return out.JSON(map[string]any{"services": services})
	}
	for _, s := range services {
		out.Text("%s\n", s)
	}
	return nil
}

type methodSummary struct {
	Name           string `json:"name"`
	Input          string `json:"input"`
	Output         string `json:"output"`
	StreamClient   bool   `json:"stream_client"`
	StreamServer   bool   `json:"stream_server"`
}

type serviceSummary struct {
	Service string          `json:"service"`
	Methods []methodSummary `json:"methods"`
}

func listServiceSymbols(cmd *cobra.Command, ctx context.Context, target, serviceName string, opts ...grpc.DialOption) error {
	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return err
	}
	defer conn.Close()

	fds, err := grpcrefl.FileDescriptorsForSymbol(ctx, conn, serviceName)
	if err != nil {
		return err
	}

	var services []serviceSummary
	for _, fd := range fds {
		for i := 0; i < fd.Services().Len(); i++ {
			srv := fd.Services().Get(i)
			s := serviceSummary{Service: string(srv.FullName())}
			for j := 0; j < srv.Methods().Len(); j++ {
				m := srv.Methods().Get(j)
				s.Methods = append(s.Methods, methodSummary{
					Name:         string(m.Name()),
					Input:        string(m.Input().FullName()),
					Output:       string(m.Output().FullName()),
					StreamClient: m.IsStreamingClient(),
					StreamServer: m.IsStreamingServer(),
				})
			}
			services = append(services, s)
		}
	}

	out := cli.NewOutput(cmd)
	if out.Format() == cli.FormatJSON {
		return out.JSON(map[string]any{"services": services})
	}
	for _, s := range services {
		out.Text("%s\n", s.Service)
		for _, m := range s.Methods {
			out.Text("\t%s(%s) returns (%s) stream_client=%v stream_server=%v\n",
				m.Name, m.Input, m.Output, m.StreamClient, m.StreamServer)
		}
	}
	return nil
}

func invokeByReflection(cmd *cobra.Command, ctx context.Context, target, fullMethod string, jsonData []byte, opts ...grpc.DialOption) error {
	if contextPath, ok := ctx.Value(contextPathKey).(string); ok {
		opts = append(
			opts,
			grpc.WithUnaryInterceptor(contextPathUnaryInterceptor(contextPath)),
			grpc.WithStreamInterceptor(contextPathStreamInterceptor(contextPath)),
		)
	}

	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return err
	}
	defer conn.Close()

	service, method, err := grpcrefl.ParseFullMethod(fullMethod)
	if err != nil {
		return err
	}

	methodDesc, err := grpcrefl.LoadViaReflection(ctx, conn, service, method)
	if err != nil {
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.Unimplemented {
			// Fall through to the untyped JSON frame path when reflection
			// is unavailable or the symbol is unknown.
			if !ok {
				slog.Warn("reflection unavailable, using JSON frame codec", "err", err)
			}
			methodDesc = nil
		} else {
			return err
		}
	}

	if methodDesc != nil {
		resp, er := grpcrefl.InvokeJSON(ctx, conn, methodDesc, jsonData)
		if er != nil {
			return er
		}
		out := cli.NewOutput(cmd)
		out.Text("%s\n", resp)
		return nil
	}

	return invokeJSONFrame(cmd, ctx, conn, service, method, jsonData)
}

func invokeJSONFrame(cmd *cobra.Command, ctx context.Context, conn *grpc.ClientConn, service, method string, jsonData []byte) error {
	if len(jsonData) == 0 {
		jsonData = []byte("{}")
	}
	request := &codec.JsonFrame{RawData: jsonData}
	response := &codec.JsonFrame{}
	callOpts := []grpc.CallOption{
		grpc.ForceCodec(&codec.JsonFrame{}),
		grpc.CallContentSubtype("json"),
	}
	rpc := fmt.Sprintf("/%s/%s", service, method)
	if err := conn.Invoke(ctx, rpc, request, response, callOpts...); err != nil {
		return fmt.Errorf("grpc call %s: %w", rpc, err)
	}
	cli.NewOutput(cmd).Text("%s\n", response.RawData)
	return nil
}

func contextPathStreamInterceptor(contextPath string) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		modified := prefixMethod(contextPath, method)
		slog.Debug("grpc stream method rewrite", "from", method, "to", modified)
		return streamer(ctx, desc, cc, modified, opts...)
	}
}

func contextPathUnaryInterceptor(contextPath string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		modified := prefixMethod(contextPath, method)
		slog.Debug("grpc unary method rewrite", "from", method, "to", modified)
		return invoker(ctx, modified, req, reply, cc, opts...)
	}
}

func prefixMethod(contextPath, method string) string {
	sep := "/"
	if strings.HasPrefix(method, "/") {
		sep = ""
	}
	return "/" + contextPath + sep + method
}
