# netool — network & protocol debugging toolbox

`netool` is a Cobra CLI that bundles a handful of networking utilities: an
HTTP/2 + WebSocket fetcher, a gRPC client that can talk to servers via
reflection or a local `.proto`, a transparent HTTP proxy, a file server
with no-cache middleware, a gRPC echo service, service registry/discovery
helpers for etcd, and assorted key/UUID/random helpers.

```bash
go install github.com/pysugar/netool/cmd/netool@latest
# or build from source:
git clone https://github.com/pysugar/netool && cd netool
go -C cmd build -o netool ./main
```

## Quick tour

```bash
$ netool --help
A simple CLI for Net tool

Usage:
  netool [flags]
  netool [command]

Available Commands:
  devtool     Start an HTTP debug/devtool endpoint
  discovery   Discover services from a naming registry
  echoservice Start a gRPC echo service
  etcdget     Get etcd values for a given key
  fetch       fetch http2 response from url
  fileserver  Start a file server
  grpc        Call a gRPC service (JSON in, JSON out)
  httpproxy   Start a transparent HTTP proxy
  keygen      Generate asymmetric key pairs
  rand        Generate Rand String
  read-proto  Read proto binary file
  registry    Register a service into a naming registry
  signature   Signature Commands
  uuid        Generate UUIDv4 or UUIDv5

Flags:
  -h, --help            help for netool
  -o, --output string   output format: text|json (default "text")
  -V, --verbose         verbose output (debug-level logging)
```

### fetch — HTTP/1, HTTP/2, WebSocket, gRPC-from-proto

```bash
netool fetch https://www.google.com
netool fetch --http2 https://example.com
netool fetch -W wss://echo.websocket.events/

# gRPC via local .proto:
netool echoservice --port=50051 &
netool fetch --grpc http://localhost:50051/proto.EchoService/Echo \
  --proto-path=echo.proto -d '{"message": "netool"}'
netool fetch --grpc http://localhost:50051/grpc.health.v1.Health/Check \
  --proto-path=health.proto -d '{"service": "echoservice"}'
```

### grpc — talk to a server via reflection

```bash
netool grpc grpc.server.com:443 list
netool grpc grpc.server.com:443 list my.custom.server.Service
netool grpc grpc.server.com:443 my.Service/Method -d '{"foo":"bar"}' \
  -H "Authorization: Bearer $token"
netool grpc grpc.server.com:443 my.Service/Method --insecure
```

If the remote server doesn't implement the reflection API the command
falls back to a JSON-frame codec (`application/grpc+json`).

### keygen — Curve25519 key pairs

```bash
netool keygen x25519                          # base64.RawURLEncoding (VLESS)
netool keygen x25519 -e                       # base64.StdEncoding
netool keygen x25519 -i "bW9zGdp..."          # derive from given private key
netool keygen wg                              # WireGuard (base64.StdEncoding)
netool --output json keygen x25519
```

### servers

```bash
netool fileserver --dir=. --port=8088         # static file server, no-cache
netool httpproxy --port=8080                  # CONNECT + plain HTTP forward
netool devtool --port=8080 --verbose          # request echo for debugging
netool echoservice --port=50051               # gRPC echo + health + reflection
```

All server commands shut down gracefully on SIGINT / SIGTERM.

### service discovery (etcd)

```bash
netool registry  --endpoints=127.0.0.1:2379 --env-name=live \
                 --service=myservice --address=192.168.1.5:8080
netool discovery --endpoints=127.0.0.1:2379 --env-name=live \
                 --service=myservice --watch
```

## Global flags

- `-V`, `--verbose` — switch slog to debug level, include HTTP trace for `fetch`.
- `-o`, `--output` — `text` (default) or `json` for commands that support it
  (currently `keygen`, parts of `grpc` / `etcdget`).

## Build / test

See `CLAUDE.md` for the multi-module layout, proto regeneration, and the
macOS `-ldflags="-linkmode=external"` note.
