package config

import (
	"fmt"
	"os"
	"time"

	sharedConfig "github.com/mohamedfawas/qubool-kallyanam/pkg/config"
)

// Config holds the configuration for the gateway service
type Config struct {
	Common           sharedConfig.CommonConfig    `mapstructure:"common"`
	Server           ServerConfig                 `mapstructure:"server"`
	Gateway          GatewayConfig                `mapstructure:"gateway"`
	ServiceURLs      ServiceURLsConfig            `mapstructure:"service_urls"`
	Database         DatabaseConfig               `mapstructure:"database"`
	Messaging        MessagingConfig              `mapstructure:"messaging"`
	Telemetry        sharedConfig.TelemetryConfig `mapstructure:"telemetry"`
	ServiceDiscovery ServiceDiscovery             `mapstructure:"service_discovery"`
}

// ServiceDiscovery holds configuration for service discovery
type ServiceDiscovery struct {
	RefreshInterval string                   `mapstructure:"refresh_interval"`
	Services        map[string]ServiceConfig `mapstructure:"services"`
}

// ServiceConfig holds information about a service
type ServiceConfig struct {
	Host         string   `mapstructure:"host"`
	Port         int      `mapstructure:"port"`
	Type         string   `mapstructure:"type"`
	Routes       []string `mapstructure:"routes"`
	HealthCheck  string   `mapstructure:"health_check"`
	Dependencies []string `mapstructure:"dependencies"`
}

// MessagingConfig holds messaging service configurations
type MessagingConfig struct {
	RabbitMQ sharedConfig.RabbitMQConfig `mapstructure:"rabbitmq"`
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

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Redis RedisConfig `mapstructure:"redis"`
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
		Database: DatabaseConfig{
			Redis: RedisConfig{
				Host:     "redis",
				Port:     6379,
				Password: "",
				DB:       0,
				MaxConns: 10,
				MinIdle:  5,
				Timeout:  2 * time.Second,
			},
		},
		Messaging: MessagingConfig{
			RabbitMQ: sharedConfig.DefaultRabbitMQConfig(),
		},
		Telemetry: sharedConfig.DefaultTelemetryConfig(),
		ServiceDiscovery: ServiceDiscovery{
			RefreshInterval: "30s",
			Services: map[string]ServiceConfig{
				"auth": {
					Host:        "qubool-auth",
					Port:        8081,
					Type:        "http",
					Routes:      []string{"/api/auth/*"},
					HealthCheck: "/health",
				},
				"user": {
					Host:        "qubool-user",
					Port:        8082,
					Type:        "http",
					Routes:      []string{"/api/users/*"},
					HealthCheck: "/health",
				},
				"chat": {
					Host:        "qubool-chat",
					Port:        8083,
					Type:        "http",
					Routes:      []string{"/api/chat/*"},
					HealthCheck: "/health",
				},
				"admin": {
					Host:        "qubool-admin",
					Port:        8084,
					Type:        "http",
					Routes:      []string{"/api/admin/*"},
					HealthCheck: "/health",
				},
			},
		},
	}
}

// Load loads the configuration from files and environment variables
// Load loads the configuration from files and environment variables
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Get the config path from environment variable or use the default "."
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "."
	}

	v, err := sharedConfig.LoadConfig(configPath, "gateway")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}
