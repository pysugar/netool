package extensions

import (
	"crypto/tls"
	"log/slog"
	"net/http/httptrace"
	"net/textproto"
)

func NewDebugClientTrace(prefix string) *httptrace.ClientTrace {
	log := slog.With("trace", prefix)
	return &httptrace.ClientTrace{
		GetConn: func(hostPort string) {
			log.Debug("GetConn", "hostPort", hostPort)
		},
		GotConn: func(info httptrace.GotConnInfo) {
			log.Debug("GotConn", "info", info)
		},
		PutIdleConn: func(err error) {
			log.Debug("PutIdleConn", "err", err)
		},
		GotFirstResponseByte: func() {
			log.Debug("GotFirstResponseByte")
		},
		Got100Continue: func() {
			log.Debug("Got100Continue")
		},
		Got1xxResponse: func(code int, header textproto.MIMEHeader) error {
			log.Debug("Got1xxResponse", "code", code, "header", header)
			return nil
		},
		DNSStart: func(info httptrace.DNSStartInfo) {
			log.Debug("DNSStart", "info", info)
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			log.Debug("DNSDone", "info", info)
		},
		ConnectStart: func(network, addr string) {
			log.Debug("ConnectStart", "network", network, "addr", addr)
		},
		ConnectDone: func(network, addr string, err error) {
			log.Debug("ConnectDone", "network", network, "addr", addr, "err", err)
		},
		TLSHandshakeStart: func() {
			log.Debug("TLSHandshakeStart")
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			log.Debug("TLSHandshakeDone", "state", state, "err", err)
		},
		WroteHeaderField: func(key string, value []string) {
			log.Debug("WroteHeaderField", "key", key, "value", value)
		},
		WroteHeaders: func() {
			log.Debug("WroteHeaders")
		},
		Wait100Continue: func() {
			log.Debug("Wait100Continue")
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			log.Debug("WroteRequest", "info", info)
		},
	}
}
