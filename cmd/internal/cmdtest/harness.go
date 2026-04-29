// Package cmdtest provides a minimal harness for exercising cobra commands
// inside unit tests. It captures stdout/stderr, routes Context through the
// test deadline, and returns everything the test needs to assert on.
package cmdtest

import (
	"bytes"
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Result bundles the output of a single command invocation.
type Result struct {
	Stdout string
	Stderr string
	Err    error
}

// Run executes cmd with args, captures its output, and returns the result.
// The command's context is set to t.Context() (or context.Background() on
// older Go versions) so tests can cancel work via t.Cleanup.
//
// Cobra commands are typically package-level vars, so both flag values and
// per-command ctx bleed between successive Run() calls in the same test
// binary. Cobra v1.8.1 only inherits the parent ctx when the subcommand's
// ctx is nil (command.go:1113), so a previous test's t.Context() — already
// canceled by t.Cleanup — would otherwise persist on the subcommand and
// every dial in a follow-up test would fail with "context canceled". Each
// Run resets both flags and ctx on the entire command tree so tests stay
// independent regardless of execution order.
func Run(t *testing.T, root *cobra.Command, args ...string) Result {
	t.Helper()

	resetFlags(root)
	resetContext(root)

	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)

	ctx := context.Background()
	if cd, ok := any(t).(interface{ Context() context.Context }); ok {
		ctx = cd.Context()
	}

	err := root.ExecuteContext(ctx)
	return Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Err:    err,
	}
}

func resetFlags(cmd *cobra.Command) {
	resetFlagSet(cmd.Flags())
	resetFlagSet(cmd.PersistentFlags())
	for _, c := range cmd.Commands() {
		resetFlags(c)
	}
}

func resetContext(cmd *cobra.Command) {
	cmd.SetContext(nil)
	for _, c := range cmd.Commands() {
		resetContext(c)
	}
}

func resetFlagSet(fs *pflag.FlagSet) {
	fs.VisitAll(func(f *pflag.Flag) {
		if !f.Changed {
			return
		}
		// Slice/array values accumulate via Append; SliceValue exposes Replace
		// which atomically swaps the underlying slice. Fall back to Set for
		// scalar flags.
		if sv, ok := f.Value.(pflag.SliceValue); ok {
			_ = sv.Replace(nil)
		} else {
			_ = f.Value.Set(f.DefValue)
		}
		f.Changed = false
	})
}
