package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/telemetry"
)

// OpenTelemetryProvider implements telemetry.TracingProvider using OpenTelemetry
type OpenTelemetryProvider struct {
	tracerProvider *sdktrace.TracerProvider
	tracer         trace.Tracer
	serviceName    string
	serviceVersion string
	logger         logging.Logger
}

// Config holds configuration for OpenTelemetryProvider
type Config struct {
	ServiceName     string
	ServiceVersion  string
	Endpoint        string
	Insecure        bool
	Sampler         sdktrace.Sampler
	PropagatorNames []string
}

// NewOpenTelemetryProvider creates a new OpenTelemetryProvider
func NewOpenTelemetryProvider(cfg Config, logger logging.Logger) (*OpenTelemetryProvider, error) {
	if logger == nil {
		logger = logging.Get().Named("opentelemetry")
	}

	// Create OTLP exporter
	var exporter *otlptrace.Exporter
	var err error

	// Configure OTLP client options
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
	}

	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	// Initialize exporter
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exporter, err = otlptrace.New(ctx, otlptracegrpc.NewClient(opts...))
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Configure resource attributes
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Configure sampler
	var sampler sdktrace.Sampler
	if cfg.Sampler != nil {
		sampler = cfg.Sampler
	} else {
		// Default: sample 1 in 10 traces in production
		sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1))
	}

	// Create trace provider
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Setup propagators
	var textMapPropagator propagation.TextMapPropagator
	if len(cfg.PropagatorNames) > 0 {
		props := make([]propagation.TextMapPropagator, 0, len(cfg.PropagatorNames))
		for _, name := range cfg.PropagatorNames {
			switch name {
			case "tracecontext":
				props = append(props, propagation.TraceContext{})
			case "baggage":
				props = append(props, propagation.Baggage{})
			default:
				logger.Warn("Unknown propagator", logging.String("name", name))
			}
		}
		textMapPropagator = propagation.NewCompositeTextMapPropagator(props...)
	} else {
		// Default propagators
		textMapPropagator = propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)
	}

	// Set global propagator
	otel.SetTextMapPropagator(textMapPropagator)

	// Set global trace provider
	otel.SetTracerProvider(tracerProvider)

	// Create tracer
	tracer := tracerProvider.Tracer(cfg.ServiceName)

	return &OpenTelemetryProvider{
		tracerProvider: tracerProvider,
		tracer:         tracer,
		serviceName:    cfg.ServiceName,
		serviceVersion: cfg.ServiceVersion,
		logger:         logger,
	}, nil
}

// Name returns the name of the provider
func (p *OpenTelemetryProvider) Name() string {
	return "opentelemetry"
}

// Shutdown gracefully shuts down the provider
func (p *OpenTelemetryProvider) Shutdown(ctx context.Context) error {
	err := p.tracerProvider.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("failed to shutdown trace provider: %w", err)
	}

	p.logger.Info("OpenTelemetry trace provider shutdown complete")
	return nil
}

// StartSpan starts a new span
func (p *OpenTelemetryProvider) StartSpan(ctx context.Context, name string, opts ...telemetry.SpanOption) (context.Context, telemetry.Span) {
	ctx, span := p.tracer.Start(ctx, name)

	// Apply options
	for _, opt := range opts {
		opt(span)
	}

	return ctx, &openTelemetrySpan{span: span}
}

// openTelemetrySpan implements telemetry.Span
type openTelemetrySpan struct {
	span trace.Span
}

func (s *openTelemetrySpan) End() {
	s.span.End()
}

func (s *openTelemetrySpan) SetAttribute(key string, value interface{}) {
	var attr attribute.KeyValue

	switch v := value.(type) {
	case string:
		attr = attribute.String(key, v)
	case int:
		attr = attribute.Int(key, v)
	case int64:
		attr = attribute.Int64(key, v)
	case float64:
		attr = attribute.Float64(key, v)
	case bool:
		attr = attribute.Bool(key, v)
	default:
		attr = attribute.String(key, fmt.Sprintf("%v", v))
	}

	s.span.SetAttributes(attr)
}

func (s *openTelemetrySpan) SetStatus(code int, description string) {
	var statusCode codes.Code

	switch code {
	case 0:
		statusCode = codes.Ok
	case 1:
		statusCode = codes.Error
	default:
		statusCode = codes.Error
	}

	s.span.SetStatus(statusCode, description)
}

func (s *openTelemetrySpan) RecordError(err error) {
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
}

// Common tracing utilities for matrimony application

// StartHTTPSpan starts a span for HTTP request handling
func StartHTTPSpan(ctx context.Context, provider *OpenTelemetryProvider, method, path string) (context.Context, telemetry.Span) {
	return provider.StartSpan(ctx, fmt.Sprintf("HTTP %s %s", method, path),
		telemetry.WithSpanAttributes(map[string]interface{}{
			"http.method": method,
			"http.path":   path,
		}),
	)
}

// StartDBSpan starts a span for database operations
func StartDBSpan(ctx context.Context, provider *OpenTelemetryProvider, operation, collection string) (context.Context, telemetry.Span) {
	return provider.StartSpan(ctx, fmt.Sprintf("DB %s %s", operation, collection),
		telemetry.WithSpanAttributes(map[string]interface{}{
			"db.operation":  operation,
			"db.collection": collection,
		}),
	)
}

// StartMatchingSpan starts a span for profile matching operations
func StartMatchingSpan(ctx context.Context, provider *OpenTelemetryProvider, matchType string, profileID string) (context.Context, telemetry.Span) {
	return provider.StartSpan(ctx, fmt.Sprintf("Match %s", matchType),
		telemetry.WithSpanAttributes(map[string]interface{}{
			"matching.type":    matchType,
			"matching.profile": profileID,
		}),
	)
}
