# netool CLI

```bash
$ go build -o netool ./main

$ ./netool --help
$ ./netool keygen x25519                       # random key pair
$ ./netool keygen x25519 -i "bW9zGdp..."       # derive from given private key
$ ./netool keygen wg                           # WireGuard flavour
$ ./netool fetch https://www.google.com
$ ./netool grpc grpc.server.com:443 my.Service/Method -d '{"foo":"bar"}'
$ ./netool fileserver --dir=. --port=8088
$ ./netool echoservice --port=50051
```

## Layout

- `base/`   — root cobra command, wires persistent `-V / --verbose` and `-o / --output`.
- `main/`   — entry point. Blank-imports `distro/` and `subcmds/` for their `init()`s.
- `distro/` — server-style commands (fileserver, httpproxy, registry, discovery, echoservice, devtool).
- `subcmds/` — one-shot utilities (fetch, grpc, keygen, uuid, etcdget, rand, signature, read-proto).
- `internal/cli/`            — shared flag/context/output/serve helpers.
- `internal/cmdtest/`        — cobra test harness.
- `internal/grpcrefl/`       — gRPC descriptor resolution + JSON invocation.
- `internal/discovery/etcd/` — etcd service registry/discovery used by registry/discovery commands.

## Conventions

New subcommands should:

- register themselves from their own `init()` via `base.AddSubCommands(cmd)` (parent→child within a package uses `cli.Register(parent, children...)`).
- read `--verbose` via `cli.Verbose(cmd)`, not a per-command flag.
- wrap timeouts / cancellation with `cli.RunContext(cmd, defaultTimeout)`.
- wrap long-running servers with `cli.RunServer(ctx, name, start, stop)`.
- emit JSON when `cli.NewOutput(cmd).Format() == cli.FormatJSON`.
- reserve `-p` for `--port`; avoid single-letter shorts when the flag has a `--long` that works fine.
