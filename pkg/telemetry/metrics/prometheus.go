package metrics

import (
	"context"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/telemetry"
)

const (
	namespace = "matrimony"
)

// PrometheusProvider implements telemetry.MetricsProvider using Prometheus
type PrometheusProvider struct {
	registry    *prometheus.Registry
	counters    map[string]*prometheus.CounterVec
	gauges      map[string]*prometheus.GaugeVec
	histograms  map[string]*prometheus.HistogramVec
	serviceName string
	logger      logging.Logger
	mu          sync.RWMutex
}

// Config holds configuration for PrometheusProvider
type Config struct {
	ServiceName   string
	EnabledLabels []string
}

// NewPrometheusProvider creates a new PrometheusProvider
func NewPrometheusProvider(cfg Config, logger logging.Logger) *PrometheusProvider {
	if logger == nil {
		logger = logging.Get().Named("prometheus")
	}

	registry := prometheus.NewRegistry()

	// Register the Go collector (optional)
	registry.MustRegister(prometheus.NewGoCollector())

	return &PrometheusProvider{
		registry:    registry,
		counters:    make(map[string]*prometheus.CounterVec),
		gauges:      make(map[string]*prometheus.GaugeVec),
		histograms:  make(map[string]*prometheus.HistogramVec),
		serviceName: cfg.ServiceName,
		logger:      logger,
	}
}

// Name returns the name of the provider
func (p *PrometheusProvider) Name() string {
	return "prometheus"
}

// Shutdown gracefully shuts down the provider
func (p *PrometheusProvider) Shutdown(ctx context.Context) error {
	// Prometheus doesn't require explicit shutdown
	p.logger.Info("Prometheus metrics provider shutdown complete")
	return nil
}

// Counter creates or returns a counter metric
func (p *PrometheusProvider) Counter(name, help string, labels ...string) telemetry.CounterMetric {
	p.mu.RLock()
	counter, exists := p.counters[name]
	p.mu.RUnlock()

	if !exists {
		p.mu.Lock()
		defer p.mu.Unlock()

		// Check again in case another goroutine created it
		counter, exists = p.counters[name]
		if !exists {
			counter = prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: p.serviceName,
					Name:      name,
					Help:      help,
				},
				labels,
			)
			p.registry.MustRegister(counter)
			p.counters[name] = counter
		}
	}

	return &prometheusCounter{counter: counter}
}

// Gauge creates or returns a gauge metric
func (p *PrometheusProvider) Gauge(name, help string, labels ...string) telemetry.GaugeMetric {
	p.mu.RLock()
	gauge, exists := p.gauges[name]
	p.mu.RUnlock()

	if !exists {
		p.mu.Lock()
		defer p.mu.Unlock()

		// Check again in case another goroutine created it
		gauge, exists = p.gauges[name]
		if !exists {
			gauge = prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: namespace,
					Subsystem: p.serviceName,
					Name:      name,
					Help:      help,
				},
				labels,
			)
			p.registry.MustRegister(gauge)
			p.gauges[name] = gauge
		}
	}

	return &prometheusGauge{gauge: gauge}
}

// Histogram creates or returns a histogram metric
func (p *PrometheusProvider) Histogram(name, help string, buckets []float64, labels ...string) telemetry.HistogramMetric {
	p.mu.RLock()
	histogram, exists := p.histograms[name]
	p.mu.RUnlock()

	if !exists {
		p.mu.Lock()
		defer p.mu.Unlock()

		// Check again in case another goroutine created it
		histogram, exists = p.histograms[name]
		if !exists {
			if buckets == nil || len(buckets) == 0 {
				// Default buckets
				buckets = prometheus.DefBuckets
			}

			histogram = prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: namespace,
					Subsystem: p.serviceName,
					Name:      name,
					Help:      help,
					Buckets:   buckets,
				},
				labels,
			)
			p.registry.MustRegister(histogram)
			p.histograms[name] = histogram
		}
	}

	return &prometheusHistogram{histogram: histogram}
}

// Handler returns an HTTP handler for the metrics endpoint
func (p *PrometheusProvider) Handler() http.Handler {
	return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{})
}

// prometheusCounter implements telemetry.CounterMetric
type prometheusCounter struct {
	counter *prometheus.CounterVec
}

func (c *prometheusCounter) Inc(labelValues ...string) {
	c.counter.WithLabelValues(labelValues...).Inc()
}

func (c *prometheusCounter) Add(value float64, labelValues ...string) {
	c.counter.WithLabelValues(labelValues...).Add(value)
}

// prometheusGauge implements telemetry.GaugeMetric
type prometheusGauge struct {
	gauge *prometheus.GaugeVec
}

func (g *prometheusGauge) Set(value float64, labelValues ...string) {
	g.gauge.WithLabelValues(labelValues...).Set(value)
}

func (g *prometheusGauge) Inc(labelValues ...string) {
	g.gauge.WithLabelValues(labelValues...).Inc()
}

func (g *prometheusGauge) Dec(labelValues ...string) {
	g.gauge.WithLabelValues(labelValues...).Dec()
}

func (g *prometheusGauge) Add(value float64, labelValues ...string) {
	g.gauge.WithLabelValues(labelValues...).Add(value)
}

func (g *prometheusGauge) Sub(value float64, labelValues ...string) {
	g.gauge.WithLabelValues(labelValues...).Sub(value)
}

// prometheusHistogram implements telemetry.HistogramMetric
type prometheusHistogram struct {
	histogram *prometheus.HistogramVec
}

func (h *prometheusHistogram) Observe(value float64, labelValues ...string) {
	h.histogram.WithLabelValues(labelValues...).Observe(value)
}

// Common metrics creation helpers for matrimony application

// CreateAPIMetrics creates common API metrics
func CreateAPIMetrics(provider *PrometheusProvider) (
	telemetry.CounterMetric,
	telemetry.CounterMetric,
	telemetry.HistogramMetric,
) {
	requestsTotal := provider.Counter(
		"http_requests_total",
		"Total number of HTTP requests",
		"method", "path", "status",
	)

	errorsTotal := provider.Counter(
		"http_errors_total",
		"Total number of HTTP errors",
		"method", "path", "status", "error_type",
	)

	latency := provider.Histogram(
		"http_request_duration_seconds",
		"HTTP request latency in seconds",
		[]float64{0.001, 0.01, 0.1, 0.5, 1, 2, 5, 10},
		"method", "path", "status",
	)

	return requestsTotal, errorsTotal, latency
}

// CreateMatchingMetrics creates metrics for profile matching
func CreateMatchingMetrics(provider *PrometheusProvider) (
	telemetry.CounterMetric,
	telemetry.HistogramMetric,
	telemetry.GaugeMetric,
) {
	matchesTotal := provider.Counter(
		"matches_generated_total",
		"Total number of matches generated",
		"matching_type",
	)

	matchingDuration := provider.Histogram(
		"matching_duration_seconds",
		"Time taken to generate matches",
		[]float64{0.1, 0.5, 1, 5, 10, 30, 60},
		"matching_type", "filters_count",
	)

	activeProfiles := provider.Gauge(
		"active_profiles",
		"Number of active profiles",
		"gender", "age_range", "region",
	)

	return matchesTotal, matchingDuration, activeProfiles
}
