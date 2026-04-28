module github.com/pysugar/netool/cmd

go 1.22.0

toolchain go1.22.12

replace (
	github.com/pysugar/netool => ../
	github.com/pysugar/netool/examples => ../examples
)

require (
	github.com/golang/protobuf v1.5.4
	github.com/jhump/protoreflect v1.17.0
	github.com/pysugar/netool v0.0.0-00010101000000-000000000000
	github.com/pysugar/netool/examples v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5
	go.etcd.io/etcd/api/v3 v3.5.13
	go.etcd.io/etcd/client/v3 v3.5.13
	google.golang.org/grpc v1.71.3
	google.golang.org/protobuf v1.36.4
)

require (
	github.com/bufbuild/protocompile v0.14.1 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.13 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.17.0 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250106144421-5f5ef82da422 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da9e5054f // indirect
)
