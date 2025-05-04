package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// ServiceClient represents a client for a service with health check capability
type ServiceClient struct {
	Name   string
	Client grpc_health_v1.HealthClient
	Conn   *grpc.ClientConn
}

// HealthHandler handles HTTP health check requests
type HealthHandler struct {
	serviceClients []*ServiceClient
	timeout        time.Duration
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(timeout time.Duration) *HealthHandler {
	return &HealthHandler{
		serviceClients: make([]*ServiceClient, 0),
		timeout:        timeout,
	}
}

// AddServiceClient registers a service client for health checks
func (h *HealthHandler) AddServiceClient(name, address string) error {
	// Set up connection to service
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	// Create health client
	client := grpc_health_v1.NewHealthClient(conn)

	// Add to service clients
	h.serviceClients = append(h.serviceClients, &ServiceClient{
		Name:   name,
		Client: client,
		Conn:   conn,
	})

	return nil
}

// HealthCheck handles the GET /health endpoint
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	results := make(map[string]string)
	allHealthy := true

	for _, service := range h.serviceClients {
		resp, err := service.Client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		if err != nil {
			results[service.Name] = "NOT_SERVING"
			allHealthy = false
		} else {
			switch resp.Status {
			case grpc_health_v1.HealthCheckResponse_SERVING:
				results[service.Name] = "SERVING"
			case grpc_health_v1.HealthCheckResponse_NOT_SERVING:
				results[service.Name] = "NOT_SERVING"
				allHealthy = false
			default:
				results[service.Name] = "UNKNOWN"
				allHealthy = false
			}
		}
	}

	status := http.StatusOK
	if !allHealthy {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"status":   allHealthy,
		"services": results,
	})
}

// RegisterRoutes registers the health check routes
func (h *HealthHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/health", h.HealthCheck)
}

// Close closes all gRPC connections
func (h *HealthHandler) Close() {
	for _, service := range h.serviceClients {
		if service.Conn != nil {
			service.Conn.Close()
		}
	}
}
