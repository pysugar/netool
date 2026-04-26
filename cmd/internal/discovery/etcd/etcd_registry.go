package etcd

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	ttl           int64 = 10
	retryInterval       = 5 * time.Second
)

type (
	etcdRegistry struct {
		client      *clientv3.Client
		ctx         context.Context
		stop        context.CancelFunc
		instance    *Instance
		lease       clientv3.LeaseID
		keepaliveCh <-chan *clientv3.LeaseKeepAliveResponse
	}
)

// RegisterETCD registers the service into etcd, then blocks until ctx is
// cancelled (typically by SIGINT/SIGTERM propagated from cobra). On exit it
// deregisters before returning.
func RegisterETCD(ctx context.Context, endpoints []string, envName, serviceName, address string) error {
	client, err := newEtcdClient(endpoints)
	if err != nil {
		return fmt.Errorf("etcd client: %w", err)
	}
	defer client.Close()

	registrar := NewEtcdRegistry(client)
	if err := registrar.Register(ctx, &Instance{
		ServiceName: serviceName,
		Env:         envName,
		Endpoint:    Endpoint{Address: address, Group: DefaultGroup},
	}); err != nil {
		return fmt.Errorf("register: %w", err)
	}

	<-ctx.Done()
	slog.Info("registrar shutting down", "service", serviceName, "cause", context.Cause(ctx))

	deregisterCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := registrar.Deregister(deregisterCtx); err != nil {
		slog.Warn("deregister failed", "service", serviceName, "err", err)
	}
	return nil
}

func NewEtcdRegistry(cli *clientv3.Client) Registrar {
	ctx, cancel := context.WithCancel(context.Background())
	return &etcdRegistry{
		client: cli,
		ctx:    ctx,
		stop:   cancel,
	}
}

func (r *etcdRegistry) Register(appCtx context.Context, instance *Instance) error {
	ctx, cancel := context.WithTimeout(appCtx, 3*time.Second)
	defer cancel()

	lgr, err := r.client.Grant(ctx, ttl)
	if err != nil {
		return fmt.Errorf("grant lease: %w", err)
	}

	instanceKey := instance.Key()
	value := instance.Endpoint.Encode()

	pr, err := r.client.Put(ctx, instanceKey, value, clientv3.WithLease(lgr.ID))
	if err != nil {
		return fmt.Errorf("put %s: %w", instanceKey, err)
	}

	r.keepaliveCh, err = r.client.KeepAlive(context.Background(), lgr.ID)
	if err != nil {
		return fmt.Errorf("keepalive: %w", err)
	}
	r.lease = lgr.ID
	r.instance = instance

	slog.Info("etcd register success",
		"key", instanceKey,
		"value", value,
		"put_revision", pr.Header.GetRevision(),
		"lease", lgr.ID)

	go r.keepalive()
	return nil
}

func (r *etcdRegistry) Deregister(ctx context.Context) error {
	r.stop()

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	lrr, err := r.client.Revoke(ctx, r.lease)
	if err != nil {
		slog.Warn("revoke lease failed", "lease", r.lease, "err", err)
		return err
	}
	slog.Info("deregistered", "lease", r.lease, "revision", lrr.Header.GetRevision())
	return nil
}

func (r *etcdRegistry) keepalive() {
	for {
		select {
		case resp, ok := <-r.keepaliveCh:
			if !ok {
				slog.Warn("etcd keepalive channel closed, retrying registration")
				go r.retry()
				return
			}
			if resp == nil {
				slog.Warn("etcd keepalive response nil, retrying registration")
				go r.retry()
				return
			}
			slog.Debug("keepalive ok", "lease", resp.ID, "ttl", resp.TTL)
		case <-r.ctx.Done():
			slog.Debug("keepalive context done")
			return
		}
	}
}

func (r *etcdRegistry) retry() {
	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := r.Register(context.Background(), r.instance); err == nil {
				slog.Info("etcd register retry success")
				return
			} else {
				slog.Warn("etcd register retry failed", "err", err)
			}
		case <-r.ctx.Done():
			slog.Debug("retry context done")
			return
		}
	}
}
