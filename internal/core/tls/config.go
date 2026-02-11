package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// Config holds TLS configuration for mTLS and certificate management.
type Config struct {
	CertFile           string `yaml:"cert_file,omitempty"`
	KeyFile            string `yaml:"key_file,omitempty"`
	CAFile             string `yaml:"ca_file,omitempty"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify,omitempty"`
}

// BuildTLSConfig creates a *tls.Config from the configuration.
func (c *Config) BuildTLSConfig() (*tls.Config, error) {
	if c == nil {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.InsecureSkipVerify,
	}

	// Load client certificate for mTLS
	if c.CertFile != "" && c.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("loading client cert: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate
	if c.CAFile != "" {
		caCert, err := os.ReadFile(c.CAFile)
		if err != nil {
			return nil, fmt.Errorf("reading CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA cert")
		}
		tlsConfig.RootCAs = pool
	}

	return tlsConfig, nil
}

// IsEmpty returns true if no TLS settings are configured.
func (c *Config) IsEmpty() bool {
	if c == nil {
		return true
	}
	return c.CertFile == "" && c.KeyFile == "" && c.CAFile == "" && !c.InsecureSkipVerify
}
