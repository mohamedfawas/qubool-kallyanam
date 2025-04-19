package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/telemetry/metrics"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/telemetry/tracing"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// TelemetryMiddleware adds metrics and tracing to HTTP requests
func TelemetryMiddleware(metricsProvider metrics.Provider, tracingProvider tracing.Provider, logger *zap.Logger) gin.HandlerFunc {
	tracer := tracingProvider.Tracer("http")
	propagator := tracingProvider.Propagator()

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Extract trace context from headers
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// Start a new span
		opts := []trace.SpanStartOption{
			trace.WithAttributes(
				semconv.HTTPMethodKey.String(method),
				semconv.HTTPTargetKey.String(path),
				semconv.HTTPURLKey.String(c.Request.URL.String()),
				semconv.HTTPUserAgentKey.String(c.Request.UserAgent()),
				semconv.HTTPRequestContentLengthKey.Int64(c.Request.ContentLength),
			),
			trace.WithSpanKind(trace.SpanKindServer),
		}

		ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", method, path), opts...)
		defer span.End()

		// Set the context with the span
		c.Request = c.Request.WithContext(ctx)

		// Process the request
		c.Next()

		// Record metrics after the request is processed
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		// Record span attributes based on the response
		span.SetAttributes(
			semconv.HTTPStatusCodeKey.Int(statusCode),
			semconv.HTTPResponseContentLengthKey.Int64(int64(c.Writer.Size())),
		)

		// If there was an error, record it in the span
		if len(c.Errors) > 0 {
			span.RecordError(fmt.Errorf("%v", c.Errors))
		}

		// Record metrics
		metricsProvider.RequestDuration(method, path, statusCode, duration)

		// Log the request
		logger.Info("HTTP request",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("duration", duration),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)
	}
}

// GRPCTelemetryInterceptor adds metrics and tracing to gRPC calls
// This would be implemented for gRPC services later
