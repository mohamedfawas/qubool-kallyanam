package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	GRPC     GRPCConfig     `mapstructure:"grpc"`
	HTTP     HTTPConfig     `mapstructure:"http"`
	Database DatabaseConfig `mapstructure:"database"`
	Razorpay RazorpayConfig `mapstructure:"razorpay"`
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
	Gateway  GatewayConfig  `mapstructure:"gateway"`
	Plans    PlansConfig    `mapstructure:"plans"`
}

type GRPCConfig struct {
	Port int `mapstructure:"port"`
}

type HTTPConfig struct {
	Port             int `mapstructure:"port"`
	ReadTimeoutSecs  int `mapstructure:"read_timeout_secs"`
	WriteTimeoutSecs int `mapstructure:"write_timeout_secs"`
	IdleTimeoutSecs  int `mapstructure:"idle_timeout_secs"`
}

type DatabaseConfig struct {
	Postgres PostgresConfig `mapstructure:"postgres"`
}

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RazorpayConfig struct {
	KeyID     string `mapstructure:"key_id"`
	KeySecret string `mapstructure:"key_secret"`
}

type RabbitMQConfig struct {
	DSN          string `mapstructure:"dsn"`
	ExchangeName string `mapstructure:"exchange_name"`
}

type GatewayConfig struct {
	Address string `mapstructure:"address"`
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
	if host := os.Getenv("DB_HOST"); host != "" {
		config.Database.Postgres.Host = host
	}
	if keyID := os.Getenv("RAZORPAY_KEY_ID"); keyID != "" {
		config.Razorpay.KeyID = keyID
	}
	if keySecret := os.Getenv("RAZORPAY_KEY_SECRET"); keySecret != "" {
		config.Razorpay.KeySecret = keySecret
	}
	if gatewayAddr := os.Getenv("GATEWAY_ADDRESS"); gatewayAddr != "" {
		config.Gateway.Address = gatewayAddr
	}

	// Set default plans if none configured
	if len(config.Plans.Available) == 0 {
		config.Plans = *GetDefaultPlansConfig()
	}

	return &config, nil
}
