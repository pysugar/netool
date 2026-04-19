package cmdtest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// StartHTTP starts an httptest.Server and registers t.Cleanup to close it.
// Returns the server so tests can read URL / Client() from it directly.
func StartHTTP(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}
