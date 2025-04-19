// pkg/log/structured.go
package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is a wrapper around zap.Logger with additional context
type Logger struct {
	*zap.Logger
	serviceName string
}

// createZapLogger creates a new zap logger with the specified configuration
func createZapLogger(debug bool) (*zap.Logger, error) {
	config := zap.NewProductionConfig()

	if debug {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	config.OutputPaths = []string{"stdout"}
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return config.Build()
}

// NewLogger creates a new structured logger for the specified service
func NewLogger(serviceName string, debug bool) (*Logger, error) {
	zapLogger, err := createZapLogger(debug)
	if err != nil {
		return nil, err
	}

	return &Logger{
		Logger:      zapLogger.Named(serviceName),
		serviceName: serviceName,
	}, nil
}

// WithComponent adds a component field to the logger
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger:      l.With(zap.String("component", component)),
		serviceName: l.serviceName,
	}
}
