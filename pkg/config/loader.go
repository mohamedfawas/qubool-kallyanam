package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// LoadConfig loads configuration from .env files and environment variables
// configPath is the path where the config files are located
func LoadConfig(configPath string, configName string) (*viper.Viper, error) {
	v := viper.New()

	// Set defaults for config
	v.SetConfigName(configName)
	v.SetConfigType("env")
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
