package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	GRPC        GRPCConfig        `mapstructure:"grpc"`
	Database    DatabaseConfig    `mapstructure:"database"`
	RabbitMQ    RabbitMQConfig    `mapstructure:"rabbitmq"`
	Email       EmailConfig       `mapstructure:"email"`
	Auth        AuthConfig        `mapstructure:"auth"`
	Storage     StorageConfig     `mapstructure:"storage"`
	Matchmaking MatchmakingConfig `mapstructure:"matchmaking"`
}

type EmailConfig struct {
	SMTPHost  string `mapstructure:"smtp_host"`
	SMTPPort  int    `mapstructure:"smtp_port"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
	FromEmail string `mapstructure:"from"`
	FromName  string `mapstructure:"from_name"`
}

type StorageConfig struct {
	S3 S3Config `mapstructure:"s3"`
}

type MatchmakingConfig struct {
	Weights     MatchWeights      `mapstructure:"weights"`
	HardFilters HardFiltersConfig `mapstructure:"hard_filters"`
}

type MatchWeights struct {
	Community  float64 `mapstructure:"community"`
	Profession float64 `mapstructure:"profession"`
	Location   float64 `mapstructure:"location"`
	Recency    float64 `mapstructure:"recency"`
}

type HardFiltersConfig struct {
	Enabled                         bool `mapstructure:"enabled"`
	ApplyAgeFilter                  bool `mapstructure:"apply_age_filter"`
	ApplyHeightFilter               bool `mapstructure:"apply_height_filter"`
	ApplyMaritalStatusFilter        bool `mapstructure:"apply_marital_status_filter"`
	ApplyPhysicallyChallengedFilter bool `mapstructure:"apply_physically_challenged_filter"`
	ApplyEducationFilter            bool `mapstructure:"apply_education_filter"`
}

type S3Config struct {
	Endpoint        string `mapstructure:"endpoint"`
	Region          string `mapstructure:"region"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	BucketName      string `mapstructure:"bucket_name"`
	UseSSL          bool   `mapstructure:"use_ssl"`
}

type AuthConfig struct {
	JWT JWTConfig `mapstructure:"jwt"`
}

type JWTConfig struct {
	SecretKey string `mapstructure:"secret_key"`
	Issuer    string `mapstructure:"issuer"`
}

type RabbitMQConfig struct {
	DSN          string `mapstructure:"dsn"`
	ExchangeName string `mapstructure:"exchange_name"`
}

// GRPCConfig represents gRPC server configuration
type GRPCConfig struct {
	Port int `mapstructure:"port"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis    RedisConfig    `mapstructure:"redis"`
}

// PostgresConfig represents PostgreSQL configuration
type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
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
	if host := os.Getenv("DB_HOST"); host != "" {
		config.Database.Postgres.Host = host
	}
	if host := os.Getenv("REDIS_HOST"); host != "" {
		config.Database.Redis.Host = host
	}

	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		config.Auth.JWT.SecretKey = jwtSecret
	}

	// S3/MinIO environment variables
	if endpoint := os.Getenv("S3_ENDPOINT"); endpoint != "" {
		config.Storage.S3.Endpoint = endpoint
	}
	if region := os.Getenv("S3_REGION"); region != "" {
		config.Storage.S3.Region = region
	}
	if accessKey := os.Getenv("S3_ACCESS_KEY"); accessKey != "" {
		config.Storage.S3.AccessKeyID = accessKey
	}
	if secretKey := os.Getenv("S3_SECRET_KEY"); secretKey != "" {
		config.Storage.S3.SecretAccessKey = secretKey
	}
	if bucketName := os.Getenv("S3_BUCKET_NAME"); bucketName != "" {
		config.Storage.S3.BucketName = bucketName
	}
	if useSSL := os.Getenv("S3_USE_SSL"); useSSL == "true" {
		config.Storage.S3.UseSSL = true
	}

	if smtpHost := os.Getenv("SMTP_HOST"); smtpHost != "" {
		config.Email.SMTPHost = smtpHost
	}
	if smtpPort := os.Getenv("SMTP_PORT"); smtpPort != "" {
		if port, err := strconv.Atoi(smtpPort); err == nil {
			config.Email.SMTPPort = port
		}
	}
	if emailPass := os.Getenv("EMAIL_PASSWORD"); emailPass != "" {
		config.Email.Password = emailPass
	}

	return &config, nil
}
