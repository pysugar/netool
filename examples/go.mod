module github.com/pysugar/netool/examples

go 1.24.0

toolchain go1.24.13

replace github.com/pysugar/netool => ../

require (
	github.com/pysugar/netool v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.10
)

require (
	github.com/gorilla/websocket v1.5.3
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/jhump/protoreflect v1.17.0
	github.com/pires/go-proxyproto v0.8.0
	github.com/prometheus/client_golang v1.20.5
	golang.org/x/net v0.48.0
	google.golang.org/grpc v1.79.3
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bufbuild/protocompile v0.14.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
)
