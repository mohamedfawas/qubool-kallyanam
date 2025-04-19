package logging

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/grafana/loki-client-go/loki"
	"github.com/grafana/loki-client-go/pkg/urlutil"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LokiConfig holds configuration for Loki logging
type LokiConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	URL       string `mapstructure:"url"`
	BatchSize int    `mapstructure:"batch_size"`
	Timeout   string `mapstructure:"timeout"`
	TenantID  string `mapstructure:"tenant_id"`
}

// DefaultLokiConfig returns default configuration for Loki
func DefaultLokiConfig() LokiConfig {
	return LokiConfig{
		Enabled:   true,
		URL:       "http://loki:3100/loki/api/v1/push",
		BatchSize: 1024 * 1024,
		Timeout:   "10s",
		TenantID:  "",
	}
}

// LoggerConfig holds configuration for the logger
type LoggerConfig struct {
	ServiceName string     `mapstructure:"service_name"`
	Environment string     `mapstructure:"environment"`
	Debug       bool       `mapstructure:"debug"`
	Loki        LokiConfig `mapstructure:"loki"`
}

// LokiWriter implements zapcore.WriteSyncer by sending logs to Loki
type LokiWriter struct {
	client *loki.Client
	labels model.LabelSet
	ctx    context.Context
	cancel context.CancelFunc
}

// NewLokiWriter creates a new LokiWriter
func NewLokiWriter(config LokiConfig, serviceName, environment string) (*LokiWriter, error) {
	if !config.Enabled {
		return nil, nil
	}

	// Parse timeout
	timeout, err := time.ParseDuration(config.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout duration: %w", err)
	}

	// Parse URL using the standard library
	parsedURL, err := url.Parse(config.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid Loki URL: %w", err)
	}

	// Create client config
	cfg := loki.Config{
		URL:       urlutil.URLValue{URL: parsedURL}, // Wrap parsedURL in urlutil.URLValue
		BatchSize: config.BatchSize,
		Timeout:   timeout,
		TenantID:  config.TenantID,
	}

	// Create client
	c, err := loki.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Loki client: %w", err)
	}

	// Create labels
	hostName, err := os.Hostname()
	if err != nil {
		hostName = "unknown"
	}

	labels := model.LabelSet{
		"service":     model.LabelValue(serviceName),
		"environment": model.LabelValue(environment),
		"host":        model.LabelValue(hostName),
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &LokiWriter{
		client: c,
		labels: labels,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Write implements zapcore.WriteSyncer
func (w *LokiWriter) Write(p []byte) (n int, err error) {
	if w.client == nil {
		return 0, nil
	}

	if err := w.client.Handle(w.labels, time.Now(), string(p)); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Sync implements zapcore.WriteSyncer
func (w *LokiWriter) Sync() error {
	if w.client == nil {
		return nil
	}
	w.client.Stop()
	return nil
}

// Close closes the writer
func (w *LokiWriter) Close() error {
	if w.client == nil {
		return nil
	}
	w.cancel()
	w.client.Stop()
	return nil
}

// NewLogger creates a new zap logger with Loki integration
func NewLogger(config LoggerConfig) (*zap.Logger, error) {
	// Configure encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Configure Loki writer
	var lokiWriter zapcore.WriteSyncer
	if config.Loki.Enabled {
		writer, err := NewLokiWriter(config.Loki, config.ServiceName, config.Environment)
		if err != nil {
			return nil, fmt.Errorf("failed to create Loki writer: %w", err)
		}
		lokiWriter = writer
	}

	// Configure console output
	consoleLevel := zap.InfoLevel
	if config.Debug {
		consoleLevel = zap.DebugLevel
	}

	// Create cores
	var cores []zapcore.Core

	// Console core
	consoleEncoder := zapcore.NewJSONEncoder(encoderConfig)
	consoleCore := zapcore.NewCore(
		consoleEncoder,
		zapcore.Lock(os.Stdout),
		consoleLevel,
	)
	cores = append(cores, consoleCore)

	// Loki core (if enabled)
	if lokiWriter != nil {
		lokiEncoder := zapcore.NewJSONEncoder(encoderConfig)
		lokiCore := zapcore.NewCore(
			lokiEncoder,
			lokiWriter,
			zap.InfoLevel,
		)
		cores = append(cores, lokiCore)
	}

	// Create logger
	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))

	return logger, nil
}
