package etcd

import "context"

type (
	Registrar interface {
		Register(ctx context.Context, instance *Instance) error
		Deregister(ctx context.Context) error
	}

	RegisterNamingService func(ctx context.Context, endpoints []string, env, service, address string) error
)
