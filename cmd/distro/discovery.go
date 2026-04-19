package distro

import (
	"log/slog"
	"strings"

	"github.com/pysugar/netool/cmd/base"
	etcddisc "github.com/pysugar/netool/cmd/internal/discovery/etcd"
	"github.com/spf13/cobra"
)

var (
	NamingDiscoverGetServices = map[string]etcddisc.DiscoverNamingService{
		"etcd": etcddisc.DiscoverETCD,
	}

	discoveryCmd = &cobra.Command{
		Use:   "discovery --service=name [--naming-type=etcd] [--endpoints=127.0.0.1:2379] [--env-name=live] [--watch]",
		Short: "Discover services from a naming registry",
		Long: `
Discover services from a naming registry.

Discover: netool discovery --endpoints=127.0.0.1:2379 --env-name=live --service=svc --watch
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			namingType, _ := cmd.Flags().GetString("naming-type")
			fn, ok := NamingDiscoverGetServices[namingType]
			if !ok {
				slog.Error("unsupported naming type", "type", namingType)
				return nil
			}
			serviceName, _ := cmd.Flags().GetString("service")
			endpoints, _ := cmd.Flags().GetString("endpoints")
			envName, _ := cmd.Flags().GetString("env-name")
			group, _ := cmd.Flags().GetString("group")
			watchEnabled, _ := cmd.Flags().GetBool("watch")

			eps, err := fn(strings.Split(endpoints, ","), envName, serviceName, group, watchEnabled)
			if err != nil {
				slog.Error("discover failed", "type", namingType, "err", err)
				return err
			}
			slog.Info("discovered",
				"watch", watchEnabled,
				"path", "/"+envName+"/"+serviceName+":"+group)
			for _, ep := range eps {
				slog.Info("endpoint", "address", ep.Address, "group", ep.Group)
			}
			return nil
		},
	}
)

func init() {
	discoveryCmd.Flags().String("endpoints", "127.0.0.1:2379", "naming service addresses (comma-separated)")
	discoveryCmd.Flags().StringP("naming-type", "t", "etcd", "naming service type")
	discoveryCmd.Flags().StringP("env-name", "e", "live", "env name")
	discoveryCmd.Flags().StringP("service", "s", "", "your service")
	discoveryCmd.Flags().StringP("group", "g", "default", "group")
	discoveryCmd.Flags().BoolP("watch", "w", false, "watch enabled")
	base.AddSubCommands(discoveryCmd)
}
