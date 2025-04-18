package config

import (
	"fmt"

	sharedConfig "github.com/mohamedfawas/qubool-kallyanam/pkg/config"
)

// Config holds the configuration for the user service
type Config struct {
	Common sharedConfig.CommonConfig `mapstructure:"common"`
	Server ServerConfig              `mapstructure:"server"`
	User   UserConfig                `mapstructure:"user"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

// UserConfig holds user service specific configuration
type UserConfig struct {
	ProfilePhotoMaxSize int64  `mapstructure:"profile_photo_max_size"` // in bytes
	DefaultAvatarURL    string `mapstructure:"default_avatar_url"`
}

// DefaultConfig returns a default configuration for the user service
func DefaultConfig() *Config {
	return &Config{
		Common: sharedConfig.DefaultCommonConfig(),
		Server: ServerConfig{
			Port: 8082,
			Host: "0.0.0.0",
		},
		User: UserConfig{
			ProfilePhotoMaxSize: 5 * 1024 * 1024, // 5MB
			DefaultAvatarURL:    "/static/default-avatar.png",
		},
	}
}

// Load loads the configuration from files and environment variables
func Load() (*Config, error) {
	cfg := DefaultConfig()

	v, err := sharedConfig.LoadConfig(".", "user")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}
