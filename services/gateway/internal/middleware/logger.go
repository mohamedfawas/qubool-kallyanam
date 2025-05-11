package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// RequestLogger is a function that returns a Gin middleware handler.
// This one logs details about each HTTP request and its response.
func RequestLogger(logger logging.Logger) gin.HandlerFunc {
	// This returned function is what Gin will run for each incoming request
	return func(c *gin.Context) {

		start := time.Now()        // Capture the current time before the request starts processing.
		path := c.Request.URL.Path // Extract the request path (like "/api/products")
		method := c.Request.Method // Extract the HTTP method (GET, POST, PUT, DELETE, etc.)

		// Call the next middleware or handler in the chain.
		// If this is the last middleware, it will execute the actual route logic (your endpoint code)
		c.Next()

		// After the request has been processed, calculate how much time it took
		duration := time.Since(start)

		// Get the status code that was sent back to the client (e.g., 200, 404, 500)
		status := c.Writer.Status()

		// Create a log entry with useful information
		// This adds metadata to the log such as the method, path, duration, status, and client IP.
		logEntry := logger.With(
			"method", method,
			"path", path,
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"client_ip", c.ClientIP(),
		)

		// Depending on the status code, log at different levels:
		// - 5xx (500 to 599): Server-side errors => log as Error
		// - 4xx (400 to 499): Client-side errors (e.g., bad request, not found) => log as Warning
		// - Everything else (e.g., 2xx success or 3xx redirects) => log as Info
		if status >= 500 {
			logEntry.Error("Server error")
		} else if status >= 400 {
			logEntry.Warn("Client error")
		} else {
			logEntry.Info("Request completed")
		}
	}
}
