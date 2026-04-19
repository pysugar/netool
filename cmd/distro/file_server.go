package distro

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"

	"github.com/pysugar/netool/cmd/base"
	"github.com/pysugar/netool/cmd/internal/cli"
	"github.com/pysugar/netool/http/extensions"
	"github.com/pysugar/netool/net/ipaddr"
	"github.com/spf13/cobra"
)

var fileServerCmd = &cobra.Command{
	Use:   "fileserver [-d .] [-p 8080]",
	Short: "Start a file server",
	Long: `
Start a file server.

Start file server: netool fileserver --dir=. --port=8088
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sharedDirectory, _ := cmd.Flags().GetString("dir")
		port, _ := cmd.Flags().GetInt("port")
		return runFileServer(cmd.Context(), sharedDirectory, port, cli.Verbose(cmd))
	},
}

func init() {
	fileServerCmd.Flags().IntP("port", "p", 8080, "file server port")
	fileServerCmd.Flags().StringP("dir", "d", ".", "file server directory")
	base.AddSubCommands(fileServerCmd)
}

func runFileServer(ctx context.Context, dir string, port int, verbose bool) error {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", dir, err)
	}

	mux := http.NewServeMux()
	handler := http.StripPrefix("/", extensions.NoCacheMiddleware(http.FileServer(http.Dir(absPath))))
	mux.Handle("/", handler)

	addrs, err := ipaddr.GetLocalIPv4Addrs(verbose)
	if err != nil {
		slog.Warn("failed to enumerate local IPv4 addresses", "err", err)
	}
	if len(addrs) == 0 {
		addrs = []string{"0.0.0.0"}
	}
	addr := fmt.Sprintf(":%d", port)

	srv := &http.Server{Addr: addr, Handler: mux}

	return cli.RunServer(ctx, "fileserver",
		func(ctx context.Context) error {
			ln, lerr := net.Listen("tcp", addr)
			if lerr != nil {
				return lerr
			}
			slog.Info("file server listening",
				"directory", absPath,
				"address", fmt.Sprintf("http://%s:%d", addrs[0], port))
			return srv.Serve(ln)
		},
		srv.Shutdown,
	)
}
