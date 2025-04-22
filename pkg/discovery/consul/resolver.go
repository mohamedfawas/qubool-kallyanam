package consul

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/discovery"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// Resolver provides service instance resolution with caching and background refresh
type Resolver struct {
	client              *Client
	logger              logging.Logger
	cache               map[string][]discovery.ServiceInstance
	cacheLock           sync.RWMutex
	refreshInterval     time.Duration
	watchedServices     map[string]discovery.ServiceQuery
	watchedServicesLock sync.RWMutex
	stopCh              chan struct{}
}

// ResolverConfig contains configuration for the resolver
type ResolverConfig struct {
	RefreshInterval time.Duration
}

// NewResolver creates a new service resolver
func NewResolver(client *Client, cfg ResolverConfig, logger logging.Logger) *Resolver {
	if logger == nil {
		logger = logging.Get().Named("consul-resolver")
	}

	if cfg.RefreshInterval <= 0 {
		cfg.RefreshInterval = 30 * time.Second
	}

	resolver := &Resolver{
		client:          client,
		logger:          logger,
		cache:           make(map[string][]discovery.ServiceInstance),
		refreshInterval: cfg.RefreshInterval,
		watchedServices: make(map[string]discovery.ServiceQuery),
		stopCh:          make(chan struct{}),
	}

	// Start background refresh
	go resolver.refreshLoop()

	return resolver
}

// WatchService adds a service to the watch list for background refresh
func (r *Resolver) WatchService(serviceName string, onlyHealthy bool, tags []string) {
	r.watchedServicesLock.Lock()
	defer r.watchedServicesLock.Unlock()

	query := discovery.ServiceQuery{
		Name:        serviceName,
		OnlyHealthy: onlyHealthy,
		Tags:        tags,
		Timeout:     5 * time.Second,
	}

	key := r.queryKey(query)
	r.watchedServices[key] = query

	// Immediately refresh this service
	go func() {
		if _, err := r.refreshService(context.Background(), query); err != nil {
			r.logger.Warn("Failed to perform initial refresh for service",
				logging.String("service", serviceName),
				logging.Error(err),
			)
		}
	}()
}

// UnwatchService removes a service from the watch list
func (r *Resolver) UnwatchService(serviceName string, tags []string) {
	r.watchedServicesLock.Lock()
	defer r.watchedServicesLock.Unlock()

	query := discovery.ServiceQuery{
		Name: serviceName,
		Tags: tags,
	}

	key := r.queryKey(query)
	delete(r.watchedServices, key)
}

// Resolve resolves a service name to instances, using cache if available
func (r *Resolver) Resolve(ctx context.Context, serviceName string, onlyHealthy bool, tags []string) ([]discovery.ServiceInstance, error) {
	query := discovery.ServiceQuery{
		Name:        serviceName,
		OnlyHealthy: onlyHealthy,
		Tags:        tags,
		Timeout:     5 * time.Second,
	}

	key := r.queryKey(query)

	// First try from cache
	r.cacheLock.RLock()
	instances, found := r.cache[key]
	r.cacheLock.RUnlock()

	if found && len(instances) > 0 {
		return instances, nil
	}

	// If not in cache or empty, refresh
	return r.refreshService(ctx, query)
}

// Shutdown stops the resolver
func (r *Resolver) Shutdown() {
	close(r.stopCh)
}

// refreshLoop periodically refreshes the cache for watched services
func (r *Resolver) refreshLoop() {
	ticker := time.NewTicker(r.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.refreshWatchedServices()
		case <-r.stopCh:
			return
		}
	}
}

// refreshWatchedServices refreshes all watched services
func (r *Resolver) refreshWatchedServices() {
	r.watchedServicesLock.RLock()
	services := make([]discovery.ServiceQuery, 0, len(r.watchedServices))
	for _, query := range r.watchedServices {
		services = append(services, query)
	}
	r.watchedServicesLock.RUnlock()

	for _, query := range services {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := r.refreshService(ctx, query)
		cancel()

		if err != nil {
			r.logger.Warn("Failed to refresh service",
				logging.String("service", query.Name),
				logging.Error(err),
			)
		}
	}
}

// refreshService refreshes a specific service in the cache
func (r *Resolver) refreshService(ctx context.Context, query discovery.ServiceQuery) ([]discovery.ServiceInstance, error) {
	instances, err := r.client.GetService(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get service %s: %w", query.Name, err)
	}

	key := r.queryKey(query)

	r.cacheLock.Lock()
	r.cache[key] = instances
	r.cacheLock.Unlock()

	return instances, nil
}

// queryKey generates a cache key for a query
func (r *Resolver) queryKey(query discovery.ServiceQuery) string {
	key := query.Name
	if query.OnlyHealthy {
		key += "-healthy"
	}
	return key
}

// ResolveURL builds a URL for a service
func (r *Resolver) ResolveURL(ctx context.Context, serviceName string, path string) (string, error) {
	instances, err := r.Resolve(ctx, serviceName, true, nil)
	if err != nil {
		return "", err
	}

	if len(instances) == 0 {
		return "", errors.New("no instances available for service: " + serviceName)
	}

	// Use the first available instance (in a production system, you might want to implement load balancing)
	instance := instances[0]

	protocol := "http"
	if instance.Secure {
		protocol = "https"
	}

	// Ensure path starts with a slash
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return fmt.Sprintf("%s://%s:%d%s", protocol, instance.Address, instance.Port, path), nil
}
