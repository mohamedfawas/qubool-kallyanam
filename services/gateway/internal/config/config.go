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
	Auth        AuthConfig     `mapstructure:"auth"`
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
	Auth    ServiceConfig `mapstructure:"auth"`
	User    ServiceConfig `mapstructure:"user"`
	Chat    ServiceConfig `mapstructure:"chat"`
	Payment ServiceConfig `mapstructure:"payment"`
	Admin   ServiceConfig `mapstructure:"admin"`
}

// AuthConfig contains authentication-related configuration
type AuthConfig struct {
	JWT JWTConfig `mapstructure:"jwt"`
}

// JWTConfig contains JWT configuration
type JWTConfig struct {
	SecretKey          string `mapstructure:"secret_key"`
	AccessTokenMinutes int    `mapstructure:"access_token_minutes"`
	RefreshTokenDays   int    `mapstructure:"refresh_token_days"`
	Issuer             string `mapstructure:"issuer"`
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

	if userAddr := os.Getenv("USER_SERVICE_ADDRESS"); userAddr != "" {
		config.Services.User.Address = userAddr
	}

	if chatAddr := os.Getenv("CHAT_SERVICE_ADDRESS"); chatAddr != "" { // Add this block
		config.Services.Chat.Address = chatAddr
	}

	if paymentAddr := os.Getenv("PAYMENT_SERVICE_ADDRESS"); paymentAddr != "" {
		config.Services.Payment.Address = paymentAddr
	}

	if adminAddr := os.Getenv("ADMIN_SERVICE_ADDRESS"); adminAddr != "" {
		config.Services.Admin.Address = adminAddr
	}

	// JWT environment variables
	if secretKey := os.Getenv("JWT_SECRET_KEY"); secretKey != "" {
		config.Auth.JWT.SecretKey = secretKey
	}

	return &config, nil
}
