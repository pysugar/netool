package etcd

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type (
	etcdDiscoverer struct {
		cli *clientv3.Client
		rev int64
	}
)

// DiscoverETCD fetches the current set of endpoints. When watchEnabled is
// true it also subscribes to changes and blocks until ctx is cancelled,
// logging incremental updates via slog.
func DiscoverETCD(ctx context.Context, etcdEndpoints []string, envName, serviceName, group string, watchEnabled bool) ([]*Endpoint, error) {
	client, err := newEtcdClient(etcdEndpoints)
	if err != nil {
		return nil, fmt.Errorf("etcd client: %w", err)
	}

	serviceWithEnv := fmt.Sprintf("/%s/%s", envName, serviceName)
	serviceDiscoverKey := fmt.Sprintf("%s/", serviceWithEnv)
	if group != DefaultGroup && group != "" {
		serviceDiscoverKey = fmt.Sprintf("%s:%s/", serviceWithEnv, group)
	}

	getCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	discoverer := NewEtcdDiscoverer(client)
	if !watchEnabled {
		endpoints, err := discoverer.Get(getCtx, serviceDiscoverKey)
		if err != nil {
			return nil, fmt.Errorf("discover get: %w", err)
		}
		return endpoints, nil
	}

	endpoints, watcher, err := discoverer.Watch(getCtx, serviceDiscoverKey)
	if err != nil {
		return nil, fmt.Errorf("discover watch: %w", err)
	}
	logUpdate(watcher.Service(), endpoints)

	go watching(ctx, watcher)

	<-ctx.Done()
	if cerr := watcher.Close(); cerr != nil {
		slog.Warn("close watcher failed", "service", watcher.Service(), "err", cerr)
	}
	slog.Info("discover watch shutting down", "service", watcher.Service(), "cause", context.Cause(ctx))
	return endpoints, nil
}

func watching(ctx context.Context, watcher Watcher) {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("discoverer watching done", "service", watcher.Service())
			return
		default:
		}

		endpoints, err := watcher.Next()
		if err != nil {
			slog.Warn("discoverer watch error", "service", watcher.Service(), "err", err)
			time.Sleep(time.Second)
			continue
		}
		logUpdate(watcher.Service(), endpoints)
	}
}

func logUpdate(service string, endpoints []*Endpoint) {
	slog.Info("endpoints updated", "service", service, "count", len(endpoints))
	for _, ep := range endpoints {
		slog.Info("endpoint", "service", service, "address", ep.Address, "group", ep.Group)
	}
}

func NewEtcdDiscoverer(cli *clientv3.Client) Discoverer {
	return &etcdDiscoverer{
		cli: cli,
	}
}

func (d *etcdDiscoverer) Get(ctx context.Context, serviceDiscoverKey string) ([]*Endpoint, error) {
	serviceRegisterKey, group := parseServiceGroup(serviceDiscoverKey)

	resp, err := d.cli.Get(ctx, serviceRegisterKey, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", serviceDiscoverKey, err)
	}

	if resp.Header != nil {
		d.rev = resp.Header.GetRevision()
	}

	endpoints := etcdInstancesToEndpoints(resp.Kvs)
	return FilterOrDefault(endpoints, group), nil
}

func (d *etcdDiscoverer) Watch(ctx context.Context, serviceDiscoverKey string) ([]*Endpoint, Watcher, error) {
	endpoints, err := d.Get(ctx, serviceDiscoverKey)
	if err != nil {
		return nil, nil, err
	}

	w := newWatcher(d.cli, serviceDiscoverKey, endpoints, d.rev)
	return endpoints, w, nil
}

func parseServiceGroup(serviceKeyPrefix string) (string, string) {
	colonIndex := strings.LastIndex(serviceKeyPrefix, ":")

	if colonIndex == -1 {
		return serviceKeyPrefix, DefaultGroup
	}

	basePath := serviceKeyPrefix[:colonIndex]
	feature := serviceKeyPrefix[colonIndex+1:]

	feature = strings.TrimSuffix(feature, "/")

	return basePath + "/", feature
}
