package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	gatewayConfig "github.com/mohamedfawas/qubool-kallyanam/internal/gateway/config"
	gatewayHandlers "github.com/mohamedfawas/qubool-kallyanam/internal/gateway/handlers"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/db/redis"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/discovery"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/events/pingpong"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/events/rabbitmq"
	gatewayRouter "github.com/mohamedfawas/qubool-kallyanam/pkg/gateway/router"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/middleware"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/telemetry/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/telemetry/metrics"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/telemetry/tracing"
)

func main() {
	// Load configuration
	cfg, err := gatewayConfig.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Set Gin mode based on environment
	if cfg.Common.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create root context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize logger with Loki integration
	logConfig := logging.LoggerConfig{
		ServiceName: "gateway",
		Environment: cfg.Common.Environment,
		Debug:       cfg.Common.Debug,
		Loki: logging.LokiConfig{
			Enabled:   cfg.Telemetry.Logging.Loki.Enabled,
			URL:       cfg.Telemetry.Logging.Loki.URL,
			BatchSize: cfg.Telemetry.Logging.Loki.BatchSize,
			Timeout:   cfg.Telemetry.Logging.Loki.Timeout,
			TenantID:  cfg.Telemetry.Logging.Loki.TenantID,
		},
	}
	logger, err := logging.NewLogger(logConfig)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Initialize metrics provider
	metricsConfig := metrics.Config{
		Enabled:       cfg.Telemetry.Metrics.Enabled,
		ListenAddress: cfg.Telemetry.Metrics.ListenAddress,
		MetricsPath:   cfg.Telemetry.Metrics.MetricsPath,
		ServiceName:   "gateway",
	}
	metricsProvider := metrics.NewPrometheusProvider(metricsConfig, logger)
	if err := metricsProvider.Start(); err != nil {
		logger.Fatal("Failed to start metrics provider", zap.Error(err))
	}
	defer metricsProvider.Stop()

	// Initialize tracing provider
	tracingConfig := tracing.Config{
		Enabled:     cfg.Telemetry.Tracing.Enabled,
		ServiceName: "gateway",
		Endpoint:    cfg.Telemetry.Tracing.Endpoint,
		Insecure:    cfg.Telemetry.Tracing.Insecure,
		SampleRate:  cfg.Telemetry.Tracing.SampleRate,
	}
	tracingProvider := tracing.NewOpenTelemetryProvider(tracingConfig, logger)
	if err := tracingProvider.Start(ctx); err != nil {
		logger.Fatal("Failed to start tracing provider", zap.Error(err))
	}
	defer tracingProvider.Stop(ctx)

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
	redisClient, err := redis.NewClient(context.Background(), redisConfig, "gateway", logger)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	// Ensure Redis client is closed on exit
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("Error closing Redis connection", zap.Error(err))
		}
	}()

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

	rabbitClient := rabbitmq.NewClient(rabbitConfig, logger.With(zap.String("component", "rabbitmq")))
	if err := rabbitClient.Connect(); err != nil {
		logger.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
	}
	defer rabbitClient.Close()

	// Initialize ping service
	pingService, err := pingpong.NewPingService(
		rabbitClient,
		"gateway",
		logger.With(zap.String("component", "ping-service")),
	)
	if err != nil {
		logger.Fatal("Failed to create ping service", zap.Error(err))
	}

	// Start the ping service
	if err := pingService.Start(ctx); err != nil {
		logger.Fatal("Failed to start ping service", zap.Error(err))
	}
	defer pingService.Stop()

	// Send a test ping every minute
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := pingService.SendPing(ctx, "Hello from Gateway!"); err != nil {
					logger.Error("Failed to send ping", zap.Error(err))
				}
			}
		}
	}()

	// Initialize service registry
	registry := discovery.NewServiceRegistry(logger)

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
	serviceRouter := gatewayRouter.NewServiceRouter(registry, tracingProvider, logger)

	// Initialize dependency checker
	dependencyChecker := discovery.NewDependencyChecker(registry, logger)

	// Set up service dependencies based on configuration
	for name, svcCfg := range cfg.ServiceDiscovery.Services {
		for _, dep := range svcCfg.Dependencies {
			if err := dependencyChecker.RegisterDependency(name, dep); err != nil {
				logger.Warn("Failed to register dependency",
					zap.String("service", name),
					zap.String("dependency", dep),
					zap.Error(err))
			}
		}
	}

	// Start dependency checks
	dependencyInterval, _ := time.ParseDuration("30s")
	dependencyChecker.Start(dependencyInterval)

	// Initialize router with telemetry middleware
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.TelemetryMiddleware(metricsProvider, tracingProvider, logger))

	// Load templates for dashboard
	templates := template.Must(template.ParseGlob(filepath.Join("internal", "gateway", "templates", "*.html")))
	router.SetHTMLTemplate(templates)

	// Register service routes
	serviceRouter.RegisterRoutes(router)

	// Register health handler with registry
	healthHandler := gatewayHandlers.NewHealthHandler(logger, redisClient, registry)
	healthHandler.Register(router)

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting gateway service",
			zap.String("host", cfg.Server.Host),
			zap.Int("port", cfg.Server.Port))

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down gateway service...")

	// Create a deadline for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Gateway service exited")
}
