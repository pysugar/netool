package distro

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/pysugar/netool/cmd/base"
	"github.com/pysugar/netool/cmd/internal/cli"
	"github.com/pysugar/netool/http/extensions"
	"github.com/spf13/cobra"
)

var devtoolCmd = &cobra.Command{
	Use:   "devtool [-p 8080]",
	Short: "Start an HTTP debug/devtool endpoint",
	Long: `
Start an HTTP debug endpoint that echoes request detail.

Start a DevTool for HTTP: netool devtool --port=8080 --verbose
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		return runDevtoolServer(cmd.Context(), port, cli.Verbose(cmd))
	},
}

func init() {
	devtoolCmd.Flags().IntP("port", "p", 8080, "devtool server port")
	base.AddSubCommands(devtoolCmd)
}

func runDevtoolServer(ctx context.Context, port int, verbose bool) error {
	debugHandler := extensions.CORSMiddleware(http.HandlerFunc(extensions.DebugHandler))
	debugHandlerJSON := extensions.CORSMiddleware(http.HandlerFunc(extensions.DebugHandlerJSON))
	if verbose {
		debugHandler = extensions.LoggingMiddleware(debugHandler)
		debugHandlerJSON = extensions.LoggingMiddleware(debugHandlerJSON)
	}

	mux := http.NewServeMux()
	mux.Handle("/", extensions.CORSMiddleware(debugHandler))
	mux.Handle("/json", extensions.CORSMiddleware(debugHandlerJSON))

	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{Addr: addr, Handler: mux}

	return cli.RunServer(ctx, "devtool",
		func(ctx context.Context) error {
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return err
			}
			slog.Info("devtool listening", "address", fmt.Sprintf("http://localhost%s", addr))
			return srv.Serve(ln)
		},
		srv.Shutdown,
	)
}
