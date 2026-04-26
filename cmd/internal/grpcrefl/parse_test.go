package grpcrefl

import (
	"net/url"
	"testing"
)

func TestParseFullMethod(t *testing.T) {
	cases := []struct {
		in              string
		wantSvc, wantM  string
		wantErr         bool
	}{
		{in: "proto.EchoService/Echo", wantSvc: "proto.EchoService", wantM: "Echo"},
		{in: "/proto.EchoService/Echo/", wantSvc: "proto.EchoService", wantM: "Echo"},
		{in: "/proto.EchoService/Echo", wantSvc: "proto.EchoService", wantM: "Echo"},
		{in: "proto.EchoService/Echo/", wantSvc: "proto.EchoService", wantM: "Echo"},

		// Errors.
		{in: "", wantErr: true},
		{in: "/", wantErr: true},
		{in: "Echo", wantErr: true},
		{in: "a/b/c", wantErr: true},
		{in: "/Echo/", wantErr: true},
		{in: "Echo/", wantErr: true},
		{in: "/Echo", wantErr: true},
		{in: "Foo//Bar", wantErr: true}, // empty middle segment after split
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			svc, m, err := ParseFullMethod(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got svc=%q m=%q", svc, m)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if svc != tc.wantSvc || m != tc.wantM {
				t.Fatalf("got (%q, %q), want (%q, %q)", svc, m, tc.wantSvc, tc.wantM)
			}
		})
	}
}

func TestParseURLPath(t *testing.T) {
	cases := []struct {
		raw             string
		wantSvc, wantM  string
		wantErr         bool
	}{
		{raw: "http://h:50051/proto.EchoService/Echo", wantSvc: "proto.EchoService", wantM: "Echo"},
		// Extra leading prefix segments (e.g. context-path mounting) are tolerated;
		// the last two segments win.
		{raw: "http://h:50051/api/v1/proto.EchoService/Echo", wantSvc: "proto.EchoService", wantM: "Echo"},
		{raw: "http://h:50051/grpc/proto.EchoService/Echo/", wantSvc: "proto.EchoService", wantM: "Echo"},

		// Errors.
		{raw: "http://h:50051/", wantErr: true},
		{raw: "http://h:50051/Echo", wantErr: true},
		{raw: "http://h:50051", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			u, err := url.Parse(tc.raw)
			if err != nil {
				t.Fatalf("url.Parse: %v", err)
			}
			svc, m, err := ParseURLPath(u)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got svc=%q m=%q", svc, m)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if svc != tc.wantSvc || m != tc.wantM {
				t.Fatalf("got (%q, %q), want (%q, %q)", svc, m, tc.wantSvc, tc.wantM)
			}
		})
	}
}
