package main

import (
	"fmt"
	"os"

	adminConfig "github.com/mohamedfawas/qubool-kallyanam/internal/admin/config"
	adminHandlers "github.com/mohamedfawas/qubool-kallyanam/internal/admin/handlers"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/db/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/db/redis"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/service"
	"go.uber.org/zap"
)

// Wrapper function that converts the specific type to interface{}
func loadConfig() (interface{}, error) {
	return adminConfig.Load()
}

func main() {
	// Create a new service instance
	svc, err := service.New("admin", loadConfig)
	if err != nil {
		fmt.Printf("Failed to create admin service: %v\n", err)
		os.Exit(1)
	}

	// Get the config
	cfg := svc.Config.(*adminConfig.Config)

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
	pgClient, err := postgres.NewClient(pgConfig, "admin", svc.Logger)
	if err != nil {
		svc.Logger.Fatal("Failed to connect to Postgres", zap.Error(err))
	}
	svc.AddResource(pgClient)

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
	redisClient, err := redis.NewClient(svc.Context(), redisConfig, "admin", svc.Logger)
	if err != nil {
		svc.Logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	svc.AddResource(redisClient)

	// Register handlers
	healthHandler := adminHandlers.NewHealthHandler(svc.Logger, pgClient, redisClient)
	healthHandler.Register(svc.Router)

	// Configure server
	svc.SetupServer(cfg.Server.Host, cfg.Server.Port)

	// Run the service
	if err := svc.Run(); err != nil {
		svc.Logger.Fatal("Service failed", zap.Error(err))
		os.Exit(1)
	}
}
