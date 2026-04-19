// Package cmdtest provides a minimal harness for exercising cobra commands
// inside unit tests. It captures stdout/stderr, routes Context through the
// test deadline, and returns everything the test needs to assert on.
package cmdtest

import (
	"bytes"
	"context"
	"testing"

	"github.com/spf13/cobra"
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
func Run(t *testing.T, root *cobra.Command, args ...string) Result {
	t.Helper()

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
