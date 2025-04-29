package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Service  ServiceConfig  `yaml:"service"`
	Server   ServerConfig   `yaml:"server"`
	Services ServicesConfig `yaml:"services"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// ServiceConfig contains service metadata
type ServiceConfig struct {
	Name        string `yaml:"name"`
	Environment string `yaml:"environment"`
	Version     string `yaml:"version"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	Timeout int    `yaml:"timeout"`
}

// ServicesConfig contains endpoints for microservices
type ServicesConfig struct {
	AuthHTTP  string `yaml:"auth_http"`
	AuthGRPC  string `yaml:"auth_grpc"`
	UserHTTP  string `yaml:"user_http"`
	UserGRPC  string `yaml:"user_grpc"`
	ChatHTTP  string `yaml:"chat_http"`
	ChatGRPC  string `yaml:"chat_grpc"`
	AdminHTTP string `yaml:"admin_http"`
	AdminGRPC string `yaml:"admin_grpc"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level       string   `yaml:"level"`
	Development bool     `yaml:"development"`
	Encoding    string   `yaml:"encoding"`
	OutputPaths []string `yaml:"output_paths"`
	ServiceName string   `yaml:"service_name"`
}

// Load loads configuration from file
func Load() (*Config, error) {
	configPath := "configs/config.yaml"
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		configPath = path
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
