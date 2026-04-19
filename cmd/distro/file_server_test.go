package distro

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunFileServerServesAndShutsDown(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hi"), 0644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use port 0? runFileServer takes a fixed port. Pick an unlikely-free port.
	port := 18981 + (int(time.Now().UnixNano()/1e6) % 500)

	done := make(chan error, 1)
	go func() { done <- runFileServer(ctx, dir, port, false) }()

	// Wait for the listener to be up.
	url := "http://127.0.0.1:" + itoa(port) + "/hello.txt"
	if !waitForGet(t, url, 2*time.Second) {
		cancel()
		<-done
		t.Fatal("server did not come up in time")
	}

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "hi" {
		t.Fatalf("unexpected body: %q", body)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runFileServer returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("runFileServer did not return after cancel")
	}
}

func waitForGet(t *testing.T, url string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return true
		}
		time.Sleep(30 * time.Millisecond)
	}
	return false
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	n := len(buf)
	for i > 0 {
		n--
		buf[n] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[n:])
}
