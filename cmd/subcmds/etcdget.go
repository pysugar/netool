package subcmds

import (
	"strings"
	"time"

	"github.com/pysugar/netool/cmd/base"
	"github.com/pysugar/netool/cmd/internal/cli"
	"github.com/spf13/cobra"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var etcdGetCmd = &cobra.Command{
	Use:   "etcdget --key=KEY [--endpoints=127.0.0.1:2379]",
	Short: "Get etcd values for a given key",
	Long: `
Get etcd values for a given key.

Examples:
  netool etcdget --key=/live/myservice
  netool etcdget --key=/live/ --prefix --endpoints=127.0.0.1:2379,127.0.0.1:2380
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		endpoints, _ := cmd.Flags().GetString("endpoints")
		client, err := clientv3.New(clientv3.Config{
			Endpoints: strings.Split(endpoints, ","),
		})
		if err != nil {
			return err
		}
		defer client.Close()

		ctx, cancel := cli.RunContext(cmd, 30*time.Second)
		defer cancel()

		key, _ := cmd.Flags().GetString("key")
		limit, _ := cmd.Flags().GetInt64("limit")
		prefix, _ := cmd.Flags().GetBool("prefix")

		options := []clientv3.OpOption{clientv3.WithLimit(limit)}
		if prefix {
			options = append(options, clientv3.WithPrefix())
		}
		resp, err := client.Get(ctx, key, options...)
		if err != nil {
			return err
		}

		out := cli.NewOutput(cmd)
		if out.Format() == cli.FormatJSON {
			type pair struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			}
			rows := make([]pair, 0, len(resp.Kvs))
			for _, kv := range resp.Kvs {
				rows = append(rows, pair{Key: string(kv.Key), Value: string(kv.Value)})
			}
			return out.JSON(map[string]any{"results": rows})
		}
		for _, kv := range resp.Kvs {
			out.Text("%s : %s\n", kv.Key, string(kv.Value))
		}
		return nil
	},
}

func init() {
	etcdGetCmd.Flags().String("endpoints", "127.0.0.1:2379", "etcd server addresses (comma-separated)")
	etcdGetCmd.Flags().String("key", "", "search key")
	etcdGetCmd.Flags().Int64("limit", 100, "max results")
	etcdGetCmd.Flags().Bool("prefix", false, "treat key as prefix")
	cli.AddTimeout(etcdGetCmd, 0)
	base.AddSubCommands(etcdGetCmd)
}
