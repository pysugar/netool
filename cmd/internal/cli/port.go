package cli

import "github.com/spf13/cobra"

const FlagPort = "port"

// AddPort registers the canonical -p/--port flag on cmd. Use this for any
// server command so the short -p stays reserved for "port" across the CLI.
func AddPort(cmd *cobra.Command, def int) {
	cmd.Flags().IntP(FlagPort, "p", def, "listen port")
}

// Port returns the parsed --port value, or 0 if the flag is not registered.
func Port(cmd *cobra.Command) int {
	v, _ := cmd.Flags().GetInt(FlagPort)
	return v
}
