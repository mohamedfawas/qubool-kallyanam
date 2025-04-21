package config

import (
	"sync"
)

// Manager handles access to configuration values
type Manager struct {
	mu     sync.RWMutex
	config interface{}
	loader *Loader
}

// NewManager creates a new configuration manager
func NewManager(config interface{}, loader *Loader) *Manager {
	return &Manager{
		config: config,
		loader: loader,
	}
}

// Load loads the configuration from the specified file
func (m *Manager) Load(filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.loader.LoadConfig(filename, m.config)
}

// Get returns the configuration
func (m *Manager) Get() interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// Reload reloads the configuration
func (m *Manager) Reload(filename string) error {
	return m.Load(filename)
}
