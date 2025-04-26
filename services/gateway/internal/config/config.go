// services/gateway/internal/config/config.go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	pkgconfig "github.com/mohamedfawas/qubool-kallyanam/pkg/config"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// Config represents the API gateway configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Services ServicesConfig `yaml:"services"`
	Logging  logging.Config `yaml:"logging"`
}

// ServerConfig holds the HTTP server configuration
type ServerConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	Timeout int    `yaml:"timeout_seconds"`
}

// ServicesConfig holds the configuration for downstream services
type ServicesConfig struct {
	User  string `yaml:"user"`
	Auth  string `yaml:"auth"`
	Chat  string `yaml:"chat"`
	Admin string `yaml:"admin"`
}

// Load loads the configuration from a file
func Load(path string) (*Config, error) {
	var cfg Config

	// Set default values
	cfg.Server.Host = "0.0.0.0"
	cfg.Server.Port = 8080
	cfg.Server.Timeout = 30

	// Default service addresses
	cfg.Services.User = "user-service:50051"
	cfg.Services.Auth = "auth-service:50052"
	cfg.Services.Chat = "chat-service:50053"
	cfg.Services.Admin = "admin-service:50054"

	// Load configuration file
	if _, err := os.Stat(path); err == nil {
		// Create config loader
		loader := pkgconfig.NewLoader("GATEWAY", filepath.Dir(path))

		// Load configuration
		if err := loader.LoadConfig(filepath.Base(path), &cfg); err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to access config file: %w", err)
	}

	// Override with environment variables if present
	if host := os.Getenv("SERVER_HOST"); host != "" {
		cfg.Server.Host = host
	}
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if n, err := fmt.Sscanf(port, "%d", &cfg.Server.Port); n != 1 || err != nil {
			return nil, fmt.Errorf("invalid SERVER_PORT: %s", port)
		}
	}

	return &cfg, nil
}
