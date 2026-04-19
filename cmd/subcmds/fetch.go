package subcmds

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pysugar/netool/cmd/base"
	cliflags "github.com/pysugar/netool/cmd/internal/cli"
	grpcrefl "github.com/pysugar/netool/cmd/internal/grpcrefl"
	"github.com/pysugar/netool/http/client"
	"github.com/pysugar/netool/http/extensions"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/dynamicpb"
)

var (
	traceIdGen uint32

	fetchCmd = &cobra.Command{
		Use:   `fetch https://www.google.com`,
		Short: "fetch http2 response from url",
		Long: `
fetch http2 response from url

fetch http2 response from url: netool fetch https://www.google.com
call grpc service: netool fetch --grpc https://localhost:8443/grpc.health.v1.Health/Check
call grpc via context path: netool fetch --grpc http://localhost:8080/grpc/grpc.health.v1.Health/Check
call grpc service: netool fetch --grpc https://localhost:8443/grpc.health.v1.Health/Check --proto-path=health.proto -d'{"service": ""}'
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				log.Printf("you must specify the url")
				return
			}

			targetURL, err := url.Parse(args[0])
			if err != nil {
				log.Printf("invalid url %s\n", args[0])
				return
			}

			isGRPC, _ := cmd.Flags().GetBool("grpc")
			if isGRPC {
				err := gRPCCall(cmd, targetURL)
				if err != nil {
					log.Fatal(err)
				}
				return
			}

			isWS, _ := cmd.Flags().GetBool("websocket")
			isGorilla, _ := cmd.Flags().GetBool("ws")
			if isWS || isGorilla {
				err := wsCall(cmd, targetURL, isGorilla)
				if err != nil {
					log.Fatal(err)
				}
				return
			}

			isVerbose := cliflags.Verbose(cmd)
			isUpgrade, _ := cmd.Flags().GetBool("upgrade")
			ctx, cancel := newContext(isVerbose, isUpgrade)
			defer cancel()

			isHTTP1, _ := cmd.Flags().GetBool("http1")
			isHTTP2, _ := cmd.Flags().GetBool("http2")
			if isHTTP1 {
				ctx = client.WithProtocol(ctx, client.HTTP1)
			} else if isHTTP2 {
				ctx = client.WithProtocol(ctx, client.HTTP2)
			}
			method, _ := cmd.Flags().GetString("method")
			headers, _ := cmd.Flags().GetStringSlice("header")

			var body io.Reader
			var contentLength int64 = -1
			if strings.EqualFold(method, http.MethodPost) || strings.EqualFold(method, http.MethodPut) || strings.EqualFold(method, http.MethodPatch) {
				data, _ := cmd.Flags().GetString("data")
				if data != "" {
					body = strings.NewReader(data)
					contentLength = int64(len(data))
				}
			}

			req, err := http.NewRequestWithContext(ctx, method, targetURL.String(), body)
			if err != nil {
				fmt.Printf("failed to create request: %v\n", err)
				return
			}
			if len(headers) > 0 {
				for _, h := range headers {
					parts := strings.Split(h, ":")
					if len(parts) != 2 {
						log.Printf("invalid header: %s\n", h)
						continue
					}
					if isVerbose {
						fmt.Printf("set header %s: %s\n", parts[0], parts[1])
					}
					req.Header.Add(parts[0], parts[1])
				}
			}
			req.ContentLength = contentLength

			res, er := client.NewFetcher().Do(ctx, req)
			if er != nil {
				fmt.Printf("Call %v %s error: %v\n", client.ProtocolFromContext(ctx), targetURL, er)
				return
			}

			if isVerbose {
				fmt.Println("\n+++++++++++++++++++++++++++")
			}
			fmt.Printf("%s %s\r\n", res.Status, res.Proto)
			for k, v := range res.Header {
				fmt.Printf("%s: %s\r\n", k, strings.Join(v, ","))
			}
			fmt.Printf("\r\n")
			resBody, _ := io.ReadAll(res.Body)
			fmt.Printf("%s", resBody)
		},
	}
)

func init() {
	fetchCmd.Flags().StringP("user-agent", "A", "", "User Agent")
	fetchCmd.Flags().StringP("method", "M", "GET", "HTTP Method")
	fetchCmd.Flags().StringSliceP("header", "H", nil, "HTTP Header")
	fetchCmd.Flags().StringP("data", "d", "{}", "request data")
	fetchCmd.Flags().BoolP("http1", "", false, "Is HTTP1 Request Or Not")
	fetchCmd.Flags().BoolP("http2", "", false, "Is HTTP2 Request Or Not")
	fetchCmd.Flags().BoolP("websocket", "W", false, "Is WebSocket Request Or Not")
	fetchCmd.Flags().BoolP("ws", "", false, "Use gorilla client for websocket")
	fetchCmd.Flags().BoolP("grpc", "G", false, "Is GRPC Request Or Not")
	fetchCmd.Flags().BoolP("upgrade", "U", false, "try http upgrade")
	fetchCmd.Flags().StringP("proto-path", "P", "", "Proto Path")
	fetchCmd.Flags().BoolP("insecure", "i", false, "Skip server certificate and domain verification (skip TLS)")
	base.AddSubCommands(fetchCmd)
}

func wsCall(cmd *cobra.Command, targetURL *url.URL, isGorilla bool) error {
	isVerbose := cliflags.Verbose(cmd)
	isInsecure, _ := cmd.Flags().GetBool("insecure")
	ctx, cancel := newContext(isVerbose, true)
	defer cancel()
	ctx = client.WithProtocol(ctx, client.WebSocket)
	if isGorilla {
		ctx = client.WithGorilla(ctx)
	}
	if isInsecure {
		ctx = client.WithInsecure(ctx)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL.String(), nil)
	if err != nil {
		fmt.Printf("failed to create request: %v\n", err)
		return err
	}
	err = client.NewFetcher().WS(ctx, req)
	if err != nil {
		fmt.Printf("failed to fetch websocket: %v\n", err)
		return err
	}
	return nil
}

func gRPCCall(cmd *cobra.Command, targetURL *url.URL) error {
	service, method, err := grpcrefl.ParseURLPath(targetURL)
	if err != nil {
		return fmt.Errorf("invalid service or method: %w", err)
	}

	requestJson, _ := cmd.Flags().GetString("data")
	protoPath, _ := cmd.Flags().GetString("proto-path")
	isVerbose := cliflags.Verbose(cmd)

	methodDesc, err := grpcrefl.LoadFromProtoFile(protoPath, service, method)
	if err != nil {
		return err
	}

	reqMessage := dynamicpb.NewMessage(methodDesc.Input())
	resMessage := dynamicpb.NewMessage(methodDesc.Output())

	if err := protojson.Unmarshal([]byte(requestJson), reqMessage); err != nil {
		return fmt.Errorf("parse request JSON: %w", err)
	}

	isUpgrade, _ := cmd.Flags().GetBool("upgrade")
	ctx, cancel := newContext(isVerbose, isUpgrade)
	defer cancel()

	ctx = client.WithProtocol(ctx, client.HTTP2)
	if err := client.NewFetcher().CallGRPC(ctx, targetURL, reqMessage, resMessage); err != nil {
		return fmt.Errorf("call grpc %s: %w", targetURL, err)
	}

	responseJson, err := protojson.Marshal(resMessage)
	if err != nil {
		return fmt.Errorf("serialize response JSON: %w", err)
	}
	fmt.Printf("%s\n", responseJson)
	return nil
}

func newContext(isVerbose, isUpgrade bool) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	if isVerbose {
		ctx = client.WithVerbose(ctx)
		traceId := atomic.AddUint32(&traceIdGen, 1)
		ctx = httptrace.WithClientTrace(ctx, extensions.NewDebugClientTrace(fmt.Sprintf("trace-req-%03d", traceId)))
	}
	if isUpgrade {
		ctx = client.WithUpgrade(ctx)
	}
	return ctx, cancel
}

