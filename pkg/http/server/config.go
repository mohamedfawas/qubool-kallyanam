package server

import (
	"time"
)

// Config holds server configuration
type Config struct {
	// Host to bind to (default: "")
	Host string `yaml:"host" json:"host"`

	// Port to listen on (default: 8080)
	Port int `yaml:"port" json:"port"`

	// Timeouts
	Timeout struct {
		// Read timeout (default: 5s)
		Read time.Duration `yaml:"read" json:"read"`

		// Write timeout (default: 10s)
		Write time.Duration `yaml:"write" json:"write"`

		// Idle timeout (default: 120s)
		Idle time.Duration `yaml:"idle" json:"idle"`

		// Shutdown timeout (default: 10s)
		Shutdown time.Duration `yaml:"shutdown" json:"shutdown"`
	} `yaml:"timeout" json:"timeout"`

	// TLS configuration
	TLS struct {
		// Whether to enable TLS (default: false)
		Enabled bool `yaml:"enabled" json:"enabled"`

		// TLS certificate file path
		CertFile string `yaml:"cert_file" json:"cert_file"`

		// TLS key file path
		KeyFile string `yaml:"key_file" json:"key_file"`
	} `yaml:"tls" json:"tls"`
}

// DefaultConfig returns a default server configuration
func DefaultConfig() Config {
	config := Config{
		Host: "",
		Port: 8080,
	}

	// Default timeouts
	config.Timeout.Read = 5 * time.Second
	config.Timeout.Write = 10 * time.Second
	config.Timeout.Idle = 120 * time.Second
	config.Timeout.Shutdown = 10 * time.Second

	// Default TLS settings
	config.TLS.Enabled = false

	return config
}
