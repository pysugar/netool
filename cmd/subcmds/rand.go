package subcmds

import (
	"fmt"

	"github.com/pysugar/netool/cmd/base"
	"github.com/pysugar/netool/cmd/internal/cli"
	"github.com/pysugar/netool/uuid"
	"github.com/spf13/cobra"
)

var randStrCmd = &cobra.Command{
	Use:   `rand [-n 64]`,
	Short: "Generate a random string",
	Long: `
Generate a random alphanumeric string.

  netool rand -n 32
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		n, _ := cmd.Flags().GetInt("num")
		if n <= 0 {
			return fmt.Errorf("--num must be greater than 0 (got %d)", n)
		}
		s := uuid.GenerateRandomString(n)
		out := cli.NewOutput(cmd)
		if out.Format() == cli.FormatJSON {
			return out.JSON(map[string]string{"value": s})
		}
		out.Text("%s\n", s)
		return nil
	},
}

func init() {
	randStrCmd.Flags().IntP("num", "n", 32, "rand string length")
	base.AddSubCommands(randStrCmd)
}
