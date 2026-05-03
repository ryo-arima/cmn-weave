// Package share holds cross-cutting helpers used by the client component:
// TLS configuration builders and other shared utilities. The package is kept
// flat — no subdirectories — per the repository layout policy.
package share

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/ryo-arima/cmn-weave/pkg/config"
)

// ClientTLSConfig returns a *tls.Config suitable for an mTLS HTTP/gRPC
// client. The returned config presents the leaf certificate declared in cfg
// and verifies the peer server certificate against the CA bundle.
func ClientTLSConfig(cfg config.TLS) (*tls.Config, error) {
	cert, pool, err := loadTLS(cfg)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
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
