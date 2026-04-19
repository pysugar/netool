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
}

func AddSubCommands(cmds ...*cobra.Command) {
	cli.Register(rootCmd, cmds...)
}

func Run() {
	ctx, cancel := cli.SignalContext(context.Background())
	defer cancel()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
