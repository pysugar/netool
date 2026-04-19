package distro

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/pysugar/netool/cmd/base"
	"github.com/pysugar/netool/cmd/internal/cli"
	"github.com/spf13/cobra"
)

var httpProxyCmd = &cobra.Command{
	Use:   "httpproxy [-p 8080]",
	Short: "Start a transparent HTTP proxy",
	Long: `
Start a transparent HTTP proxy (CONNECT tunnel + plain HTTP forward).

Start a Transparent HTTP Proxy: netool httpproxy --port=8080
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		return runHTTPProxy(cmd.Context(), port)
	},
}

func init() {
	httpProxyCmd.Flags().IntP("port", "p", 8080, "http proxy port")
	base.AddSubCommands(httpProxyCmd)
}

func runHTTPProxy(ctx context.Context, port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("listen :%d: %w", port, err)
	}

	return cli.RunServer(ctx, "httpproxy",
		func(ctx context.Context) error {
			slog.Info("http proxy listening", "addr", lis.Addr().String())
			for {
				clientConn, er := lis.Accept()
				if er != nil {
					if errors.Is(er, net.ErrClosed) {
						return nil
					}
					slog.Warn("accept failed", "err", er)
					continue
				}
				go handleHTTPProxy(clientConn)
			}
		},
		func(_ context.Context) error { return lis.Close() },
	)
}

func handleHTTPProxy(clientConn net.Conn) {
	defer clientConn.Close()

	reader := bufio.NewReader(clientConn)
	request, err := http.ReadRequest(reader)
	if err != nil {
		slog.Debug("read client request failed", "err", err)
		return
	}

	if request.Method == http.MethodConnect {
		handleConnectMethod(clientConn, request)
	} else {
		handleHTTPRequest(clientConn, request)
	}
}

func handleConnectMethod(clientConn net.Conn, request *http.Request) {
	targetHost := request.Host
	if !strings.Contains(targetHost, ":") {
		if request.URL.Scheme == "https" {
			targetHost = fmt.Sprintf("%s:443", targetHost)
		} else {
			targetHost = fmt.Sprintf("%s:80", targetHost)
		}
	}

	targetConn, err := net.DialTimeout("tcp", targetHost, 10*time.Second)
	if err != nil {
		slog.Warn("connect target failed", "target", targetHost, "err", err)
		const errorHeaders = "\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n"
		fmt.Fprintf(clientConn, "HTTP/1.1 503 Service Unavailable"+errorHeaders+err.Error())
		return
	}
	fmt.Fprintf(clientConn, "HTTP/1.1 200 Connection Established\r\n\r\n")
	defer targetConn.Close()

	slog.Debug("tunnel",
		"client", clientConn.RemoteAddr(),
		"local", clientConn.LocalAddr(),
		"target", targetConn.RemoteAddr())

	go func() {
		if _, er := io.Copy(targetConn, clientConn); er != nil {
			slog.Debug("copy client->target", "err", er)
		}
	}()

	if _, err := io.Copy(clientConn, targetConn); err != nil {
		slog.Debug("copy target->client", "err", err)
	}
}

func handleHTTPRequest(clientConn net.Conn, request *http.Request) {
	targetHost := request.Host
	if !strings.Contains(targetHost, ":") {
		if request.URL.Scheme == "https" {
			targetHost = fmt.Sprintf("%s:443", targetHost)
		} else {
			targetHost = fmt.Sprintf("%s:80", targetHost)
		}
	}

	targetConn, err := net.DialTimeout("tcp", targetHost, 10*time.Second)
	if err != nil {
		slog.Warn("connect target failed", "target", targetHost, "err", err)
		const errorHeaders = "\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n"
		fmt.Fprintf(clientConn, "HTTP/1.1 "+"503 Service Unavailable"+errorHeaders+err.Error())
		return
	}
	defer targetConn.Close()

	remoteAddr := clientConn.RemoteAddr()
	slog.Debug("proxy forward", "remote_addr", remoteAddr)

	clientIP, _, err := net.SplitHostPort(remoteAddr.String())
	if err != nil {
		slog.Debug("parse client ip", "err", err)
		clientIP = "Unknown"
	}

	if prior, ok := request.Header["X-Forwarded-For"]; ok {
		clientIP = strings.Join(prior, ", ") + ", " + clientIP
	}
	request.Header.Set("X-Forwarded-For", clientIP)

	if err := request.Write(targetConn); err != nil {
		slog.Warn("forward request failed", "err", err)
		const errorHeaders = "\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n"
		fmt.Fprintf(clientConn, "HTTP/1.1 "+"503 Service Unavailable"+errorHeaders+err.Error())
		return
	}

	go func() {
		if _, er := io.Copy(targetConn, clientConn); er != nil {
			slog.Debug("copy client->target", "err", er)
		}
	}()

	if _, err := io.Copy(clientConn, targetConn); err != nil {
		slog.Debug("copy target->client", "err", err)
	}
}
