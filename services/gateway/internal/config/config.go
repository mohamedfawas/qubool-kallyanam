package config

import (
	"os"

	"github.com/gin-gonic/gin"
)

// Config holds all configuration for the gateway service
type Config struct {
	Server   ServerConfig
	Services ServicesConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Address string
	Mode    string
}

// ServicesConfig holds configurations for all services
type ServicesConfig struct {
	Auth  ServiceConfig
	User  ServiceConfig
	Admin ServiceConfig
	Chat  ServiceConfig
}

// ServiceConfig holds configuration for a service
type ServiceConfig struct {
	Address string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Address: getEnv("SERVER_ADDRESS", ":8080"),
			Mode:    getEnv("GIN_MODE", gin.ReleaseMode),
		},
		Services: ServicesConfig{
			Auth: ServiceConfig{
				Address: getEnv("AUTH_SERVICE_ADDRESS", "localhost:50051"),
			},
			User: ServiceConfig{
				Address: getEnv("USER_SERVICE_ADDRESS", "localhost:50052"),
			},
			Admin: ServiceConfig{
				Address: getEnv("ADMIN_SERVICE_ADDRESS", "localhost:50053"),
			},
			Chat: ServiceConfig{
				Address: getEnv("CHAT_SERVICE_ADDRESS", "localhost:50054"),
			},
		},
	}, nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
