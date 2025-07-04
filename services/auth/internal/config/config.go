package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/viper"
)

type Config struct {
	GRPC         GRPCConfig         `mapstructure:"grpc"`
	Database     DatabaseConfig     `mapstructure:"database"`
	Email        EmailConfig        `mapstructure:"email"`
	Auth         AuthConfig         `mapstructure:"auth"`
	Admin        AdminConfig        `mapstructure:"admin"`
	RabbitMQ     RabbitMQConfig     `mapstructure:"rabbitmq"`
	OTP          OTPConfig          `mapstructure:"otp"`
	Registration RegistrationConfig `mapstructure:"registration"`
	Security     SecurityConfig     `mapstructure:"security"`
}

type GRPCConfig struct {
	Port int `mapstructure:"port"`
}

type DatabaseConfig struct {
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis    RedisConfig    `mapstructure:"redis"`
}

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type EmailConfig struct {
	SMTPHost  string `mapstructure:"smtp_host"`
	SMTPPort  int    `mapstructure:"smtp_port"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
	FromEmail string `mapstructure:"from"`
	FromName  string `mapstructure:"from_name"`
}

type AuthConfig struct {
	JWT JWTConfig `mapstructure:"jwt"`
}

type JWTConfig struct {
	SecretKey          string `mapstructure:"secret_key"`
	AccessTokenMinutes int    `mapstructure:"access_token_minutes"`
	RefreshTokenDays   int    `mapstructure:"refresh_token_days"`
	Issuer             string `mapstructure:"issuer"`
}

type AdminConfig struct {
	DefaultEmail    string `mapstructure:"default_email"`
	DefaultPassword string `mapstructure:"default_password"`
}

type RabbitMQConfig struct {
	DSN          string `mapstructure:"dsn"`
	ExchangeName string `mapstructure:"exchange_name"`
}

type OTPConfig struct {
	Length        int `mapstructure:"length"`
	ExpiryMinutes int `mapstructure:"expiry_minutes"`
}

type RegistrationConfig struct {
	PendingExpiryHours int `mapstructure:"pending_expiry_hours"`
}

type SecurityConfig struct {
	PasswordMinLength int `mapstructure:"password_min_length"`
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

	if host := os.Getenv("DB_HOST"); host != "" {
		config.Database.Postgres.Host = host
	}
	if host := os.Getenv("REDIS_HOST"); host != "" {
		config.Database.Redis.Host = host
	}

	if smtpHost := os.Getenv("SMTP_HOST"); smtpHost != "" {
		config.Email.SMTPHost = smtpHost
	}
	if smtpPort := os.Getenv("SMTP_PORT"); smtpPort != "" {
		port, _ := strconv.Atoi(smtpPort)
		config.Email.SMTPPort = port
	}
	if smtpUsername := os.Getenv("SMTP_USERNAME"); smtpUsername != "" {
		config.Email.Username = smtpUsername
	}
	if smtpPassword := os.Getenv("SMTP_PASSWORD"); smtpPassword != "" {
		config.Email.Password = smtpPassword
	}

	if adminEmail := os.Getenv("ADMIN_DEFAULT_EMAIL"); adminEmail != "" {
		config.Admin.DefaultEmail = adminEmail
	}
	if adminPassword := os.Getenv("ADMIN_DEFAULT_PASSWORD"); adminPassword != "" {
		config.Admin.DefaultPassword = adminPassword
	}

	return &config, nil
}
