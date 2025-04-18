package config

import (
	"fmt"

	sharedConfig "github.com/mohamedfawas/qubool-kallyanam/pkg/config"
)

// Config holds the configuration for the auth service
type Config struct {
	Common sharedConfig.CommonConfig `mapstructure:"common"`
	Server ServerConfig              `mapstructure:"server"`
	Auth   AuthConfig                `mapstructure:"auth"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

// AuthConfig holds authentication-specific configuration
type AuthConfig struct {
	JWTSecret     string `mapstructure:"jwt_secret"`
	TokenLifetime int    `mapstructure:"token_lifetime"` // in minutes
}

// DefaultConfig returns a default configuration for the auth service
func DefaultConfig() *Config {
	return &Config{
		Common: sharedConfig.DefaultCommonConfig(),
		Server: ServerConfig{
			Port: 8081,
			Host: "0.0.0.0",
		},
		Auth: AuthConfig{
			JWTSecret:     "default-jwt-secret-change-in-production",
			TokenLifetime: 60, // 1 hour
		},
	}
}

// Load loads the configuration from files and environment variables
func Load() (*Config, error) {
	cfg := DefaultConfig()

	v, err := sharedConfig.LoadConfig(".", "auth")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate configuration
	if cfg.Auth.JWTSecret == "default-jwt-secret-change-in-production" &&
		cfg.Common.Environment == "production" {
		return nil, fmt.Errorf("default JWT secret cannot be used in production")
	}

	return cfg, nil
}
