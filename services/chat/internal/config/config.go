package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	GRPC     GRPCConfig     `mapstructure:"grpc"`
	Database DatabaseConfig `mapstructure:"database"`
}

// GRPCConfig represents gRPC server configuration
type GRPCConfig struct {
	Port int `mapstructure:"port"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	MongoDB MongoDBConfig `mapstructure:"mongodb"`
}

// MongoDBConfig represents MongoDB configuration
type MongoDBConfig struct {
	URI            string `mapstructure:"uri"`
	Database       string `mapstructure:"database"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
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
	if uri := os.Getenv("MONGODB_URI"); uri != "" {
		config.Database.MongoDB.URI = uri
	}

	return &config, nil
}
