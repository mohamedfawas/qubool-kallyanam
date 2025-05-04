package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// RequestLogger creates a middleware for logging requests
func RequestLogger(logger logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Process request
		c.Next()

		// Log after request is processed
		duration := time.Since(start)
		status := c.Writer.Status()

		// Log different levels based on status code
		logEntry := logger.With(
			"method", method,
			"path", path,
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"client_ip", c.ClientIP(),
		)

		if status >= 500 {
			logEntry.Error("Server error")
		} else if status >= 400 {
			logEntry.Warn("Client error")
		} else {
			logEntry.Info("Request completed")
		}
	}
}
