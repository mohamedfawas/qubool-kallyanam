// pkg/service/server.go
package service

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ServerOptions contains options for configuring the HTTP server
type ServerOptions struct {
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	MaxHeaderBytes    int
}

// DefaultServerOptions returns the default server options
func DefaultServerOptions() ServerOptions {
	return ServerOptions{
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}
}

// CreateServer creates an HTTP server with the given options
func CreateServer(host string, port int, handler http.Handler, options ServerOptions) *http.Server {
	return &http.Server{
		Addr:              StringPort(host, port),
		Handler:           handler,
		ReadTimeout:       options.ReadTimeout,
		WriteTimeout:      options.WriteTimeout,
		IdleTimeout:       options.IdleTimeout,
		ReadHeaderTimeout: options.ReadHeaderTimeout,
		MaxHeaderBytes:    options.MaxHeaderBytes,
	}
}

// StringPort combines a host and port into a string address
func StringPort(host string, port int) string {
	return host + ":" + string(port)
}

// GracefulShutdown attempts to gracefully shut down the server
func GracefulShutdown(ctx context.Context, server *http.Server, logger *zap.Logger) error {
	logger.Info("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
		return err
	}

	logger.Info("Server gracefully stopped")
	return nil
}

// RegisterHandlers is a helper function to register handlers with a router
func RegisterHandlers(router *gin.Engine, handlers ...interface{}) {
	// This function would register handlers with the router
	// Implementation depends on your handler structure
	// This is a placeholder
}
