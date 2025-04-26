package config

import (
	"sync"
)

// Manager handles access to configuration values with thread-safe operations
type Manager struct {
	mu     sync.RWMutex // Read-Write mutex for concurrent access safety
	config interface{}  // The actual configuration data (usually a struct)
	loader *Loader      // Loader instance to handle configuration loading
}

// NewManager creates a new configuration manager
// Example usage:
// configStruct := MyConfig{}
// manager := NewManager(&configStruct, loader)
func NewManager(config interface{}, loader *Loader) *Manager {
	return &Manager{
		config: config,
		loader: loader,
	}
}

// Load loads the configuration from the specified file
// Example usage:
// err := manager.Load("config.yaml")
// if err != nil { ... }
func (m *Manager) Load(filename string) error {
	// Acquire an exclusive write lock to ensure only one goroutine can update the config at a time
	// Example scenario: Two goroutines trying to load config simultaneously
	// Without this lock, both could overwrite each other's changes
	m.mu.Lock()
	defer m.mu.Unlock() // Ensure lock is released even if an error occurs

	// Delegate loading to the Loader, which:
	// 1. Reads the config file
	// 2. Merges environment variables (if set)
	// 3. Populates the config struct stored in m.config
	// Example 1: Successful load
	//   - config.yaml contains: port: 8080
	//   - After Load("config.yaml"), m.config.(MyConfig).Port == 8080
	// Example 2: File not found
	//   - Returns error: "open config.yaml: no such file or directory"
	// Example 3: Invalid YAML format
	//   - Returns error: "yaml: unmarshal errors:\n  ..."
	return m.loader.LoadConfig(filename, m.config)
}

// Get returns the current configuration safely
// Example usage:
// config := manager.Get().(*MyConfig)
// port := config.Port
func (m *Manager) Get() interface{} {
	// Acquire a read lock to allow safe concurrent reads
	// Multiple goroutines can read simultaneously
	// But blocks any Load()/Reload() operations until unlocked
	m.mu.RLock()

	defer m.mu.RUnlock() // Release read lock after returning

	// Return stored configuration (stored as interface{})
	// Must type-assert to original struct type when using
	// Example: actualConfig := config.(*MyConfig)
	return m.config
}

// Reload reloads the configuration from the specified file
// Example usage:
// err := manager.Reload("new_config.yaml")
// if err != nil { ... }
func (m *Manager) Reload(filename string) error {
	return m.Load(filename)
}
