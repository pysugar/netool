package subcmds

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/pysugar/netool/cmd/internal/cli"
	"github.com/pysugar/netool/cmd/internal/cmdtest"
	"github.com/spf13/cobra"
)

// newFetchRoot mirrors what cmd/base sets up but only registers fetchCmd.
// We avoid base.AddSubCommands' init() side effects by attaching directly.
func newFetchRoot(t *testing.T) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "netool"}
	root.PersistentFlags().BoolP(cli.FlagVerbose, "V", false, "")
	root.PersistentFlags().StringP(cli.FlagOutput, "o", string(cli.FormatText), "")
	root.AddCommand(fetchCmd)
	return root
}

func TestFetchHTTP1Text(t *testing.T) {
	srv := cmdtest.StartHTTP(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Trace", "abc")
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("payload"))
	}))

	res := cmdtest.Run(t, newFetchRoot(t), "fetch", "--http1", srv.URL)
	if res.Err != nil {
		t.Fatalf("run: %v (stderr=%q)", res.Err, res.Stderr)
	}
	if !strings.HasPrefix(res.Stdout, "418 I'm a teapot HTTP/1.1\r\n") {
		t.Fatalf("status line not at start of output:\n%s", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "X-Trace: abc\r\n") {
		t.Fatalf("missing X-Trace header in output:\n%s", res.Stdout)
	}
	if !strings.HasSuffix(res.Stdout, "\r\npayload") {
		t.Fatalf("body not at end of output:\n%s", res.Stdout)
	}
}

func TestFetchSendsCustomHeaderAndUserAgent(t *testing.T) {
	var gotUA, gotHdr string
	srv := cmdtest.StartHTTP(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotHdr = r.Header.Get("X-Custom")
		w.WriteHeader(http.StatusOK)
	}))

	res := cmdtest.Run(t, newFetchRoot(t),
		"fetch", "--http1", "-A", "netool/test", "-H", "X-Custom: hello", srv.URL)
	if res.Err != nil {
		t.Fatalf("run: %v", res.Err)
	}
	if gotUA != "netool/test" {
		t.Fatalf("User-Agent not propagated: %q", gotUA)
	}
	if gotHdr != "hello" {
		t.Fatalf("custom header not propagated: %q", gotHdr)
	}
}

func TestFetchPostBody(t *testing.T) {
	var gotMethod, gotBody string
	srv := cmdtest.StartHTTP(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotBody = string(buf)
		w.WriteHeader(http.StatusNoContent)
	}))

	res := cmdtest.Run(t, newFetchRoot(t),
		"fetch", "--http1", "-M", "POST", "-d", `{"k":"v"}`, srv.URL)
	if res.Err != nil {
		t.Fatalf("run: %v", res.Err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", gotMethod)
	}
	if gotBody != `{"k":"v"}` {
		t.Fatalf("body = %q, want %q", gotBody, `{"k":"v"}`)
	}
}

func TestFetchInvalidURL(t *testing.T) {
	res := cmdtest.Run(t, newFetchRoot(t), "fetch", "://not-a-url")
	if res.Err == nil {
		t.Fatalf("expected error for invalid url, got stdout=%q", res.Stdout)
	}
}

func TestFetchInvalidHeaderIsSkipped(t *testing.T) {
	// Malformed -H entries should not crash; the server should still receive
	// the well-formed one, and the response prints normally.
	var gotGood string
	srv := cmdtest.StartHTTP(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotGood = r.Header.Get("X-Good")
		w.WriteHeader(http.StatusOK)
	}))

	res := cmdtest.Run(t, newFetchRoot(t),
		"fetch", "--http1", "-H", "no-colon-here", "-H", "X-Good: yes", srv.URL)
	if res.Err != nil {
		t.Fatalf("run: %v", res.Err)
	}
	if gotGood != "yes" {
		t.Fatalf("good header not received: %q", gotGood)
	}
}

// Sanity check that the standard JSON output mode is still wired (this
// command currently emits the response verbatim either way; the test exists
// to catch regressions if that changes).
func TestFetchOutputJSONFlagAccepted(t *testing.T) {
	srv := cmdtest.StartHTTP(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	res := cmdtest.Run(t, newFetchRoot(t),
		"--output", "json", "fetch", "--http1", srv.URL)
	if res.Err != nil {
		t.Fatalf("run: %v", res.Err)
	}
	// The body is well-formed JSON; verify it survived round-trip.
	bodyStart := strings.Index(res.Stdout, "{")
	if bodyStart < 0 {
		t.Fatalf("no JSON body in output:\n%s", res.Stdout)
	}
	var got map[string]bool
	if err := json.Unmarshal([]byte(res.Stdout[bodyStart:]), &got); err != nil {
		t.Fatalf("body not valid JSON: %v\noutput:%s", err, res.Stdout)
	}
	if !got["ok"] {
		t.Fatalf("unexpected body payload: %+v", got)
	}
}
