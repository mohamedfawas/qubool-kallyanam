// services/auth/internal/config/config.go
package config

import (
	"github.com/mohamedfawas/qubool-kallyanam/pkg/config"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/redis"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/http/server"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// Config represents the auth service configuration
type Config struct {
	// Service information
	Service struct {
		Name        string `yaml:"name"`
		Environment string `yaml:"environment"`
		Version     string `yaml:"version"`
	} `yaml:"service"`

	// Server configuration
	Server server.Config `yaml:"server"`

	// Database connections
	Postgres postgres.Config `yaml:"postgres"`
	Redis    redis.Config    `yaml:"redis"`

	// Logging configuration
	Logging logging.Config `yaml:"logging"`
}

// Load loads the configuration from the specified file
func Load(configPath string) (*Config, error) {
	loader := config.NewLoader("AUTH", ".", "./configs")

	var cfg Config
	if err := loader.LoadConfig(configPath, &cfg); err != nil {
		return nil, err
	}

	// Set defaults if not specified
	if cfg.Service.Name == "" {
		cfg.Service.Name = "auth-service"
	}

	return &cfg, nil
}
