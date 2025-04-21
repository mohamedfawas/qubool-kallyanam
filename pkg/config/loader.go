package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Loader handles loading configuration from various sources
type Loader struct {
	v         *viper.Viper
	envPrefix string
}

// NewLoader creates a new configuration loader
func NewLoader(envPrefix string, configPaths ...string) *Loader {
	v := viper.New()

	// Set paths to search for config files
	for _, path := range configPaths {
		v.AddConfigPath(path)
	}

	// Setup environment variables
	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	return &Loader{
		v:         v,
		envPrefix: envPrefix,
	}
}

// LoadConfig loads configuration from file into the provided config struct
func (l *Loader) LoadConfig(filename string, config interface{}) error {
	l.v.SetConfigName(strings.TrimSuffix(filename, ".yaml"))
	l.v.SetConfigType("yaml")

	if err := l.v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	if err := l.v.Unmarshal(config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// GetViper returns the underlying Viper instance
func (l *Loader) GetViper() *viper.Viper {
	return l.v
}
