package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// OutputFormat enumerates the values accepted by the root --output flag.
type OutputFormat string

const (
	FormatText OutputFormat = "text"
	FormatJSON OutputFormat = "json"
)

// Output is the rendering surface for user-facing command results. It is
// intentionally minimal — diagnostic / trace output belongs in slog, not here.
type Output interface {
	Text(format string, args ...any)
	JSON(v any) error
	Writer() io.Writer
	Format() OutputFormat
}

// NewOutput resolves the effective output format for cmd. If --output is not
// set anywhere in the ancestor chain, it defaults to text. Text writes to
// cmd.OutOrStdout(); JSON always writes to stdout so pipelines are stable even
// when a command also emits human-readable diagnostics through cobra.
func NewOutput(cmd *cobra.Command) Output {
	f := FormatText
	if v, err := cmd.Flags().GetString(FlagOutput); err == nil && v != "" {
		f = OutputFormat(strings.ToLower(v))
	}
	return &stdOutput{format: f, w: cmd.OutOrStdout()}
}

type stdOutput struct {
	format OutputFormat
	w      io.Writer
}

func (o *stdOutput) Text(format string, args ...any) {
	if o.format == FormatJSON {
		return
	}
	fmt.Fprintf(o.w, format, args...)
}

func (o *stdOutput) JSON(v any) error {
	enc := json.NewEncoder(o.w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func (o *stdOutput) Writer() io.Writer { return o.w }

func (o *stdOutput) Format() OutputFormat { return o.format }

// ConfigureLogging switches the default slog logger to human-friendly text on
// stderr and raises the level to debug when --verbose is set.
func ConfigureLogging(cmd *cobra.Command) {
	level := slog.LevelInfo
	if Verbose(cmd) {
		level = slog.LevelDebug
	}
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(h))
}
