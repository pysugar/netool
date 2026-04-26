package cli

import (
	"log/slog"
	"net/http"
	"strings"
)

// ParseHeaders parses repeatable "Key: Value" strings (typically collected via
// `-H` / --header) into an http.Header. Malformed entries are dropped with a
// warning rather than failing the command, matching curl behaviour.
func ParseHeaders(raw []string) http.Header {
	h := http.Header{}
	for _, item := range raw {
		k, v, ok := strings.Cut(item, ":")
		if !ok {
			slog.Warn("invalid header, expected K:V", "header", item)
			continue
		}
		h.Add(strings.TrimSpace(k), strings.TrimSpace(v))
	}
	return h
}
