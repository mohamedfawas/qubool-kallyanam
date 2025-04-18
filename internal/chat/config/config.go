package config

import (
	"fmt"

	sharedConfig "github.com/mohamedfawas/qubool-kallyanam/pkg/config"
)

// Config holds the configuration for the chat service
type Config struct {
	Common sharedConfig.CommonConfig `mapstructure:"common"`
	Server ServerConfig              `mapstructure:"server"`
	Chat   ChatConfig                `mapstructure:"chat"`
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
	}
}

// Load loads the configuration from files and environment variables
func Load() (*Config, error) {
	cfg := DefaultConfig()

	v, err := sharedConfig.LoadConfig(".", "chat")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}
