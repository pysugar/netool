package cli

import "testing"

func TestParseTarget(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    Target
		wantErr bool
	}{
		{name: "host:port", raw: "127.0.0.1:2379", want: Target{Host: "127.0.0.1", Port: 2379}},
		{name: "https URL", raw: "https://example.com/foo", want: Target{Scheme: "https", Host: "example.com", Port: 443, Path: "/foo"}},
		{name: "http URL default port", raw: "http://example.com", want: Target{Scheme: "http", Host: "example.com", Port: 80}},
		{name: "ipv6 host:port", raw: "[::1]:8080", want: Target{Host: "::1", Port: 8080}},
		{name: "grpc scheme", raw: "grpc://svc.local/method", want: Target{Scheme: "grpc", Host: "svc.local", Port: 50051, Path: "/method"}},
		{name: "bare host", raw: "example.com", want: Target{Host: "example.com"}},
		{name: "empty", raw: "", wantErr: true},
		{name: "bad port", raw: "host:abc", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseTarget(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got %+v", tc.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.raw, err)
			}
			if got.Scheme != tc.want.Scheme || got.Host != tc.want.Host ||
				got.Port != tc.want.Port || got.Path != tc.want.Path {
				t.Fatalf("ParseTarget(%q)\n  got:  %+v\n  want: %+v", tc.raw, got, tc.want)
			}
			if got.Raw != tc.raw {
				t.Errorf("Raw round-trip: got %q, want %q", got.Raw, tc.raw)
			}
		})
	}
}

func TestTargetHostPort(t *testing.T) {
	if hp := (&Target{Host: "127.0.0.1", Port: 8080}).HostPort(); hp != "127.0.0.1:8080" {
		t.Errorf("v4 host:port: got %q", hp)
	}
	if hp := (&Target{Host: "::1", Port: 443}).HostPort(); hp != "[::1]:443" {
		t.Errorf("v6 host:port: got %q", hp)
	}
	if hp := (&Target{Host: "example.com"}).HostPort(); hp != "example.com" {
		t.Errorf("port=0 should return host only: got %q", hp)
	}
}
