// Package config loads runtime configuration for cmn-weave components.
//
// All three components (server, agent, client) share a single YAML schema
// defined by YamlConfig. A deployment may use one unified file for all
// components, or separate per-component files that each contain only the
// relevant sections.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// YamlConfig is the top-level configuration struct shared by all components.
type YamlConfig struct {
	Application Application `yaml:"application"`
	PostgreSQL   PostgreSQL  `yaml:"postgresql"`
	Redis        Redis       `yaml:"redis"`
}

// Application groups the per-component runtime settings.
type Application struct {
	Server Server `yaml:"server"`
	Agent  Agent  `yaml:"agent"`
	Client Client `yaml:"client"`
}

// TLS holds the file paths for the X.509 material used by every component.
//
// All paths are loaded at process start. Hot reload is out of scope for the
// initial implementation; restart the process to pick up rotated certs.
type TLS struct {
	// CAFile is the PEM-encoded CA bundle used to verify the peer.
	CAFile string `yaml:"ca_file"`
	// CertFile is this component's leaf certificate (PEM).
	CertFile string `yaml:"cert_file"`
	// KeyFile is the private key matching CertFile (PEM).
	KeyFile string `yaml:"key_file"`
}

// Validate ensures the TLS section points at readable files.
func (t TLS) Validate() error {
	for label, path := range map[string]string{
		"ca_file":   t.CAFile,
		"cert_file": t.CertFile,
		"key_file":  t.KeyFile,
	} {
		if path == "" {
			return fmt.Errorf("tls.%s is required", label)
		}
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("tls.%s: %w", label, err)
		}
	}
	return nil
}

// Server holds configuration for the REST server component.
type Server struct {
	// ListenAddr is the address the REST API binds to (e.g. ":8443").
	ListenAddr string `yaml:"listen_addr"`
	// TLS material used both for the inbound REST listener (server cert)
	// and as the gRPC client identity toward agents.
	TLS TLS `yaml:"tls"`
	// Agents is the list of reachable agent gRPC endpoints
	// (e.g. "agent-1.internal:9443").
	Agents []string `yaml:"agents"`
}

// Agent holds configuration for the agent gRPC component.
type Agent struct {
	// ListenAddr is the address the gRPC service binds to (e.g. ":9443").
	ListenAddr string `yaml:"listen_addr"`
	// TLS material used by the gRPC listener.
	TLS TLS `yaml:"tls"`
}

// Client holds configuration for the CLI client component.
type Client struct {
	// ServerEndpoint is the REST URL of the server (e.g. "https://server:8443").
	ServerEndpoint string `yaml:"server_endpoint"`
	// TLS material used as the client identity toward the server.
	TLS TLS `yaml:"tls"`
}

// PostgreSQL holds connection parameters for the PostgreSQL database.
type PostgreSQL struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	User    string `yaml:"user"`
	Pass    string `yaml:"pass"`
	DBName  string `yaml:"dbname"`
	SSLMode string `yaml:"sslmode"`
}

// DSN builds a libpq-style connection string.
// Port defaults to 5432 and sslmode defaults to "require" when not set.
func (p PostgreSQL) DSN() string {
	port := p.Port
	if port == 0 {
		port = 5432
	}
	sslmode := p.SSLMode
	if sslmode == "" {
		sslmode = "require"
	}
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.Host, port, p.User, p.Pass, p.DBName, sslmode,
	)
}

// Redis holds connection parameters for the Redis instance.
type Redis struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
	DB   int    `yaml:"db"`
}

// Load reads a YAML configuration file into out.
func Load(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %q: %w", path, err)
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse config %q: %w", path, err)
	}
	return nil
}
