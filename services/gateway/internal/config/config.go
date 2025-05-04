package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Environment string         `mapstructure:"environment"`
	HTTP        HTTPConfig     `mapstructure:"http"`
	Services    ServicesConfig `mapstructure:"services"`
}

// HTTPConfig represents HTTP server configuration
type HTTPConfig struct {
	Port             int `mapstructure:"port"`
	ReadTimeoutSecs  int `mapstructure:"read_timeout_secs"`
	WriteTimeoutSecs int `mapstructure:"write_timeout_secs"`
	IdleTimeoutSecs  int `mapstructure:"idle_timeout_secs"`
}

// ServicesConfig holds addresses of all services
type ServicesConfig struct {
	Auth ServiceConfig `mapstructure:"auth"`
	User ServiceConfig `mapstructure:"user"`
}

// ServiceConfig represents a service configuration
type ServiceConfig struct {
	Address string `mapstructure:"address"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Override with environment variables if they exist
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		config.Environment = env
	}
	if authAddr := os.Getenv("AUTH_SERVICE_ADDRESS"); authAddr != "" {
		config.Services.Auth.Address = authAddr
	}

	return &config, nil
}
