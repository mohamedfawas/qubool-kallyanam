package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/context"
)

// Logger returns a middleware that logs requests
func Logger(logger logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get or generate request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = context.GenerateRequestID()
			c.Header("X-Request-ID", requestID)
		}

		// Store request ID in context
		c.Request = c.Request.WithContext(
			context.WithRequestID(c.Request.Context(), requestID),
		)

		// Start timer and capture path
		start := time.Now()
		path := c.Request.URL.Path

		// Process request
		c.Next()

		// Calculate request time
		duration := time.Since(start)
		status := c.Writer.Status()

		// Log the request
		log := logger.With(
			logging.String("request_id", requestID),
			logging.String("method", c.Request.Method),
			logging.String("path", path),
			logging.String("ip", c.ClientIP()),
			logging.Int("status", status),
			logging.Int("size", c.Writer.Size()),
			logging.Int64("duration_ms", duration.Milliseconds()),
		)

		// Log based on status code
		if status >= 500 {
			log.Error("Server error")
		} else if status >= 400 {
			log.Warn("Client error")
		} else {
			log.Info("Request completed")
		}
	}
}
