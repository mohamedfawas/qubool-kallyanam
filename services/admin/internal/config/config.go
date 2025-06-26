package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	GRPC     GRPCConfig     `mapstructure:"grpc"`
	Services ServicesConfig `mapstructure:"services"`
}

type GRPCConfig struct {
	Port int `mapstructure:"port"`
}

// Only auth and user services needed
type ServicesConfig struct {
	Auth ServiceConfig `mapstructure:"auth"`
	User ServiceConfig `mapstructure:"user"`
}

type ServiceConfig struct {
	Address string `mapstructure:"address"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Environment variable support
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %w", err)
	}

	// Validate required fields
	if config.GRPC.Port == 0 {
		config.GRPC.Port = 50052 // Default admin service port
	}

	if config.Services.Auth.Address == "" {
		config.Services.Auth.Address = os.Getenv("AUTH_SERVICE_ADDRESS")
		if config.Services.Auth.Address == "" {
			return nil, fmt.Errorf("auth service address is required")
		}
	}

	if config.Services.User.Address == "" {
		config.Services.User.Address = os.Getenv("USER_SERVICE_ADDRESS")
		if config.Services.User.Address == "" {
			return nil, fmt.Errorf("user service address is required")
		}
	}

	return &config, nil
}
