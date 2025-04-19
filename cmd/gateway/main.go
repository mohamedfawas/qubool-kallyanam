package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	gatewayConfig "github.com/mohamedfawas/qubool-kallyanam/internal/gateway/config"
	gatewayHandlers "github.com/mohamedfawas/qubool-kallyanam/internal/gateway/handlers"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/db/redis"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/discovery"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/events/pingpong"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/events/rabbitmq"
	gatewayRouter "github.com/mohamedfawas/qubool-kallyanam/pkg/gateway/router"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/middleware"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/service"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/telemetry/metrics"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/telemetry/tracing"
)

// Wrapper function that converts the specific type to interface{}
func loadConfig() (interface{}, error) {
	return gatewayConfig.Load()
}

func main() {
	// Create a new service instance
	svc, err := service.New("gateway", loadConfig)
	if err != nil {
		fmt.Printf("Failed to create gateway service: %v\n", err)
		os.Exit(1)
	}

	// Get the config
	cfg := svc.Config.(*gatewayConfig.Config)

	// Initialize metrics provider
	metricsConfig := metrics.Config{
		Enabled:       cfg.Telemetry.Metrics.Enabled,
		ListenAddress: cfg.Telemetry.Metrics.ListenAddress,
		MetricsPath:   cfg.Telemetry.Metrics.MetricsPath,
		ServiceName:   "gateway",
	}
	metricsProvider := metrics.NewPrometheusProvider(metricsConfig, svc.Logger)
	if err := metricsProvider.Start(); err != nil {
		svc.Logger.Fatal("Failed to start metrics provider", zap.Error(err))
	}
	// We can't use AddResource here because metricsProvider doesn't implement io.Closer
	// Instead we'll manually stop it in a defer
	defer metricsProvider.Stop()

	// Initialize tracing provider
	tracingConfig := tracing.Config{
		Enabled:     cfg.Telemetry.Tracing.Enabled,
		ServiceName: "gateway",
		Endpoint:    cfg.Telemetry.Tracing.Endpoint,
		Insecure:    cfg.Telemetry.Tracing.Insecure,
		SampleRate:  cfg.Telemetry.Tracing.SampleRate,
	}
	tracingProvider := tracing.NewOpenTelemetryProvider(tracingConfig, svc.Logger)
	if err := tracingProvider.Start(svc.Context()); err != nil {
		svc.Logger.Fatal("Failed to start tracing provider", zap.Error(err))
	}
	defer tracingProvider.Stop(svc.Context())

	// Initialize Redis client
	redisConfig := redis.Config{
		Host:     cfg.Database.Redis.Host,
		Port:     cfg.Database.Redis.Port,
		Password: cfg.Database.Redis.Password,
		DB:       cfg.Database.Redis.DB,
		MaxConns: cfg.Database.Redis.MaxConns,
		MinIdle:  cfg.Database.Redis.MinIdle,
		Timeout:  cfg.Database.Redis.Timeout,
	}
	redisClient, err := redis.NewClient(svc.Context(), redisConfig, "gateway", svc.Logger)
	if err != nil {
		svc.Logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	svc.AddResource(redisClient)

	// Initialize RabbitMQ client
	rabbitConfig := rabbitmq.Config{
		Host:           cfg.Messaging.RabbitMQ.Host,
		Port:           cfg.Messaging.RabbitMQ.Port,
		Username:       cfg.Messaging.RabbitMQ.Username,
		Password:       cfg.Messaging.RabbitMQ.Password,
		VHost:          cfg.Messaging.RabbitMQ.VHost,
		Reconnect:      cfg.Messaging.RabbitMQ.Reconnect,
		ReconnectDelay: cfg.Messaging.RabbitMQ.ReconnectDelay,
	}
	rabbitClient := rabbitmq.NewClient(rabbitConfig, svc.Logger.With(zap.String("component", "rabbitmq")))
	if err := rabbitClient.Connect(); err != nil {
		svc.Logger.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
	}
	// Add RabbitMQ client as a resource if it implements io.Closer
	// If not, we need to manually close it in a defer
	defer rabbitClient.Close()

	// Initialize ping service
	pingService, err := pingpong.NewPingService(
		rabbitClient,
		"gateway",
		svc.Logger.With(zap.String("component", "ping-service")),
	)
	if err != nil {
		svc.Logger.Fatal("Failed to create ping service", zap.Error(err))
	}
	// Start the ping service
	if err := pingService.Start(svc.Context()); err != nil {
		svc.Logger.Fatal("Failed to start ping service", zap.Error(err))
	}
	defer pingService.Stop()

	// Send a test ping every minute
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-svc.Context().Done():
				return
			case <-ticker.C:
				if err := pingService.SendPing(svc.Context(), "Hello from Gateway!"); err != nil {
					svc.Logger.Error("Failed to send ping", zap.Error(err))
				}
			}
		}
	}()

	// Initialize service registry
	registry := discovery.NewServiceRegistry(svc.Logger)

	// Initialize HTTP client for health checks and dependency checks
	httpClient := &http.Client{Timeout: 5 * time.Second}

	// Register services from config
	refreshInterval, _ := time.ParseDuration(cfg.ServiceDiscovery.RefreshInterval)
	for name, svcCfg := range cfg.ServiceDiscovery.Services {
		registry.Register(&discovery.ServiceInfo{
			Name:             name,
			Host:             svcCfg.Host,
			Port:             svcCfg.Port,
			Type:             svcCfg.Type,
			Routes:           svcCfg.Routes,
			HealthCheck:      svcCfg.HealthCheck,
			Dependencies:     []string{},
			DependencyStatus: make(map[string]bool),
		})
	}

	// Start health checks
	registry.StartHealthCheck(httpClient, refreshInterval)

	// Create service router
	serviceRouter := gatewayRouter.NewServiceRouter(registry, tracingProvider, svc.Logger)

	// Initialize dependency checker
	dependencyChecker := discovery.NewDependencyChecker(registry, svc.Logger)

	// Set up service dependencies based on configuration
	for name, svcCfg := range cfg.ServiceDiscovery.Services {
		for _, dep := range svcCfg.Dependencies {
			if err := dependencyChecker.RegisterDependency(name, dep); err != nil {
				svc.Logger.Warn("Failed to register dependency",
					zap.String("service", name),
					zap.String("dependency", dep),
					zap.Error(err))
			}
		}
	}

	// Start dependency checks
	dependencyInterval, _ := time.ParseDuration("30s")
	dependencyChecker.Start(dependencyInterval)

	// Use telemetry middleware
	svc.Router.Use(middleware.TelemetryMiddleware(metricsProvider, tracingProvider, svc.Logger))

	// Load templates for dashboard
	templates := template.Must(template.ParseGlob(filepath.Join("internal", "gateway", "templates", "*.html")))
	svc.Router.SetHTMLTemplate(templates)

	// Register service routes
	serviceRouter.RegisterRoutes(svc.Router)

	// Register health handler with registry
	healthHandler := gatewayHandlers.NewHealthHandler(svc.Logger, redisClient, registry)
	healthHandler.Register(svc.Router)

	// Configure server
	svc.SetupServer(cfg.Server.Host, cfg.Server.Port)

	// Run the service
	if err := svc.Run(); err != nil {
		svc.Logger.Fatal("Service failed", zap.Error(err))
		os.Exit(1)
	}
}
