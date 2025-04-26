// logger.go
package logging

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Level represents a logging level
type Level string

// Available log levels
const (
	DebugLevel Level = "debug"
	InfoLevel  Level = "info"
	WarnLevel  Level = "warn"
	ErrorLevel Level = "error"
	FatalLevel Level = "fatal"
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

// Logger interface provides logging methods
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	WithContext(ctx context.Context) Logger
	With(fields ...Field) Logger
	Named(name string) Logger
	Sync() error
}

// Field represents a structured log field
type Field = zapcore.Field

// Global logger instance
var globalLogger *zapLogger

// Initialize sets up the global logger with the provided configuration
func Initialize(cfg Config) error {
	zapLog, err := newZapLogger(cfg)
	if err != nil {
		return err
	}

	globalLogger = &zapLogger{zap: zapLog}
	return nil
}

// Get returns the global logger, initializing with defaults if needed
func Get() Logger {
	if globalLogger == nil {
		_ = Initialize(DefaultConfig())
	}
	return globalLogger
}

// zapLogger implements the Logger interface using zap
type zapLogger struct {
	zap *zap.Logger
}

// Create a new zap.Logger with the given config
func newZapLogger(cfg Config) (*zap.Logger, error) {
	zapCfg := zap.NewProductionConfig()

	if cfg.Development {
		zapCfg = zap.NewDevelopmentConfig()
	}

	// Set log level
	zapLevel := zapcore.InfoLevel
	switch cfg.Level {
	case DebugLevel:
		zapLevel = zapcore.DebugLevel
	case InfoLevel:
		zapLevel = zapcore.InfoLevel
	case WarnLevel:
		zapLevel = zapcore.WarnLevel
	case ErrorLevel:
		zapLevel = zapcore.ErrorLevel
	case FatalLevel:
		zapLevel = zapcore.FatalLevel
	}
	zapCfg.Level = zap.NewAtomicLevelAt(zapLevel)

	// Set encoding
	zapCfg.Encoding = cfg.Encoding

	// Set output paths
	if len(cfg.OutputPaths) > 0 {
		zapCfg.OutputPaths = cfg.OutputPaths
	}

	// Add default fields
	logger, err := zapCfg.Build(
		zap.Fields(
			zap.String("service", cfg.ServiceName),
			zap.String("instance", cfg.InstanceID),
		),
	)

	return logger, err
}

// Log methods implementation
func (l *zapLogger) Debug(msg string, fields ...Field) {
	l.zap.Debug(msg, fields...)
}

func (l *zapLogger) Info(msg string, fields ...Field) {
	l.zap.Info(msg, fields...)
}

func (l *zapLogger) Warn(msg string, fields ...Field) {
	l.zap.Warn(msg, fields...)
}

func (l *zapLogger) Error(msg string, fields ...Field) {
	l.zap.Error(msg, fields...)
}

func (l *zapLogger) Fatal(msg string, fields ...Field) {
	l.zap.Fatal(msg, fields...)
}

func (l *zapLogger) WithContext(ctx context.Context) Logger {
	if ctx == nil {
		return l
	}

	fields := extractContextFields(ctx)
	if len(fields) == 0 {
		return l
	}

	return &zapLogger{zap: l.zap.With(fields...)}
}

func (l *zapLogger) With(fields ...Field) Logger {
	return &zapLogger{zap: l.zap.With(fields...)}
}

func (l *zapLogger) Named(name string) Logger {
	return &zapLogger{zap: l.zap.Named(name)}
}

func (l *zapLogger) Sync() error {
	return l.zap.Sync()
}
