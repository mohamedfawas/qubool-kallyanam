package config

import (
	"fmt"

	sharedConfig "github.com/mohamedfawas/qubool-kallyanam/pkg/config"
)

// Config holds the configuration for the gateway service
type Config struct {
	Common      sharedConfig.CommonConfig `mapstructure:"common"`
	Server      ServerConfig              `mapstructure:"server"`
	Gateway     GatewayConfig             `mapstructure:"gateway"`
	ServiceURLs ServiceURLsConfig         `mapstructure:"service_urls"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

// GatewayConfig holds gateway-specific configuration
type GatewayConfig struct {
	RequestTimeout          int   `mapstructure:"request_timeout"`  // in seconds
	MaxRequestSize          int64 `mapstructure:"max_request_size"` // in bytes
	EnableCORS              bool  `mapstructure:"enable_cors"`
	RateLimitRequestsPerMin int   `mapstructure:"rate_limit_requests_per_min"`
}

// ServiceURLsConfig holds URLs for the backend services
type ServiceURLsConfig struct {
	AuthService  string `mapstructure:"auth_service"`
	UserService  string `mapstructure:"user_service"`
	ChatService  string `mapstructure:"chat_service"`
	AdminService string `mapstructure:"admin_service"`
}

// DefaultConfig returns a default configuration for the gateway service
func DefaultConfig() *Config {
	return &Config{
		Common: sharedConfig.DefaultCommonConfig(),
		Server: ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Gateway: GatewayConfig{
			RequestTimeout:          30,
			MaxRequestSize:          10 * 1024 * 1024, // 10MB
			EnableCORS:              true,
			RateLimitRequestsPerMin: 60,
		},
		ServiceURLs: ServiceURLsConfig{
			AuthService:  "http://auth:8081",
			UserService:  "http://user:8082",
			ChatService:  "http://chat:8083",
			AdminService: "http://admin:8084",
		},
	}
}

// Load loads the configuration from files and environment variables
func Load() (*Config, error) {
	cfg := DefaultConfig()

	v, err := sharedConfig.LoadConfig(".", "gateway")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}
