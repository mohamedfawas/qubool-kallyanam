package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/firestore"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/mongodb"
	"github.com/spf13/viper"
)

type Config struct {
	GRPC     GRPCConfig     `mapstructure:"grpc"`
	Database DatabaseConfig `mapstructure:"database"`
	Auth     AuthConfig     `mapstructure:"auth"`
}

type GRPCConfig struct {
	Port int `mapstructure:"port"`
}

type DatabaseConfig struct {
	Type      string          `mapstructure:"type"` // "mongodb" or "firestore"
	MongoDB   MongoDBConfig   `mapstructure:"mongodb"`
	Firestore FirestoreConfig `mapstructure:"firestore"`
}

type MongoDBConfig struct {
	URI            string `mapstructure:"uri"`
	Database       string `mapstructure:"database"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
}

type FirestoreConfig struct {
	ProjectID       string `mapstructure:"project_id"`
	CredentialsFile string `mapstructure:"credentials_file"`
	EmulatorHost    string `mapstructure:"emulator_host"`
}

type AuthConfig struct {
	JWT JWTConfig `mapstructure:"jwt"`
}

type JWTConfig struct {
	SecretKey string `mapstructure:"secret_key"`
	Issuer    string `mapstructure:"issuer"`
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

	// Database type selection - default to MongoDB for development
	if dbType := os.Getenv("DB_TYPE"); dbType != "" {
		config.Database.Type = strings.ToLower(dbType)
	} else {
		config.Database.Type = "mongodb" // Default fallback
	}

	// MongoDB environment variable overrides
	if mongoURI := os.Getenv("MONGODB_URI"); mongoURI != "" {
		config.Database.MongoDB.URI = mongoURI
	} else if mongoHost := os.Getenv("MONGODB_HOST"); mongoHost != "" {
		mongoUser := os.Getenv("MONGODB_USER")
		mongoPassword := os.Getenv("MONGODB_PASSWORD")
		mongoDatabase := os.Getenv("MONGODB_DATABASE")

		if mongoDatabase == "" {
			mongoDatabase = config.Database.MongoDB.Database
		}

		if mongoUser != "" && mongoPassword != "" {
			config.Database.MongoDB.URI = fmt.Sprintf("mongodb://%s:%s@%s:27017/%s?authSource=admin",
				mongoUser, mongoPassword, mongoHost, mongoDatabase)
		} else {
			config.Database.MongoDB.URI = fmt.Sprintf("mongodb://admin:password@%s:27017/%s?authSource=admin",
				mongoHost, mongoDatabase)
		}
	}

	// Firestore environment variable overrides
	if projectID := os.Getenv("FIRESTORE_PROJECT_ID"); projectID != "" {
		config.Database.Firestore.ProjectID = projectID
	}
	if credFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credFile != "" {
		config.Database.Firestore.CredentialsFile = credFile
	}
	if emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST"); emulatorHost != "" {
		config.Database.Firestore.EmulatorHost = emulatorHost
	}

	// JWT override
	if jwtSecret := os.Getenv("JWT_SECRET_KEY"); jwtSecret != "" {
		config.Auth.JWT.SecretKey = jwtSecret
	}

	return &config, nil
}

// GetMongoDBConfig returns MongoDB config compatible with pkg/database/mongodb
func (c *Config) GetMongoDBConfig() *mongodb.Config {
	return &mongodb.Config{
		URI:      c.Database.MongoDB.URI,
		Database: c.Database.MongoDB.Database,
		Timeout:  time.Duration(c.Database.MongoDB.TimeoutSeconds) * time.Second,
	}
}

// GetFirestoreConfig returns Firestore config compatible with pkg/database/firestore
func (c *Config) GetFirestoreConfig() *firestore.Config {
	return &firestore.Config{
		ProjectID:       c.Database.Firestore.ProjectID,
		CredentialsFile: c.Database.Firestore.CredentialsFile,
		EmulatorHost:    c.Database.Firestore.EmulatorHost,
	}
}

// GetDatabaseType returns the configured database type
func (c *Config) GetDatabaseType() string {
	return c.Database.Type
}
