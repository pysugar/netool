package cli

import "github.com/spf13/cobra"

func Register(parent *cobra.Command, children ...*cobra.Command) {
	parent.AddCommand(children...)
}
