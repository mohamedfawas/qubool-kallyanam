package consul

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/hashicorp/consul/api"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/discovery"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// Client implements discovery.Registry using Consul
type Client struct {
	client     *api.Client
	logger     logging.Logger
	datacenter string
	nodeName   string
	serviceIDs map[string]bool
}

// Config contains Consul client configuration
type Config struct {
	// Consul address (e.g., "localhost:8500")
	Address string
	// Datacenter to use
	Datacenter string
	// Node name to use for registrations (defaults to hostname)
	NodeName string
	// Optional ACL token
	Token string
	// Optional scheme (http or https)
	Scheme string
}

// NewClient creates a new Consul client
func NewClient(cfg Config, logger logging.Logger) (*Client, error) {
	if logger == nil {
		logger = logging.Get().Named("consul")
	}

	consulCfg := api.DefaultConfig()

	if cfg.Address != "" {
		consulCfg.Address = cfg.Address
	}

	if cfg.Datacenter != "" {
		consulCfg.Datacenter = cfg.Datacenter
	}

	if cfg.Token != "" {
		consulCfg.Token = cfg.Token
	}

	if cfg.Scheme != "" {
		consulCfg.Scheme = cfg.Scheme
	}

	client, err := api.NewClient(consulCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul client: %w", err)
	}

	nodeName := cfg.NodeName
	if nodeName == "" {
		// Use the agent's node name
		agentInfo, err := client.Agent().Self()
		if err != nil {
			return nil, fmt.Errorf("failed to get agent info: %w", err)
		}
		nodeName, _ = agentInfo["Config"]["NodeName"].(string)
	}

	return &Client{
		client:     client,
		logger:     logger,
		datacenter: cfg.Datacenter,
		nodeName:   nodeName,
		serviceIDs: make(map[string]bool),
	}, nil
}

// Register registers a service instance with Consul
func (c *Client) Register(ctx context.Context, name, address string, port int, opts discovery.RegistrationOptions) (string, error) {
	// Generate a unique ID if multiple instances of the same service exist
	serviceID := fmt.Sprintf("%s-%s-%d", name, address, port)

	// Create service registration
	reg := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    name,
		Address: address,
		Port:    port,
		Tags:    opts.Tags,
		Meta:    opts.Metadata,
	}

	// Add health check if TTL is provided
	if opts.TTL > 0 {
		reg.Check = &api.AgentServiceCheck{
			TTL:                            opts.TTL.String(),
			DeregisterCriticalServiceAfter: (opts.TTL * 3).String(),
		}
	}

	// Register service
	if err := c.client.Agent().ServiceRegister(reg); err != nil {
		return "", fmt.Errorf("failed to register service with Consul: %w", err)
	}

	c.serviceIDs[serviceID] = true
	c.logger.Info("Registered service with Consul",
		logging.String("service_id", serviceID),
		logging.String("service_name", name),
		logging.String("address", address),
		logging.Int("port", port),
	)

	// Initial health pass if using TTL checks
	if opts.TTL > 0 {
		if err := c.ReportHealthy(ctx, serviceID); err != nil {
			c.logger.Warn("Failed to report initial health status",
				logging.String("service_id", serviceID),
				logging.Error(err),
			)
		}
	}

	return serviceID, nil
}

// Deregister removes a service instance from Consul
func (c *Client) Deregister(ctx context.Context, serviceID string) error {
	if err := c.client.Agent().ServiceDeregister(serviceID); err != nil {
		return fmt.Errorf("failed to deregister service from Consul: %w", err)
	}

	delete(c.serviceIDs, serviceID)
	c.logger.Info("Deregistered service from Consul",
		logging.String("service_id", serviceID),
	)

	return nil
}

// GetService finds all instances of a service
func (c *Client) GetService(ctx context.Context, query discovery.ServiceQuery) ([]discovery.ServiceInstance, error) {
	// Default timeout if not specified
	if query.Timeout <= 0 {
		query.Timeout = 5 * time.Second
	}

	// Query parameters
	queryOpts := &api.QueryOptions{
		Datacenter:        c.datacenter,
		AllowStale:        true,
		RequireConsistent: false,
	}

	if deadline, ok := ctx.Deadline(); ok {
		queryOpts.WaitTime = time.Until(deadline)
	} else {
		queryOpts.WaitTime = query.Timeout
	}

	// Get service entries from Consul
	entries, _, err := c.client.Health().Service(query.Name, "", query.OnlyHealthy, queryOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to query service from Consul: %w", err)
	}

	// Convert to ServiceInstance format
	result := make([]discovery.ServiceInstance, 0, len(entries))
	for _, entry := range entries {
		// Skip if it doesn't have all the required tags
		if !hasAllTags(entry.Service.Tags, query.Tags) {
			continue
		}

		healthy := true
		for _, check := range entry.Checks {
			if check.Status != api.HealthPassing {
				healthy = false
				break
			}
		}

		// Skip unhealthy instances if requested
		if query.OnlyHealthy && !healthy {
			continue
		}

		// Parse created time from metadata if available
		var createdAt time.Time
		if createdStr, ok := entry.Service.Meta["created_at"]; ok {
			if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
				createdAt = t
			}
		}

		// Determine if service is secure
		secure := false
		if secureStr, ok := entry.Service.Meta["secure"]; ok {
			secure = secureStr == "true"
		}

		instance := discovery.ServiceInstance{
			ID:        entry.Service.ID,
			Name:      entry.Service.Service,
			Address:   entry.Service.Address,
			Port:      entry.Service.Port,
			Secure:    secure,
			Metadata:  entry.Service.Meta,
			Healthy:   healthy,
			CreatedAt: createdAt,
		}

		result = append(result, instance)
	}

	c.logger.Debug("Retrieved service instances from Consul",
		logging.String("service", query.Name),
		logging.Int("count", len(result)),
	)

	return result, nil
}

// ReportHealthy reports a service instance as healthy
func (c *Client) ReportHealthy(ctx context.Context, serviceID string) error {
	if err := c.client.Agent().PassTTL("service:"+serviceID, ""); err != nil {
		return fmt.Errorf("failed to report service as healthy: %w", err)
	}
	return nil
}

// Shutdown deregisters all services and closes the client
func (c *Client) Shutdown(ctx context.Context) error {
	for serviceID := range c.serviceIDs {
		if err := c.Deregister(ctx, serviceID); err != nil {
			c.logger.Warn("Failed to deregister service during shutdown",
				logging.String("service_id", serviceID),
				logging.Error(err),
			)
		}
	}
	c.logger.Info("Consul client shut down")
	return nil
}

// hasAllTags checks if the service has all the required tags
func hasAllTags(serviceTags, requiredTags []string) bool {
	if len(requiredTags) == 0 {
		return true
	}

	// Create a map for faster lookup
	tagMap := make(map[string]bool, len(serviceTags))
	for _, tag := range serviceTags {
		tagMap[tag] = true
	}

	// Check if all required tags are present
	for _, tag := range requiredTags {
		if !tagMap[tag] {
			return false
		}
	}

	return true
}

// ParseConsulURL parses a Consul URL into a Config
func ParseConsulURL(consulURL string) (Config, error) {
	cfg := Config{}

	u, err := url.Parse(consulURL)
	if err != nil {
		return cfg, fmt.Errorf("invalid Consul URL: %w", err)
	}

	cfg.Scheme = u.Scheme
	cfg.Address = u.Host

	// Handle basic auth
	if u.User != nil {
		cfg.Token = u.User.String()
	}

	// Handle query parameters
	q := u.Query()
	if dc := q.Get("dc"); dc != "" {
		cfg.Datacenter = dc
	}

	if node := q.Get("node"); node != "" {
		cfg.NodeName = node
	}

	return cfg, nil
}
