package base

import (
	"context"
	"fmt"
	"os"

	"github.com/pysugar/netool/cmd/internal/cli"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "netool",
	Short: "net tool",
	Long:  "A simple CLI for Net tool",
	// Cobra prints the full Usage block for any RunE error by default,
	// which buries the actual error message under a wall of help. Errors
	// are still printed (we don't silence those), just without the dump.
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cli.ConfigureLogging(cmd)
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello, this is a net tool")
	},
}

func init() {
	rootCmd.PersistentFlags().BoolP(cli.FlagVerbose, "V", false, "verbose output (debug-level logging)")
	rootCmd.PersistentFlags().StringP(cli.FlagOutput, "o", string(cli.FormatText), "output format: text|json")
	rootCmd.PersistentFlags().String(cli.FlagLogFormat, cli.LogFormatText, "log handler format: text|json")
}

func AddSubCommands(cmds ...*cobra.Command) {
	cli.Register(rootCmd, cmds...)
}

func Run() {
	ctx, cancel := cli.SignalContext(context.Background())
	defer cancel()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		// cobra has already printed "Error: <msg>" to stderr; just exit.
		os.Exit(1)
	}
}
