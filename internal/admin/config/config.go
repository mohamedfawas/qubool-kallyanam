package config

import (
	"fmt"
	"os"
	"time"

	sharedConfig "github.com/mohamedfawas/qubool-kallyanam/pkg/config"
)

// Config holds the configuration for the admin service
type Config struct {
	Common   sharedConfig.CommonConfig `mapstructure:"common"`
	Server   ServerConfig              `mapstructure:"server"`
	Admin    AdminConfig               `mapstructure:"admin"`
	Database DatabaseConfig            `mapstructure:"database"`
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

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis    RedisConfig    `mapstructure:"redis"`
}

// PostgresConfig holds postgres configuration
type PostgresConfig struct {
	Host     string        `mapstructure:"host"`
	Port     int           `mapstructure:"port"`
	Username string        `mapstructure:"username"`
	Password string        `mapstructure:"password"`
	Database string        `mapstructure:"database"`
	SSLMode  string        `mapstructure:"sslmode"`
	MaxConns int           `mapstructure:"max_conns"`
	MaxIdle  int           `mapstructure:"max_idle"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

// RedisConfig holds redis configuration
type RedisConfig struct {
	Host     string        `mapstructure:"host"`
	Port     int           `mapstructure:"port"`
	Password string        `mapstructure:"password"`
	DB       int           `mapstructure:"db"`
	MaxConns int           `mapstructure:"max_conns"`
	MinIdle  int           `mapstructure:"min_idle"`
	Timeout  time.Duration `mapstructure:"timeout"`
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
		Database: DatabaseConfig{
			Postgres: PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Password: "postgres",
				Database: "qubool_kallyanam",
				SSLMode:  "disable",
				MaxConns: 10,
				MaxIdle:  5,
				Timeout:  5 * time.Second,
			},
			Redis: RedisConfig{
				Host:     "localhost",
				Port:     6379,
				Password: "",
				DB:       0,
				MaxConns: 10,
				MinIdle:  5,
				Timeout:  2 * time.Second,
			},
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
