// pkg/gateway/router/router.go
package router

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/discovery"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/telemetry/tracing"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
)

// ServiceRouter handles routing requests to appropriate services
type ServiceRouter struct {
	registry        *discovery.ServiceRegistry
	logger          *zap.Logger
	tracingProvider tracing.Provider
	httpClient      *http.Client
}

// NewServiceRouter creates a new service router
func NewServiceRouter(registry *discovery.ServiceRegistry, tracingProvider tracing.Provider, logger *zap.Logger) *ServiceRouter {
	if logger == nil {
		logger = zap.NewNop()
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &ServiceRouter{
		registry:        registry,
		logger:          logger,
		tracingProvider: tracingProvider,
		httpClient:      httpClient,
	}
}

// RegisterRoutes sets up routes for services
func (sr *ServiceRouter) RegisterRoutes(router *gin.Engine) {
	// Register each service's routes
	for _, servicePrefix := range []string{
		"/api/auth",
		"/api/users",
		"/api/chat",
		"/api/admin",
	} {
		serviceName := strings.Split(strings.TrimPrefix(servicePrefix, "/api/"), "/")[0]
		router.Any(fmt.Sprintf("%s/*path", servicePrefix), sr.RouteToService(serviceName))
	}

	// Add service discovery endpoints
	router.GET("/services", sr.ListServices)
	router.GET("/services/:name", sr.GetServiceDetails)
}

// RouteToService returns a handler that routes requests to the appropriate service
func (sr *ServiceRouter) RouteToService(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Lookup service in registry
		service, err := sr.registry.Get(serviceName)
		if err != nil {
			sr.logger.Error("Service not found",
				zap.String("service", serviceName),
				zap.Error(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": fmt.Sprintf("Service %s is not available", serviceName),
			})
			return
		}

		if !service.Healthy {
			sr.logger.Warn("Service is unhealthy", zap.String("service", serviceName))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": fmt.Sprintf("Service %s is currently unhealthy", serviceName),
			})
			return
		}

		// Create the target URL
		targetURL := fmt.Sprintf("http://%s:%d", service.Host, service.Port)
		target, err := url.Parse(targetURL)
		if err != nil {
			sr.logger.Error("Failed to parse target URL",
				zap.String("url", targetURL),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gateway routing error"})
			return
		}

		// Get the path from the request
		path := c.Param("path")
		if path == "" {
			path = "/"
		}

		// Create a reverse proxy
		proxy := httputil.NewSingleHostReverseProxy(target)

		// Customize the director function to handle path rewriting
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.URL.Path = path
			req.Host = target.Host

			// Add tracing headers using the correct carrier
			carrier := propagation.HeaderCarrier(req.Header)
			sr.tracingProvider.Propagator().Inject(c.Request.Context(), carrier)
		}

		// Proxy the request
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// ListServices returns a list of registered services
func (sr *ServiceRouter) ListServices(c *gin.Context) {
	services := sr.registry.ListAll()
	c.JSON(http.StatusOK, services)
}

// GetServiceDetails returns details of a specific service
func (sr *ServiceRouter) GetServiceDetails(c *gin.Context) {
	serviceName := c.Param("name")
	service, err := sr.registry.Get(serviceName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, service)
}
