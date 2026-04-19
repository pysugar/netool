package distro

import (
	"log/slog"
	"strings"

	"github.com/pysugar/netool/cmd/base"
	etcddisc "github.com/pysugar/netool/cmd/internal/discovery/etcd"
	"github.com/spf13/cobra"
)

var (
	NamingRegistryServices = map[string]etcddisc.RegisterNamingService{
		"etcd": etcddisc.RegisterETCD,
	}

	registryCmd = &cobra.Command{
		Use:   "registry --service=name --address=host:port [--naming-type=etcd] [--endpoints=127.0.0.1:2379] [--env-name=live]",
		Short: "Register a service into a naming registry",
		Long: `
Register a service into a naming registry.

Register: netool registry --endpoints=127.0.0.1:2379 --env-name=live --service=svc --address=192.168.1.5:8080

Verify via etcd:
  ETCDCTL_API=3 etcdctl get '/live/svc' --endpoints=127.0.0.1:2379 --prefix
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			namingType, _ := cmd.Flags().GetString("naming-type")
			fn, ok := NamingRegistryServices[namingType]
			if !ok {
				slog.Error("unsupported naming type", "type", namingType)
				return nil
			}
			serviceName, _ := cmd.Flags().GetString("service")
			endpoints, _ := cmd.Flags().GetString("endpoints")
			address, _ := cmd.Flags().GetString("address")
			envName, _ := cmd.Flags().GetString("env-name")

			if err := fn(strings.Split(endpoints, ","), envName, serviceName, address); err != nil {
				slog.Error("register failed", "type", namingType, "err", err)
				return err
			}
			return nil
		},
	}
)

func init() {
	registryCmd.Flags().String("endpoints", "127.0.0.1:2379", "naming service addresses (comma-separated)")
	registryCmd.Flags().StringP("naming-type", "t", "etcd", "naming service type")
	registryCmd.Flags().StringP("env-name", "e", "live", "env name")
	registryCmd.Flags().StringP("service", "s", "", "your service")
	registryCmd.Flags().StringP("address", "a", "", "your service address")
	base.AddSubCommands(registryCmd)
}
