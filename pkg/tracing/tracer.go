// pkg/tracing/tracer.go - Fixed version based on ChatGPT's analysis
package tracing

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Tracer interface following your logging pattern
type Tracer interface {
	Start(ctx context.Context, spanName string) (context.Context, oteltrace.Span)
	Shutdown(ctx context.Context) error
}

// tracer wraps OpenTelemetry tracer provider
type tracer struct {
	provider *trace.TracerProvider
	cleanup  func()
}

// NewTracer creates a new tracer instance following your constructor pattern
func NewTracer(cfg Config) (Tracer, error) {
	if !cfg.Enabled {
		return &noopTracer{}, nil
	}

	// Create resource with service information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName), // ✅ This sets the service name in Jaeger
			semconv.ServiceVersionKey.String("1.0.0"),
			semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// ✅ Fix endpoint format as ChatGPT suggested
	var exporter trace.SpanExporter

	// Extract host:port from jaeger_url (remove http:// scheme)
	endpoint := cfg.JaegerURL
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimSuffix(endpoint, "/v1/traces")
	endpoint = strings.TrimSuffix(endpoint, "/api/traces")

	// Use WithEndpoint with just host:port format
	exporter, err = otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpoint(endpoint), // ✅ Now just "jaeger:4318"
		otlptracehttp.WithInsecure(),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create exporter with endpoint %s: %w", endpoint, err)
	}

	// Create tracer provider with proper sampling
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(trace.AlwaysSample()), // ✅ Always sample for debugging
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	cleanup := func() {
		tp.Shutdown(context.Background()) // TODO: Add proper error handling
	}

	return &tracer{
		provider: tp,
		cleanup:  cleanup,
	}, nil
}

// Start starts a new span
func (t *tracer) Start(ctx context.Context, spanName string) (context.Context, oteltrace.Span) {
	tracer := otel.Tracer("qubool-kallyanam") // ✅ This is fine as library name
	return tracer.Start(ctx, spanName)
}

// Shutdown shuts down the tracer with proper error handling
func (t *tracer) Shutdown(ctx context.Context) error {
	if t.cleanup != nil {
		t.cleanup()
	}
	return nil
}

// noopTracer for when tracing is disabled
type noopTracer struct{}

func (n *noopTracer) Start(ctx context.Context, spanName string) (context.Context, oteltrace.Span) {
	return ctx, oteltrace.SpanFromContext(ctx)
}

func (n *noopTracer) Shutdown(ctx context.Context) error {
	return nil
}
