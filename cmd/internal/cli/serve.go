package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// StartFunc launches the server. It should block until the server stops on
// its own (e.g. http.Serve returning) or a fatal error occurs. Returning
// http.ErrServerClosed (or any error wrapped equivalent) is considered a
// clean shutdown, not a failure.
type StartFunc func(ctx context.Context) error

// StopFunc performs graceful shutdown. It is invoked when ctx is cancelled
// (SIGINT/SIGTERM or timeout). Implementations should honour the passed
// shutdown deadline.
type StopFunc func(ctx context.Context) error

// RunServer runs start in the foreground and triggers stop when ctx is
// cancelled. It returns only after start has exited — giving the caller a
// well-defined lifetime. Pass nil for stop when the server exits cleanly
// simply by cancelling ctx.
func RunServer(ctx context.Context, name string, start StartFunc, stop StopFunc) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- start(ctx)
	}()

	select {
	case err := <-errCh:
		return normalizeServeError(err)
	case <-ctx.Done():
		slog.Info("shutdown signal received", "server", name, "cause", context.Cause(ctx))
	}

	if stop != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownGrace)
		defer cancel()
		if err := stop(shutdownCtx); err != nil {
			return fmt.Errorf("%s shutdown: %w", name, err)
		}
	}
	return normalizeServeError(<-errCh)
}

// ShutdownGrace bounds how long RunServer waits for stop() to return.
// Exposed as a var so tests can shrink it.
var ShutdownGrace = 10 * time.Second

func normalizeServeError(err error) error {
	if err == nil {
		return nil
	}
	// http.ErrServerClosed pattern without importing net/http at this layer.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	if err.Error() == "http: Server closed" {
		return nil
	}
	return err
}
