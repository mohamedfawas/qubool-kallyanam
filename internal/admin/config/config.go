package config

import (
	"fmt"
	"os"

	sharedConfig "github.com/mohamedfawas/qubool-kallyanam/pkg/config"
)

// Config holds the configuration for the admin service
type Config struct {
	Common sharedConfig.CommonConfig `mapstructure:"common"`
	Server ServerConfig              `mapstructure:"server"`
	Admin  AdminConfig               `mapstructure:"admin"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

// AdminConfig holds admin service specific configuration
type AdminConfig struct {
	SessionTimeout     int    `mapstructure:"session_timeout"`      // in minutes
	DashboardCacheTime int    `mapstructure:"dashboard_cache_time"` // in seconds
	AdminEmailDomain   string `mapstructure:"admin_email_domain"`
}

// DefaultConfig returns a default configuration for the admin service
func DefaultConfig() *Config {
	return &Config{
		Common: sharedConfig.DefaultCommonConfig(),
		Server: ServerConfig{
			Port: 8084,
			Host: "0.0.0.0",
		},
		Admin: AdminConfig{
			SessionTimeout:     60,  // 1 hour
			DashboardCacheTime: 300, // 5 minutes
			AdminEmailDomain:   "quboolkallyanam.com",
		},
	}
}

// Load loads the configuration from files and environment variables
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Get the config path from environment variable or use the default "."
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "."
	}

	v, err := sharedConfig.LoadConfig(configPath, "admin")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}
