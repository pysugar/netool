package subcmds

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pysugar/netool/cmd/base"
	"github.com/pysugar/netool/cmd/internal/cli"
	grpcrefl "github.com/pysugar/netool/cmd/internal/grpcrefl"
	"github.com/pysugar/netool/http/client"
	"github.com/pysugar/netool/http/extensions"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/dynamicpb"
)

var (
	traceIDGen uint32

	fetchTLS *cli.TLSFlags

	fetchCmd = &cobra.Command{
		Use:   `fetch URL`,
		Short: "Fetch HTTP/1, HTTP/2, WebSocket, or gRPC responses",
		Long: `
Fetch a URL using HTTP/1, HTTP/2, WebSocket, or gRPC.

  netool fetch https://www.google.com
  netool fetch --grpc https://localhost:8443/grpc.health.v1.Health/Check
  netool fetch --grpc http://localhost:8080/grpc/grpc.health.v1.Health/Check
  netool fetch --grpc https://localhost:8443/grpc.health.v1.Health/Check \
                --proto-path=health.proto -d '{"service":""}'
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetURL, err := url.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid url %q: %w", args[0], err)
			}

			ctx, cancel := cli.RunContext(cmd, 60*time.Second)
			defer cancel()

			isVerbose := cli.Verbose(cmd)
			isUpgrade, _ := cmd.Flags().GetBool("upgrade")
			ctx = decorate(ctx, isVerbose, isUpgrade)

			isGRPC, _ := cmd.Flags().GetBool("grpc")
			if isGRPC {
				return gRPCCall(cmd, ctx, targetURL)
			}

			isWS, _ := cmd.Flags().GetBool("websocket")
			isGorilla, _ := cmd.Flags().GetBool("ws")
			if isWS || isGorilla {
				return wsCall(cmd, ctx, targetURL, isGorilla)
			}

			return httpCall(cmd, ctx, targetURL)
		},
	}
)

func init() {
	fetchCmd.Flags().StringP("user-agent", "A", "", "User-Agent header value")
	fetchCmd.Flags().StringP("method", "M", "GET", "HTTP method")
	fetchCmd.Flags().StringSliceP("header", "H", nil, "extra header (repeatable, K:V)")
	fetchCmd.Flags().StringP("data", "d", "{}", "request body (raw text or JSON)")
	fetchCmd.Flags().Bool("http1", false, "force HTTP/1.1")
	fetchCmd.Flags().Bool("http2", false, "force HTTP/2 (h2/h2c)")
	fetchCmd.Flags().BoolP("websocket", "W", false, "use WebSocket transport")
	fetchCmd.Flags().Bool("ws", false, "use the gorilla WebSocket client")
	fetchCmd.Flags().BoolP("grpc", "G", false, "send a gRPC request")
	fetchCmd.Flags().BoolP("upgrade", "U", false, "negotiate HTTP upgrade")
	fetchCmd.Flags().StringP("proto-path", "P", "", "path to .proto file (gRPC mode)")
	cli.AddTimeout(fetchCmd, 60*time.Second)
	fetchTLS = cli.AddTLS(fetchCmd)
	base.AddSubCommands(fetchCmd)
}

func httpCall(cmd *cobra.Command, ctx context.Context, targetURL *url.URL) error {
	isHTTP1, _ := cmd.Flags().GetBool("http1")
	isHTTP2, _ := cmd.Flags().GetBool("http2")
	switch {
	case isHTTP1:
		ctx = client.WithProtocol(ctx, client.HTTP1)
	case isHTTP2:
		ctx = client.WithProtocol(ctx, client.HTTP2)
	}

	method, _ := cmd.Flags().GetString("method")
	headers, _ := cmd.Flags().GetStringSlice("header")
	userAgent, _ := cmd.Flags().GetString("user-agent")

	var body io.Reader
	contentLength := int64(-1)
	if methodTakesBody(method) {
		data, _ := cmd.Flags().GetString("data")
		if data != "" {
			body = strings.NewReader(data)
			contentLength = int64(len(data))
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL.String(), body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	for k, v := range parseHeaders(headers) {
		req.Header[k] = v
	}
	req.ContentLength = contentLength

	res, err := client.NewFetcher().Do(ctx, req)
	if err != nil {
		return fmt.Errorf("call %v %s: %w", client.ProtocolFromContext(ctx), targetURL, err)
	}
	defer res.Body.Close()

	out := cli.NewOutput(cmd)
	w := out.Writer()
	fmt.Fprintf(w, "%s %s\r\n", res.Status, res.Proto)
	for k, v := range res.Header {
		fmt.Fprintf(w, "%s: %s\r\n", k, strings.Join(v, ","))
	}
	fmt.Fprintf(w, "\r\n")
	if _, err := io.Copy(w, res.Body); err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	return nil
}

func wsCall(cmd *cobra.Command, ctx context.Context, targetURL *url.URL, isGorilla bool) error {
	ctx = client.WithProtocol(ctx, client.WebSocket)
	if isGorilla {
		ctx = client.WithGorilla(ctx)
	}
	if fetchTLS != nil && fetchTLS.Insecure {
		ctx = client.WithInsecure(ctx)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL.String(), nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if err := client.NewFetcher().WS(ctx, req); err != nil {
		return fmt.Errorf("websocket %s: %w", targetURL, err)
	}
	return nil
}

func gRPCCall(cmd *cobra.Command, ctx context.Context, targetURL *url.URL) error {
	service, method, err := grpcrefl.ParseURLPath(targetURL)
	if err != nil {
		return fmt.Errorf("invalid service or method: %w", err)
	}

	requestJSON, _ := cmd.Flags().GetString("data")
	protoPath, _ := cmd.Flags().GetString("proto-path")

	methodDesc, err := grpcrefl.LoadFromProtoFile(protoPath, service, method)
	if err != nil {
		return err
	}

	reqMessage := dynamicpb.NewMessage(methodDesc.Input())
	resMessage := dynamicpb.NewMessage(methodDesc.Output())
	if err := protojson.Unmarshal([]byte(requestJSON), reqMessage); err != nil {
		return fmt.Errorf("parse request JSON: %w", err)
	}

	ctx = client.WithProtocol(ctx, client.HTTP2)
	if err := client.NewFetcher().CallGRPC(ctx, targetURL, reqMessage, resMessage); err != nil {
		return fmt.Errorf("call grpc %s: %w", targetURL, err)
	}

	responseJSON, err := protojson.Marshal(resMessage)
	if err != nil {
		return fmt.Errorf("serialize response JSON: %w", err)
	}
	out := cli.NewOutput(cmd)
	out.Text("%s\n", responseJSON)
	return nil
}

func decorate(ctx context.Context, verbose, upgrade bool) context.Context {
	if verbose {
		ctx = client.WithVerbose(ctx)
		traceID := atomic.AddUint32(&traceIDGen, 1)
		ctx = httptrace.WithClientTrace(ctx, extensions.NewDebugClientTrace(fmt.Sprintf("trace-req-%03d", traceID)))
	}
	if upgrade {
		ctx = client.WithUpgrade(ctx)
	}
	return ctx
}

func methodTakesBody(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	}
	return false
}

func parseHeaders(raw []string) http.Header {
	h := http.Header{}
	for _, item := range raw {
		k, v, ok := strings.Cut(item, ":")
		if !ok {
			slog.Warn("invalid header, expected K:V", "header", item)
			continue
		}
		h.Add(strings.TrimSpace(k), strings.TrimSpace(v))
	}
	return h
}
