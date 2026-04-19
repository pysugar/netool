package cli

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

const (
	FlagVerbose = "verbose"
	FlagOutput  = "output"
	FlagTimeout = "timeout"

	FlagTLSInsecure = "insecure"
	FlagTLSCA       = "tls-ca"
	FlagTLSCert     = "tls-cert"
	FlagTLSKey      = "tls-key"
	FlagTLSServer   = "tls-server-name"
)

// Verbose reports whether --verbose is set on the command or any ancestor.
func Verbose(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool(FlagVerbose)
	return v
}

// AddTimeout registers --timeout on cmd with the given default.
func AddTimeout(cmd *cobra.Command, def time.Duration) {
	cmd.Flags().Duration(FlagTimeout, def, "operation timeout")
}

// Timeout returns the value of --timeout if set, otherwise def.
func Timeout(cmd *cobra.Command, def time.Duration) time.Duration {
	if !cmd.Flags().Changed(FlagTimeout) {
		return def
	}
	d, err := cmd.Flags().GetDuration(FlagTimeout)
	if err != nil {
		return def
	}
	return d
}

// TLSFlags holds values bound by AddTLS.
type TLSFlags struct {
	Insecure   bool
	CAFile     string
	CertFile   string
	KeyFile    string
	ServerName string
}

// AddTLS registers the standard TLS flag group on cmd. Returns a pointer whose
// fields are populated when cobra parses the command line.
func AddTLS(cmd *cobra.Command) *TLSFlags {
	t := &TLSFlags{}
	cmd.Flags().BoolVar(&t.Insecure, FlagTLSInsecure, false, "skip TLS certificate verification")
	cmd.Flags().StringVar(&t.CAFile, FlagTLSCA, "", "path to CA bundle for server verification")
	cmd.Flags().StringVar(&t.CertFile, FlagTLSCert, "", "path to client certificate (PEM)")
	cmd.Flags().StringVar(&t.KeyFile, FlagTLSKey, "", "path to client private key (PEM)")
	cmd.Flags().StringVar(&t.ServerName, FlagTLSServer, "", "override TLS ServerName (SNI)")
	return t
}

// Config builds a *tls.Config from parsed flag values. Returns nil, nil when
// no TLS options were supplied — caller decides whether to default to plain or
// a vanilla &tls.Config{}.
func (t *TLSFlags) Config() (*tls.Config, error) {
	if t == nil {
		return nil, nil
	}
	if !t.Insecure && t.CAFile == "" && t.CertFile == "" && t.KeyFile == "" && t.ServerName == "" {
		return nil, nil
	}
	cfg := &tls.Config{
		InsecureSkipVerify: t.Insecure,
		ServerName:         t.ServerName,
	}
	if t.CAFile != "" {
		pem, err := os.ReadFile(t.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read ca file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no certificates parsed from %s", t.CAFile)
		}
		cfg.RootCAs = pool
	}
	if (t.CertFile == "") != (t.KeyFile == "") {
		return nil, fmt.Errorf("--%s and --%s must be set together", FlagTLSCert, FlagTLSKey)
	}
	if t.CertFile != "" {
		cert, err := tls.LoadX509KeyPair(t.CertFile, t.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load client cert: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	return cfg, nil
}
