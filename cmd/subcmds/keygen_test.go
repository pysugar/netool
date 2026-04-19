package subcmds

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pysugar/netool/cmd/internal/cli"
	"github.com/pysugar/netool/cmd/internal/cmdtest"
	"github.com/spf13/cobra"
)

// newKeygenRoot mirrors what base/rootcmd.go sets up, minus everything we
// don't need for these tests. Keeping it local avoids pulling the entire
// command graph (and its side-effectful init()s) into the test binary.
func newKeygenRoot(t *testing.T) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "netool"}
	root.PersistentFlags().BoolP(cli.FlagVerbose, "V", false, "")
	root.PersistentFlags().StringP(cli.FlagOutput, "o", string(cli.FormatText), "")
	// keygenCmd and children are wired at package init, so we just reuse them.
	root.AddCommand(keygenCmd)
	return root
}

func TestKeygenX25519Text(t *testing.T) {
	res := cmdtest.Run(t, newKeygenRoot(t), "keygen", "x25519")
	if res.Err != nil {
		t.Fatalf("run: %v (stderr=%q)", res.Err, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "Private key:") || !strings.Contains(res.Stdout, "Public key:") {
		t.Fatalf("missing expected prefixes in output:\n%s", res.Stdout)
	}
}

func TestKeygenWgText(t *testing.T) {
	res := cmdtest.Run(t, newKeygenRoot(t), "keygen", "wg")
	if res.Err != nil {
		t.Fatalf("run: %v", res.Err)
	}
	if !strings.Contains(res.Stdout, "Private key:") {
		t.Fatalf("missing output: %s", res.Stdout)
	}
}

func TestKeygenJSON(t *testing.T) {
	res := cmdtest.Run(t, newKeygenRoot(t), "--output", "json", "keygen", "x25519")
	if res.Err != nil {
		t.Fatalf("run: %v", res.Err)
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(res.Stdout), &got); err != nil {
		t.Fatalf("JSON parse: %v; output=%q", err, res.Stdout)
	}
	if got["private"] == "" || got["public"] == "" {
		t.Fatalf("empty key fields: %+v", got)
	}
}

func TestKeygenUnknownSubcommand(t *testing.T) {
	res := cmdtest.Run(t, newKeygenRoot(t), "keygen", "ed25519-nope")
	if res.Err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
}
