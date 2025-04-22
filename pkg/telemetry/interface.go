package telemetry

import (
	"context"
)

// Provider is the top-level telemetry provider interface
type Provider interface {
	// Name returns the name of the telemetry provider
	Name() string

	// Shutdown gracefully shuts down the telemetry provider
	Shutdown(ctx context.Context) error
}

// MetricsProvider defines the interface for metrics collection
type MetricsProvider interface {
	Provider

	// Counter creates or returns a counter metric
	Counter(name, help string, labels ...string) CounterMetric

	// Gauge creates or returns a gauge metric
	Gauge(name, help string, labels ...string) GaugeMetric

	// Histogram creates or returns a histogram metric
	Histogram(name, help string, buckets []float64, labels ...string) HistogramMetric
}

// CounterMetric represents a counter metric
type CounterMetric interface {
	// Inc increments the counter by 1
	Inc(labelValues ...string)

	// Add adds the given value to the counter
	Add(value float64, labelValues ...string)
}

// GaugeMetric represents a gauge metric
type GaugeMetric interface {
	// Set sets the gauge to the given value
	Set(value float64, labelValues ...string)

	// Inc increments the gauge by 1
	Inc(labelValues ...string)

	// Dec decrements the gauge by 1
	Dec(labelValues ...string)

	// Add adds the given value to the gauge
	Add(value float64, labelValues ...string)

	// Sub subtracts the given value from the gauge
	Sub(value float64, labelValues ...string)
}

// HistogramMetric represents a histogram metric
type HistogramMetric interface {
	// Observe adds an observation to the histogram
	Observe(value float64, labelValues ...string)
}

// TracingProvider defines the interface for tracing operations
type TracingProvider interface {
	Provider

	// StartSpan starts a new span
	StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)
}

// Span represents a tracing span
type Span interface {
	// End completes the span
	End()

	// SetAttribute sets an attribute on the span
	SetAttribute(key string, value interface{})

	// SetStatus sets the status of the span
	SetStatus(code int, description string)

	// RecordError records an error on the span
	RecordError(err error)
}

// SpanOption defines a function that configures a span
type SpanOption func(interface{})

// WithSpanAttributes returns a SpanOption that adds attributes to a span
func WithSpanAttributes(attributes map[string]interface{}) SpanOption {
	return func(s interface{}) {
		if span, ok := s.(Span); ok {
			for k, v := range attributes {
				span.SetAttribute(k, v)
			}
		}
	}
}

// LoggingProvider defines the interface for logging operations
type LoggingProvider interface {
	Provider

	// Log sends a log entry to the provider
	Log(ctx context.Context, level string, message string, fields map[string]interface{}) error
}
