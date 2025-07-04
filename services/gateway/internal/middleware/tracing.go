// services/gateway/internal/middleware/tracing.go
package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Tracing creates a Gin middleware for distributed tracing
// Follows the same pattern as your existing middleware
func Tracing(serviceName string) gin.HandlerFunc {
	return otelgin.Middleware(serviceName)
}

// EnrichTrace adds additional attributes to the current span
func EnrichTrace() gin.HandlerFunc {
	return func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		if span.IsRecording() {
			// Add useful attributes
			span.SetAttributes(
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.route", c.FullPath()),
				attribute.String("http.url", c.Request.URL.String()),
				attribute.String("user_agent", c.Request.UserAgent()),
			)

			// Add user info if available
			if userID, exists := c.Get("user_id"); exists {
				span.SetAttributes(attribute.String("user.id", userID.(string)))
			}
		}

		c.Next()

		// Add response status
		if span.IsRecording() {
			statusCode := c.Writer.Status()
			span.SetAttributes(
				attribute.Int("http.status_code", statusCode),
				attribute.Int("http.response_size", c.Writer.Size()),
			)

			// ✅ Mark spans as errors for 4xx and 5xx status codes
			if statusCode >= 400 {
				span.SetStatus(codes.Error, "HTTP Error")
				span.SetAttributes(attribute.Bool("error", true))

				// Add error details for different status codes
				switch {
				case statusCode >= 500:
					span.SetAttributes(attribute.String("error.type", "server_error"))
				case statusCode >= 400:
					span.SetAttributes(attribute.String("error.type", "client_error"))
				}
			}
		}
	}
}
