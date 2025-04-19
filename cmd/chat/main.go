package main

import (
	"fmt"
	"os"

	chatConfig "github.com/mohamedfawas/qubool-kallyanam/internal/chat/config"
	chatHandlers "github.com/mohamedfawas/qubool-kallyanam/internal/chat/handlers"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/db/mongodb"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/db/redis"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/service"
	"go.uber.org/zap"
)

// Wrapper function that converts the specific type to interface{}
func loadConfig() (interface{}, error) {
	return chatConfig.Load()
}

func main() {
	// Create a new service instance
	svc, err := service.New("chat", loadConfig)
	if err != nil {
		fmt.Printf("Failed to create chat service: %v\n", err)
		os.Exit(1)
	}

	// Get the config
	cfg := svc.Config.(*chatConfig.Config)

	// Initialize MongoDB client
	mongoConfig := mongodb.Config{
		URI:      cfg.Database.MongoDB.URI,
		Database: cfg.Database.MongoDB.Database,
		Username: cfg.Database.MongoDB.Username,
		Password: cfg.Database.MongoDB.Password,
		MaxConns: cfg.Database.MongoDB.MaxConns,
		MinConns: cfg.Database.MongoDB.MinConns,
		Timeout:  cfg.Database.MongoDB.Timeout,
	}
	mongoClient, err := mongodb.NewClient(svc.Context(), mongoConfig, "chat", svc.Logger)
	if err != nil {
		svc.Logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
	}
	svc.AddResource(mongoClient)

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
	redisClient, err := redis.NewClient(svc.Context(), redisConfig, "chat", svc.Logger)
	if err != nil {
		svc.Logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	svc.AddResource(redisClient)

	// Register handlers
	healthHandler := chatHandlers.NewHealthHandler(svc.Logger, mongoClient, redisClient)
	healthHandler.Register(svc.Router)

	// Configure server
	svc.SetupServer(cfg.Server.Host, cfg.Server.Port)

	// Run the service
	if err := svc.Run(); err != nil {
		svc.Logger.Fatal("Service failed", zap.Error(err))
		os.Exit(1)
	}
}
