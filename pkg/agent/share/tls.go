// Package share holds cross-cutting helpers for the agent component.
// The package is kept flat — no subdirectories.
package share

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/ryo-arima/cmn-weave/pkg/config"
)

// ServerTLSConfig returns a *tls.Config for the agent's gRPC listener.
// It enforces TLS 1.3 and requires client certificates (mTLS).
func ServerTLSConfig(cfg config.TLS) (*tls.Config, error) {
	cert, pool, err := loadTLS(cfg)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}, nil
}

func loadTLS(cfg config.TLS) (tls.Certificate, *x509.CertPool, error) {
	if err := cfg.Validate(); err != nil {
		return tls.Certificate{}, nil, err
	}
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("load keypair: %w", err)
	}
	caPEM, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("read ca bundle: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return tls.Certificate{}, nil, fmt.Errorf("ca bundle %q contains no usable certificates", cfg.CAFile)
	}
	return cert, pool, nil
}
