package config

import (
	"fmt"
	"os"
	"time"

	sharedConfig "github.com/mohamedfawas/qubool-kallyanam/pkg/config"
)

// Config holds the configuration for the chat service
type Config struct {
	Common   sharedConfig.CommonConfig `mapstructure:"common"`
	Server   ServerConfig              `mapstructure:"server"`
	Chat     ChatConfig                `mapstructure:"chat"`
	Database DatabaseConfig            `mapstructure:"database"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

// ChatConfig holds chat service specific configuration
type ChatConfig struct {
	MaxMessageLength    int   `mapstructure:"max_message_length"`
	MessageRateLimit    int   `mapstructure:"message_rate_limit"` // messages per minute
	MessageHistoryLimit int   `mapstructure:"message_history_limit"`
	WebSocketTimeout    int64 `mapstructure:"websocket_timeout"` // in seconds
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	MongoDB MongoDBConfig `mapstructure:"mongodb"`
	Redis   RedisConfig   `mapstructure:"redis"`
}

// MongoDBConfig holds MongoDB configuration
type MongoDBConfig struct {
	URI      string        `mapstructure:"uri"`
	Database string        `mapstructure:"database"`
	Username string        `mapstructure:"username"`
	Password string        `mapstructure:"password"`
	MaxConns uint64        `mapstructure:"max_conns"`
	MinConns uint64        `mapstructure:"min_conns"`
	Timeout  time.Duration `mapstructure:"timeout"`
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

// DefaultConfig returns a default configuration for the chat service
func DefaultConfig() *Config {
	return &Config{
		Common: sharedConfig.DefaultCommonConfig(),
		Server: ServerConfig{
			Port: 8083,
			Host: "0.0.0.0",
		},
		Chat: ChatConfig{
			MaxMessageLength:    1000,
			MessageRateLimit:    60,
			MessageHistoryLimit: 100,
			WebSocketTimeout:    300, // 5 minutes
		},
		Database: DatabaseConfig{
			MongoDB: MongoDBConfig{
				URI:      "mongodb://localhost:27017/qubool_kallyanam",
				Database: "qubool_kallyanam",
				Username: "qubool",
				Password: "qubool123",
				MaxConns: 10,
				MinConns: 5,
				Timeout:  5 * time.Second,
			},
			Redis: RedisConfig{
				Host:     "localhost",
				Port:     6379,
				Password: "",
				DB:       0,
				MaxConns: 10,
				MinIdle:  5,
				Timeout:  2 * time.Second,
			},
		},
	}
}

// Load loads the configuration from files and environment variables
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Get the config path from environment variable or use the default "."
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "."
	}

	v, err := sharedConfig.LoadConfig(configPath, "chat")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}
