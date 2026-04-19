package cli

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// Target is the parsed form of a network endpoint string accepted by CLI
// commands. It normalizes the three shapes we see across fetch / grpc / etcd:
//
//	host:port
//	scheme://host:port/path
//	[ipv6]:port
type Target struct {
	Scheme string
	Host   string
	Port   int
	Path   string
	Raw    string
}

// HostPort returns host:port, properly bracketing IPv6 literals.
func (t *Target) HostPort() string {
	if t.Port == 0 {
		return t.Host
	}
	return net.JoinHostPort(t.Host, strconv.Itoa(t.Port))
}

// ParseTarget accepts the three forms described on Target and returns a
// populated struct. An empty input is an error.
func ParseTarget(raw string) (*Target, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty target")
	}
	t := &Target{Raw: raw}

	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("parse url: %w", err)
		}
		if u.Host == "" {
			return nil, fmt.Errorf("url missing host: %s", raw)
		}
		t.Scheme = u.Scheme
		t.Host = u.Hostname()
		if p := u.Port(); p != "" {
			n, err := strconv.Atoi(p)
			if err != nil {
				return nil, fmt.Errorf("invalid port %q: %w", p, err)
			}
			t.Port = n
		} else {
			t.Port = defaultPort(u.Scheme)
		}
		t.Path = u.RequestURI()
		if t.Path == "/" {
			t.Path = ""
		}
		return t, nil
	}

	host, port, err := net.SplitHostPort(raw)
	if err != nil {
		// Bare host — no port. Accept it; callers can decide whether that's valid.
		t.Host = raw
		return t, nil
	}
	t.Host = host
	n, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("invalid port %q: %w", port, err)
	}
	t.Port = n
	return t, nil
}

func defaultPort(scheme string) int {
	switch strings.ToLower(scheme) {
	case "http", "ws":
		return 80
	case "https", "wss":
		return 443
	case "grpc":
		return 50051
	}
	return 0
}
