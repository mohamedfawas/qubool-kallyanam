package config

import (
	"fmt"
	"os"
	"time"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/mongodb"
	"github.com/spf13/viper"
)

type Config struct {
	GRPC     GRPCConfig     `mapstructure:"grpc"`
	Database DatabaseConfig `mapstructure:"database"`
	Auth     AuthConfig     `mapstructure:"auth"`
}

type GRPCConfig struct {
	Port int `mapstructure:"port"`
}

type DatabaseConfig struct {
	MongoDB MongoDBConfig `mapstructure:"mongodb"`
}

type MongoDBConfig struct {
	URI            string `mapstructure:"uri"`
	Database       string `mapstructure:"database"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
}

type AuthConfig struct {
	JWT JWTConfig `mapstructure:"jwt"`
}

type JWTConfig struct {
	SecretKey string `mapstructure:"secret_key"`
	Issuer    string `mapstructure:"issuer"`
}

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

	// Environment variable overrides
	if mongoURI := os.Getenv("MONGODB_URI"); mongoURI != "" {
		config.Database.MongoDB.URI = mongoURI
	} else if mongoHost := os.Getenv("MONGODB_HOST"); mongoHost != "" {
		// Build URI with authentication for Docker
		mongoUser := os.Getenv("MONGODB_USER")
		mongoPassword := os.Getenv("MONGODB_PASSWORD")

		if mongoUser != "" && mongoPassword != "" {
			config.Database.MongoDB.URI = fmt.Sprintf("mongodb://%s:%s@%s:27017/%s?authSource=admin",
				mongoUser, mongoPassword, mongoHost, config.Database.MongoDB.Database)
		} else {
			// Default Docker credentials if not specified
			config.Database.MongoDB.URI = fmt.Sprintf("mongodb://admin:password@%s:27017/%s?authSource=admin",
				mongoHost, config.Database.MongoDB.Database)
		}
	}
	if jwtSecret := os.Getenv("JWT_SECRET_KEY"); jwtSecret != "" {
		config.Auth.JWT.SecretKey = jwtSecret
	}

	return &config, nil
}

// GetMongoDBConfig returns MongoDB config compatible with pkg/database/mongodb
func (c *Config) GetMongoDBConfig() *mongodb.Config {
	return &mongodb.Config{
		URI:      c.Database.MongoDB.URI,
		Database: c.Database.MongoDB.Database,
		Timeout:  time.Duration(c.Database.MongoDB.TimeoutSeconds) * time.Second,
	}
}
