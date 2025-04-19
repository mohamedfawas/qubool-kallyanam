package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	authConfig "github.com/mohamedfawas/qubool-kallyanam/internal/auth/config"
	authHandlers "github.com/mohamedfawas/qubool-kallyanam/internal/auth/handlers"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/db/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/db/redis"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/log"
)

func main() {
	// Load configuration
	cfg, err := authConfig.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Set Gin mode based on environment
	if cfg.Common.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize logger
	logger, err := log.NewLogger("auth", cfg.Common.Debug)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Initialize Postgres database connection
	pgConfig := postgres.Config{
		Host:     cfg.Database.Postgres.Host,
		Port:     cfg.Database.Postgres.Port,
		Username: cfg.Database.Postgres.Username,
		Password: cfg.Database.Postgres.Password,
		Database: cfg.Database.Postgres.Database,
		SSLMode:  cfg.Database.Postgres.SSLMode,
		MaxConns: cfg.Database.Postgres.MaxConns,
		MaxIdle:  cfg.Database.Postgres.MaxIdle,
		Timeout:  cfg.Database.Postgres.Timeout,
	}
	pgClient, err := postgres.NewClient(pgConfig, "auth", logger)
	if err != nil {
		logger.Fatal("Failed to connect to Postgres", zap.Error(err))
	}
	// Ensure database is closed on exit
	defer func() {
		if err := pgClient.Close(); err != nil {
			logger.Error("Error closing Postgres connection", zap.Error(err))
		}
	}()

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
	redisClient, err := redis.NewClient(context.Background(), redisConfig, "auth", logger)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	// Ensure Redis client is closed on exit
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("Error closing Redis connection", zap.Error(err))
		}
	}()

	// Initialize router
	router := gin.New()
	router.Use(gin.Recovery())

	// Register handlers
	healthHandler := authHandlers.NewHealthHandler(logger, pgClient, redisClient)
	healthHandler.Register(router)

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting auth service",
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

	logger.Info("Shutting down auth service...")

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Auth service exited")
}
