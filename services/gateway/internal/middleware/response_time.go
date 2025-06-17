package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ResponseTiming adds response time to HTTP headers (industry standard)
func ResponseTiming() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record start time
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration and add to response header
		duration := time.Since(start)
		c.Header("X-Response-Time", strconv.FormatInt(duration.Milliseconds(), 10)+"ms")
	}
}
