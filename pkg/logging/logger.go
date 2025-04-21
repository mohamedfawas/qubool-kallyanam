package logging

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Level represents a logging level
type Level string

// Log levels
const (
	DebugLevel  Level = "debug"
	InfoLevel   Level = "info"
	WarnLevel   Level = "warn"
	ErrorLevel  Level = "error"
	DPanicLevel Level = "dpanic"
	PanicLevel  Level = "panic"
	FatalLevel  Level = "fatal"
)

// Config holds logger configuration
type Config struct {
	Level       Level    `yaml:"level" json:"level"`
	Development bool     `yaml:"development" json:"development"`
	Encoding    string   `yaml:"encoding" json:"encoding"` // "json" or "console"
	OutputPaths []string `yaml:"output_paths" json:"output_paths"`
	ServiceName string   `yaml:"service_name" json:"service_name"`
	InstanceID  string   `yaml:"instance_id" json:"instance_id"`
}

// DefaultConfig returns a default logger configuration
func DefaultConfig() Config {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	return Config{
		Level:       InfoLevel,
		Development: false,
		Encoding:    "json",
		OutputPaths: []string{"stdout"},
		ServiceName: "matrimony",
		InstanceID:  hostname,
	}
}

// Logger is the interface for all logging operations
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)

	// WithContext creates a logger with context values
	WithContext(ctx context.Context) Logger

	// With creates a new logger with attached fields
	With(fields ...Field) Logger

	// Named creates a new logger with a name prefix
	Named(name string) Logger

	// Sync flushes any buffered log entries
	Sync() error
}

// Field represents a structured log field
type Field = zapcore.Field

// logger implements the Logger interface using zap
type logger struct {
	zap *zap.Logger
}

// global instance
var globalLogger *logger

// Initialize sets up the global logger with the provided configuration
func Initialize(cfg Config) error {
	zapLogger, err := newZapLogger(cfg)
	if err != nil {
		return err
	}

	globalLogger = &logger{zap: zapLogger}
	return nil
}

// Get returns the global logger
func Get() Logger {
	if globalLogger == nil {
		// Auto-initialize with defaults if not done already
		cfg := DefaultConfig()
		_ = Initialize(cfg)
	}
	return globalLogger
}

// newZapLogger creates a new zap.Logger with the given config
func newZapLogger(cfg Config) (*zap.Logger, error) {
	zapCfg := zap.NewProductionConfig()

	if cfg.Development {
		zapCfg = zap.NewDevelopmentConfig()
	}

	// Configure level
	switch cfg.Level {
	case DebugLevel:
		zapCfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case InfoLevel:
		zapCfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case WarnLevel:
		zapCfg.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case ErrorLevel:
		zapCfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case DPanicLevel:
		zapCfg.Level = zap.NewAtomicLevelAt(zapcore.DPanicLevel)
	case PanicLevel:
		zapCfg.Level = zap.NewAtomicLevelAt(zapcore.PanicLevel)
	case FatalLevel:
		zapCfg.Level = zap.NewAtomicLevelAt(zapcore.FatalLevel)
	}

	// Configure encoding
	zapCfg.Encoding = cfg.Encoding

	// Configure output paths
	if len(cfg.OutputPaths) > 0 {
		zapCfg.OutputPaths = cfg.OutputPaths
	}

	// Add default fields
	fields := []zap.Option{
		zap.Fields(
			zap.String("service", cfg.ServiceName),
			zap.String("instance", cfg.InstanceID),
		),
	}

	logger, err := zapCfg.Build(fields...)
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// Debug logs a debug message
func (l *logger) Debug(msg string, fields ...Field) {
	l.zap.Debug(msg, fields...)
}

// Info logs an info message
func (l *logger) Info(msg string, fields ...Field) {
	l.zap.Info(msg, fields...)
}

// Warn logs a warning message
func (l *logger) Warn(msg string, fields ...Field) {
	l.zap.Warn(msg, fields...)
}

// Error logs an error message
func (l *logger) Error(msg string, fields ...Field) {
	l.zap.Error(msg, fields...)
}

// Fatal logs a fatal message and then calls os.Exit(1)
func (l *logger) Fatal(msg string, fields ...Field) {
	l.zap.Fatal(msg, fields...)
}

// WithContext creates a new logger with context values
func (l *logger) WithContext(ctx context.Context) Logger {
	if ctx == nil {
		return l
	}

	fields := extractContextFields(ctx)
	if len(fields) == 0 {
		return l
	}

	return &logger{zap: l.zap.With(fields...)}
}

// With creates a new logger with attached fields
func (l *logger) With(fields ...Field) Logger {
	return &logger{zap: l.zap.With(fields...)}
}

// Named creates a new logger with a name prefix
func (l *logger) Named(name string) Logger {
	return &logger{zap: l.zap.Named(name)}
}

// Sync flushes any buffered log entries
func (l *logger) Sync() error {
	return l.zap.Sync()
}
