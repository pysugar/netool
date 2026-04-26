package subcmds

import (
	"fmt"

	"github.com/pysugar/netool/cmd/base"
	"github.com/pysugar/netool/cmd/internal/cli"
	"github.com/pysugar/netool/uuid"
	"github.com/spf13/cobra"
)

var uuidCmd = &cobra.Command{
	Use:   `uuid [-i "example"]`,
	Short: "Generate UUIDv4 or UUIDv5",
	Long: `
Generate UUIDv4 or UUIDv5.

UUIDv4 (random):     netool uuid
UUIDv5 (from input): netool uuid -i "example"
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		input, _ := cmd.Flags().GetString("input")
		var (
			u   uuid.UUID
			err error
		)
		switch {
		case input == "":
			u = uuid.New()
		case len(input) > 30:
			return fmt.Errorf("input must be within 30 bytes (got %d)", len(input))
		default:
			u, err = uuid.ParseString(input)
			if err != nil {
				return fmt.Errorf("parse input: %w", err)
			}
		}
		out := cli.NewOutput(cmd)
		if out.Format() == cli.FormatJSON {
			return out.JSON(map[string]string{"uuid": u.String()})
		}
		out.Text("%s\n", u.String())
		return nil
	},
}

func init() {
	uuidCmd.Flags().StringP("input", "i", "", "seed for UUIDv5 (omit for random UUIDv4)")
	base.AddSubCommands(uuidCmd)
}
