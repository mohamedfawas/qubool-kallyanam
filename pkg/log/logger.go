// pkg/log/logger.go
package log

import (
	"go.uber.org/zap"
)

// Legacy function kept for backward compatibility
// Will be deprecated in future versions
func NewZapLogger(serviceName string, isDevelopment bool) (*zap.Logger, error) {
	zapLogger, err := createZapLogger(isDevelopment)
	if err != nil {
		return nil, err
	}
	return zapLogger.Named(serviceName), nil
}

// ConvertStructured converts a structured logger to a zap logger
func ConvertToZap(logger *Logger) *zap.Logger {
	return logger.Logger
}

// ConvertFromZap converts a zap logger to a structured logger
func ConvertFromZap(zapLogger *zap.Logger, serviceName string) *Logger {
	return &Logger{
		Logger:      zapLogger,
		serviceName: serviceName,
	}
}
