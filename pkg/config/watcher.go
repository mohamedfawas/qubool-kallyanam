package config

// use in development only, not in production
// less priority , so not focusing on this now.

// import (
// 	"fmt"
// 	"sync"

// 	"github.com/fsnotify/fsnotify"
// )

// // Watcher watches for changes in configuration files
// type Watcher struct {
// 	manager   *Manager
// 	loader    *Loader
// 	handlers  []func()
// 	mu        sync.Mutex
// 	isRunning bool
// }

// // NewWatcher creates a new configuration watcher
// func NewWatcher(manager *Manager, loader *Loader) *Watcher {
// 	return &Watcher{
// 		manager:  manager,
// 		loader:   loader,
// 		handlers: make([]func(), 0),
// 	}
// }

// // Start begins watching for configuration changes
// func (w *Watcher) Start(configName string) error {
// 	w.mu.Lock()
// 	defer w.mu.Unlock()

// 	if w.isRunning {
// 		return nil
// 	}

// 	// Get Viper instance from loader
// 	v := w.loader.GetViper()

// 	// Setup Viper to watch for changes
// 	v.WatchConfig()

// 	// Register callback for config changes
// 	v.OnConfigChange(func(e fsnotify.Event) {
// 		// Reload the configuration
// 		if err := w.manager.Reload(configName); err != nil {
// 			fmt.Printf("Error reloading config: %v\n", err)
// 			return
// 		}

// 		// Notify all handlers
// 		w.notifyHandlers()
// 	})

// 	w.isRunning = true
// 	return nil
// }

// // RegisterChangeHandler registers a handler to be called when config changes
// func (w *Watcher) RegisterChangeHandler(handler func()) {
// 	w.mu.Lock()
// 	defer w.mu.Unlock()
// 	w.handlers = append(w.handlers, handler)
// }

// // notifyHandlers calls all registered handlers
// func (w *Watcher) notifyHandlers() {
// 	w.mu.Lock()
// 	defer w.mu.Unlock()

// 	for _, handler := range w.handlers {
// 		handler()
// 	}
// }
