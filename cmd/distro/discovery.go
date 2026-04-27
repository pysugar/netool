package distro

import (
	"fmt"
	"strings"

	"github.com/pysugar/netool/cmd/base"
	"github.com/pysugar/netool/cmd/internal/cli"
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
				return fmt.Errorf("unsupported naming type: %s", namingType)
			}
			serviceName, _ := cmd.Flags().GetString("service")
			endpoints, _ := cmd.Flags().GetString("endpoints")
			envName, _ := cmd.Flags().GetString("env-name")
			group, _ := cmd.Flags().GetString("group")
			watchEnabled, _ := cmd.Flags().GetBool("watch")

			eps, err := fn(cmd.Context(), strings.Split(endpoints, ","), envName, serviceName, group, watchEnabled)
			if err != nil {
				return fmt.Errorf("discover %s: %w", namingType, err)
			}

			path := "/" + envName + "/" + serviceName + ":" + group
			out := cli.NewOutput(cmd)
			if out.Format() == cli.FormatJSON {
				type endpoint struct {
					Address string `json:"address"`
					Group   string `json:"group"`
				}
				rows := make([]endpoint, 0, len(eps))
				for _, ep := range eps {
					rows = append(rows, endpoint{Address: ep.Address, Group: ep.Group})
				}
				return out.JSON(map[string]any{
					"path":      path,
					"watch":     watchEnabled,
					"endpoints": rows,
				})
			}

			out.Text("path: %s (watch=%v)\n", path, watchEnabled)
			for _, ep := range eps {
				out.Text("\t%s\t%s\n", ep.Address, ep.Group)
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
