// pkg/service/config.go
package service

import (
	"fmt"
)

// ConfigLoader is a function type that loads configuration
type ConfigLoader func() (interface{}, error)

// ConfigProvider is an interface for accessing common config properties
type ConfigProvider interface {
	// GetEnvironment returns the environment (development, production, etc)
	GetEnvironment() string

	// GetDebug returns the debug flag
	GetDebug() bool

	// GetServerHost returns the server host
	GetServerHost() string

	// GetServerPort returns the server port
	GetServerPort() int
}

// SetGinMode sets the Gin mode based on the environment
func SetGinMode(environment string) {
	// This function would be called during service initialization
	// to set the appropriate Gin mode based on the environment

	// Implementation will depend on the gin package
	fmt.Println("Setting Gin mode for environment:", environment)
}

// CommonConfigAdapter adapts various config types to the ConfigProvider interface
// This is a helper to work with different config structures
type CommonConfigAdapter struct {
	Config interface{}
}

// GetEnvironment returns the environment from the config
func (a *CommonConfigAdapter) GetEnvironment() string {
	// Implementation depends on your config structure
	// This is a placeholder - you would implement reflection or type assertions
	// to extract the environment from your config
	return "development"
}

// GetDebug returns the debug flag from the config
func (a *CommonConfigAdapter) GetDebug() bool {
	// Implementation depends on your config structure
	return true
}

// GetServerHost returns the server host from the config
func (a *CommonConfigAdapter) GetServerHost() string {
	// Implementation depends on your config structure
	return "0.0.0.0"
}

// GetServerPort returns the server port from the config
func (a *CommonConfigAdapter) GetServerPort() int {
	// Implementation depends on your config structure
	return 8080
}
