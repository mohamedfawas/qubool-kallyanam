package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Config holds configuration for Prometheus metrics
type Config struct {
	Enabled       bool   `mapstructure:"enabled"`
	ListenAddress string `mapstructure:"listen_address"`
	MetricsPath   string `mapstructure:"metrics_path"`
	ServiceName   string
}

// DefaultConfig returns default configuration for metrics
func DefaultConfig() Config {
	return Config{
		Enabled:       true,
		ListenAddress: ":8090",
		MetricsPath:   "/metrics",
		ServiceName:   "unknown",
	}
}

// Provider is the interface for metrics providers
type Provider interface {
	Start() error
	Stop() error
	RequestDuration(method, path string, statusCode int, duration time.Duration)
	IncrementCounter(name, help string, labels map[string]string)
	ObserveHistogram(name, help string, value float64, labels map[string]string, buckets []float64)
	SetGauge(name, help string, value float64, labels map[string]string)
}

// PrometheusProvider implements the Provider interface using Prometheus
type PrometheusProvider struct {
	config            Config
	logger            *zap.Logger
	server            *http.Server
	registry          *prometheus.Registry
	httpDurations     *prometheus.HistogramVec
	customCounters    map[string]*prometheus.CounterVec
	counterLabelNames map[string][]string
	customHistograms  map[string]*prometheus.HistogramVec
	histogramLabelMap map[string][]string
	customGauges      map[string]*prometheus.GaugeVec
	gaugeLabelNames   map[string][]string
}

// NewPrometheusProvider creates a new PrometheusProvider
func NewPrometheusProvider(config Config, logger *zap.Logger) *PrometheusProvider {
	if logger == nil {
		logger = zap.NewNop()
	}

	registry := prometheus.NewRegistry()

	httpDurations := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1, 2, 5, 10},
		},
		[]string{"service", "method", "path", "status"},
	)
	registry.MustRegister(httpDurations)

	return &PrometheusProvider{
		config:            config,
		logger:            logger,
		registry:          registry,
		httpDurations:     httpDurations,
		customCounters:    make(map[string]*prometheus.CounterVec),
		counterLabelNames: make(map[string][]string),
		customHistograms:  make(map[string]*prometheus.HistogramVec),
		histogramLabelMap: make(map[string][]string),
		customGauges:      make(map[string]*prometheus.GaugeVec),
		gaugeLabelNames:   make(map[string][]string),
	}
}

// Start starts the metrics provider
func (p *PrometheusProvider) Start() error {
	if !p.config.Enabled {
		p.logger.Info("Metrics collection is disabled")
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle(p.config.MetricsPath, promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{}))

	p.server = &http.Server{
		Addr:    p.config.ListenAddress,
		Handler: mux,
	}

	go func() {
		p.logger.Info("Starting Prometheus metrics server",
			zap.String("address", p.config.ListenAddress),
			zap.String("path", p.config.MetricsPath))

		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			p.logger.Error("Failed to start metrics server", zap.Error(err))
		}
	}()

	return nil
}

// Stop stops the metrics provider
func (p *PrometheusProvider) Stop() error {
	if p.server != nil {
		p.logger.Info("Stopping Prometheus metrics server")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return p.server.Shutdown(ctx)
	}
	return nil
}

// RequestDuration records HTTP request duration
func (p *PrometheusProvider) RequestDuration(method, path string, statusCode int, duration time.Duration) {
	if !p.config.Enabled {
		return
	}
	p.httpDurations.WithLabelValues(
		p.config.ServiceName,
		method,
		path,
		fmt.Sprintf("%d", statusCode),
	).Observe(duration.Seconds())
}

// IncrementCounter increments a counter with the given name and labels
func (p *PrometheusProvider) IncrementCounter(name, help string, labels map[string]string) {
	if !p.config.Enabled {
		return
	}

	counter, ok := p.customCounters[name]
	if !ok {
		labelNamesWithoutService := make([]string, 0, len(labels))
		for k := range labels {
			labelNamesWithoutService = append(labelNamesWithoutService, k)
		}
		labelNames := append([]string{"service"}, labelNamesWithoutService...)

		counter = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: name,
				Help: help,
			},
			labelNames,
		)
		p.registry.MustRegister(counter)
		p.customCounters[name] = counter
		p.counterLabelNames[name] = labelNamesWithoutService
	}

	labelNamesWithoutService := p.counterLabelNames[name]
	labelValues := make([]string, 0, len(labelNamesWithoutService)+1)
	labelValues = append(labelValues, p.config.ServiceName)
	for _, k := range labelNamesWithoutService {
		value, exists := labels[k]
		if !exists {
			p.logger.Error("missing label for counter", zap.String("label", k), zap.String("metric", name))
			return
		}
		labelValues = append(labelValues, value)
	}

	counter.WithLabelValues(labelValues...).Inc()
}

// ObserveHistogram observes a value in a histogram with the given name and labels
func (p *PrometheusProvider) ObserveHistogram(name, help string, value float64, labels map[string]string, buckets []float64) {
	if !p.config.Enabled {
		return
	}

	histogram, ok := p.customHistograms[name]
	if !ok {
		labelNamesWithoutService := make([]string, 0, len(labels))
		for k := range labels {
			labelNamesWithoutService = append(labelNamesWithoutService, k)
		}
		labelNames := append([]string{"service"}, labelNamesWithoutService...)

		if buckets == nil {
			buckets = prometheus.DefBuckets
		}

		histogram = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    name,
				Help:    help,
				Buckets: buckets,
			},
			labelNames,
		)
		p.registry.MustRegister(histogram)
		p.customHistograms[name] = histogram
		p.histogramLabelMap[name] = labelNamesWithoutService
	}

	labelNamesWithoutService := p.histogramLabelMap[name]
	labelValues := make([]string, 0, len(labelNamesWithoutService)+1)
	labelValues = append(labelValues, p.config.ServiceName)
	for _, k := range labelNamesWithoutService {
		value, exists := labels[k]
		if !exists {
			p.logger.Error("missing label for histogram", zap.String("label", k), zap.String("metric", name))
			return
		}
		labelValues = append(labelValues, value)
	}

	histogram.WithLabelValues(labelValues...).Observe(value)
}

// SetGauge sets a gauge with the given name and labels
func (p *PrometheusProvider) SetGauge(name, help string, value float64, labels map[string]string) {
	if !p.config.Enabled {
		return
	}

	gauge, ok := p.customGauges[name]
	if !ok {
		labelNamesWithoutService := make([]string, 0, len(labels))
		for k := range labels {
			labelNamesWithoutService = append(labelNamesWithoutService, k)
		}
		labelNames := append([]string{"service"}, labelNamesWithoutService...)

		gauge = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: name,
				Help: help,
			},
			labelNames,
		)
		p.registry.MustRegister(gauge)
		p.customGauges[name] = gauge
		p.gaugeLabelNames[name] = labelNamesWithoutService
	}

	labelNamesWithoutService := p.gaugeLabelNames[name]
	labelValues := make([]string, 0, len(labelNamesWithoutService)+1)
	labelValues = append(labelValues, p.config.ServiceName)
	for _, k := range labelNamesWithoutService {
		value, exists := labels[k]
		if !exists {
			p.logger.Error("missing label for gauge", zap.String("label", k), zap.String("metric", name))
			return
		}
		labelValues = append(labelValues, value)
	}

	gauge.WithLabelValues(labelValues...).Set(value)
}
