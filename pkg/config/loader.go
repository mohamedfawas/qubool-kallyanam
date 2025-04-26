package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Loader handles loading configuration from various sources (files, environment variables)
type Loader struct {
	v         *viper.Viper // viper instance
	envPrefix string       // Prefix used for environment variable names (e.g., "APP_")
}

// NewLoader creates a new configuration loader with given settings
// Example usage:
// loader := NewLoader("APP", "./", "/etc/myapp")
func NewLoader(envPrefix string, configPaths ...string) *Loader {
	v := viper.New() // Create new Viper instance

	// Set paths to search for config files
	for _, path := range configPaths {
		v.AddConfigPath(path) // Adds a directory to search for config files
	}

	// Setup environment variables
	v.SetEnvPrefix(envPrefix) // Sets prefix for env vars (e.g., "APP_" becomes APP_PORT)
	v.AutomaticEnv()          // Automatically use environment variables

	// Converts config keys to environment-friendly format (e.g., "app.port" → "APP_PORT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // Replaces "." in env vars with "_"

	return &Loader{
		v:         v,
		envPrefix: envPrefix,
	}
}

// LoadConfig loads configuration from file into the provided config struct
func (l *Loader) LoadConfig(filename string, config interface{}) error {
	// Set config name (removes .yaml extension if present)
	// Example: "config.yaml" becomes "config"
	l.v.SetConfigName(strings.TrimSuffix(filename, ".yaml"))

	// Explicitly set config type to YAML
	l.v.SetConfigType("yaml") // Ensures correct parsing even without file extension

	// Try to read configuration from file
	if err := l.v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Unmarshal config data into the provided struct
	// Example: Populates MyConfig.Port from config file's "port" field
	if err := l.v.Unmarshal(config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// GetViper returns the underlying Viper instance for advanced use cases
// Example: Accessing raw values
// value := loader.GetViper().GetString("app.port")
func (l *Loader) GetViper() *viper.Viper {
	return l.v
}
