package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds configuration for OpenTelemetry tracing
type Config struct {
	Enabled     bool    `mapstructure:"enabled"`
	ServiceName string  `mapstructure:"service_name"`
	Endpoint    string  `mapstructure:"endpoint"`
	Insecure    bool    `mapstructure:"insecure"`
	SampleRate  float64 `mapstructure:"sample_rate"`
}

// DefaultConfig returns default configuration for tracing
func DefaultConfig() Config {
	return Config{
		Enabled:     true,
		ServiceName: "unknown",
		Endpoint:    "otel-collector:4317",
		Insecure:    true,
		SampleRate:  1.0,
	}
}

// Provider is the interface for tracing providers
type Provider interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Tracer(name string) trace.Tracer
	Propagator() propagation.TextMapPropagator
}

// OpenTelemetryProvider implements the Provider interface using OpenTelemetry
type OpenTelemetryProvider struct {
	config     Config
	logger     *zap.Logger
	exporter   *otlptrace.Exporter
	provider   *sdktrace.TracerProvider
	propagator propagation.TextMapPropagator
}

// NewOpenTelemetryProvider creates a new OpenTelemetryProvider
func NewOpenTelemetryProvider(config Config, logger *zap.Logger) *OpenTelemetryProvider {
	if logger == nil {
		logger = zap.NewNop()
	}

	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	return &OpenTelemetryProvider{
		config:     config,
		logger:     logger,
		propagator: propagator,
	}
}

// Start initializes and starts the tracing provider
func (p *OpenTelemetryProvider) Start(ctx context.Context) error {
	if !p.config.Enabled {
		p.logger.Info("Tracing is disabled")
		return nil
	}

	p.logger.Info("Initializing OpenTelemetry tracing",
		zap.String("service", p.config.ServiceName),
		zap.String("endpoint", p.config.Endpoint))

	// Create OTLP exporter
	var dialOpts []grpc.DialOption
	if p.config.Insecure {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(p.config.Endpoint),
		otlptracegrpc.WithDialOption(dialOpts...),
	)
	if err != nil {
		return fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(p.config.ServiceName),
			attribute.String("environment", "production"),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider
	p.provider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(p.config.SampleRate)),
	)

	// Set global trace provider and propagator
	otel.SetTracerProvider(p.provider)
	otel.SetTextMapPropagator(p.propagator)

	p.logger.Info("OpenTelemetry tracing initialized")
	return nil
}

// Stop stops the tracing provider
func (p *OpenTelemetryProvider) Stop(ctx context.Context) error {
	if !p.config.Enabled || p.provider == nil {
		return nil
	}

	p.logger.Info("Shutting down OpenTelemetry tracing")

	// Shutdown gracefully
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := p.provider.Shutdown(ctxWithTimeout); err != nil {
		return fmt.Errorf("failed to shutdown trace provider: %w", err)
	}

	return nil
}

// Tracer returns a tracer
func (p *OpenTelemetryProvider) Tracer(name string) trace.Tracer {
	if !p.config.Enabled || p.provider == nil {
		return trace.NewNoopTracerProvider().Tracer(name)
	}
	return p.provider.Tracer(name)
}

// Propagator returns the text map propagator
func (p *OpenTelemetryProvider) Propagator() propagation.TextMapPropagator {
	return p.propagator
}
