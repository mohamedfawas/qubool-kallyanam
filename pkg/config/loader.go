package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// LoadConfig loads configuration from .env files and environment variables
// configPath is the path where the config files are located
func LoadConfig(configPath string, configName string) (*viper.Viper, error) {
	v := viper.New()

	// Set defaults for config
	v.SetConfigName(configName)
	v.SetConfigType("yaml") // Changed from "env" to "yaml"
	v.AddConfigPath(configPath)
	v.AddConfigPath(".")

	// Read config from file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// If config file not found, log a warning but continue
		fmt.Printf("Warning: Config file not found, using environment variables only\n")
	}

	// Override with environment variables
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	return v, nil
}

// CommonConfig contains configuration elements common to all services
type CommonConfig struct {
	Environment string `mapstructure:"environment"` // dev, staging, prod
	LogLevel    string `mapstructure:"log_level"`   // debug, info, warn, error
	Debug       bool   `mapstructure:"debug"`       // enable debug mode
}

// DefaultCommonConfig returns default common configuration
func DefaultCommonConfig() CommonConfig {
	return CommonConfig{
		Environment: "development",
		LogLevel:    "info",
		Debug:       true,
	}
}

// RabbitMQConfig holds RabbitMQ connection configuration
type RabbitMQConfig struct {
	Host           string        `mapstructure:"host"`
	Port           int           `mapstructure:"port"`
	Username       string        `mapstructure:"username"`
	Password       string        `mapstructure:"password"`
	VHost          string        `mapstructure:"vhost"`
	Reconnect      bool          `mapstructure:"reconnect"`
	ReconnectDelay time.Duration `mapstructure:"reconnect_delay"`
}

// DefaultRabbitMQConfig returns default RabbitMQ configuration
func DefaultRabbitMQConfig() RabbitMQConfig {
	return RabbitMQConfig{
		Host:           "rabbitmq",
		Port:           5672,
		Username:       "guest",
		Password:       "guest",
		VHost:          "/",
		Reconnect:      true,
		ReconnectDelay: 5 * time.Second,
	}
}

// TelemetryConfig holds configuration for all telemetry components
type TelemetryConfig struct {
	Metrics MetricsConfig `mapstructure:"metrics"`
	Tracing TracingConfig `mapstructure:"tracing"`
	Logging LoggingConfig `mapstructure:"logging"`
}

// MetricsConfig holds configuration for metrics collection
type MetricsConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	ListenAddress string `mapstructure:"listen_address"`
	MetricsPath   string `mapstructure:"metrics_path"`
}

// TracingConfig holds configuration for distributed tracing
type TracingConfig struct {
	Enabled    bool    `mapstructure:"enabled"`
	Endpoint   string  `mapstructure:"endpoint"`
	Insecure   bool    `mapstructure:"insecure"`
	SampleRate float64 `mapstructure:"sample_rate"`
}

// LoggingConfig holds configuration for logging
type LoggingConfig struct {
	Debug bool       `mapstructure:"debug"`
	Loki  LokiConfig `mapstructure:"loki"`
}

// LokiConfig holds configuration for Loki logging
type LokiConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	URL       string `mapstructure:"url"`
	BatchSize int    `mapstructure:"batch_size"`
	Timeout   string `mapstructure:"timeout"`
	TenantID  string `mapstructure:"tenant_id"`
}

// DefaultTelemetryConfig returns default telemetry configuration
func DefaultTelemetryConfig() TelemetryConfig {
	return TelemetryConfig{
		Metrics: MetricsConfig{
			Enabled:       true,
			ListenAddress: ":8090",
			MetricsPath:   "/metrics",
		},
		Tracing: TracingConfig{
			Enabled:    true,
			Endpoint:   "otel-collector:4317",
			Insecure:   true,
			SampleRate: 1.0,
		},
		Logging: LoggingConfig{
			Debug: false,
			Loki: LokiConfig{
				Enabled:   true,
				URL:       "http://loki:3100/loki/api/v1/push",
				BatchSize: 1024 * 1024,
				Timeout:   "10s",
				TenantID:  "",
			},
		},
	}
}
