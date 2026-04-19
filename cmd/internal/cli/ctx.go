package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// SignalContext returns a context cancelled on SIGINT / SIGTERM. Use this as
// the base for long-running server commands that need graceful shutdown.
func SignalContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}

// RunContext builds the standard command context: parent is cobra's context
// (honours global cancellation), combined with --timeout (or def when the flag
// is absent) and SIGINT/SIGTERM cancellation.
func RunContext(cmd *cobra.Command, def time.Duration) (context.Context, context.CancelFunc) {
	parent := cmd.Context()
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancelSig := SignalContext(parent)

	timeout := Timeout(cmd, def)
	if timeout <= 0 {
		return ctx, cancelSig
	}
	ctx, cancelTO := context.WithTimeout(ctx, timeout)
	return ctx, func() {
		cancelTO()
		cancelSig()
	}
}
