package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/metrics"
)

// Metrics creates a Gin middleware for HTTP metrics collection
// Follows the same pattern as your existing RequestLogger middleware
func Metrics(metricsRegistry *metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Track basic success/error rates
		status := "success"
		if c.Writer.Status() >= 400 {
			status = "error"
		}

		metricsRegistry.HTTPRequests.WithLabelValues(status).Inc()
	}
}
