package logging

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger defines a common logging interface with several levels of logging.
// These levels help categorize log messages by importance.
// Each method accepts a message and optional key-value pairs for context.
//
// Example:
// logger.Info("User login", "userID", 123, "ip", "127.0.0.1")
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Fatal(msg string, keysAndValues ...interface{})
	With(keysAndValues ...interface{}) Logger
}

// zapLogger wraps zap.SugaredLogger
// It provides a sugared logger that is easier for humans to write and read logs quickly.
type zapLogger struct {
	sugaredLogger *zap.SugaredLogger
}

// These variables manage the "default" logger used throughout the application.
var (
	defaultLogger     Logger    // Will hold the initialized logger
	defaultLoggerOnce sync.Once // Ensures logger is initialized only once (thread-safe)
)

// newLogger creates a new Logger with the given configuration
func newLogger(isProd bool) Logger {
	// Set up encoder config — how logs should be formatted
	// Common encoder config for both dev and prod
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",                        // Key name for timestamp
		LevelKey:       "level",                       // Log level (info, warn, etc.)
		NameKey:        "logger",                      // Logger name (optional)
		CallerKey:      "caller",                      // Where the log was called from
		MessageKey:     "msg",                         // The actual log message
		StacktraceKey:  "stacktrace",                  // Stack trace (only for error/fatal)
		LineEnding:     zapcore.DefaultLineEnding,     // newline ending
		EncodeTime:     zapcore.ISO8601TimeEncoder,    // Format time as "YYYY-MM-DDTHH:MM:SS"
		EncodeDuration: zapcore.StringDurationEncoder, // Human-readable durations
		EncodeCaller:   zapcore.ShortCallerEncoder,    // Short path to calling file
	}

	// Set defaults for development
	level := zap.DebugLevel // Show debug logs and above
	encoding := "console"   // Human-readable logs
	development := true     // Enable dev-specific features

	// If production mode is enabled, override the above settings
	if isProd {
		level = zap.InfoLevel // Do not show debug logs in prod
		encoding = "json"     // Structured machine-readable format
		development = false
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder // e.g., "info"
	} else {
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder // e.g., "INFO"
	}

	// Create zap.Config based on chosen settings
	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(level), // Set initial log level
		Development:      development,                 // Enable dev-mode stack traces
		Encoding:         encoding,                    // "console" or "json"
		EncoderConfig:    encoderConfig,               // The structure of log entries
		OutputPaths:      []string{"stdout"},          // Send logs to stdout (terminal)
		ErrorOutputPaths: []string{"stderr"},          // Send errors to stderr
	}

	// Build the zap.Logger from config
	logger, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		// If logger creation fails, fall back to example logger
		logger = zap.NewExample()
	}

	// Return our custom zapLogger wrapping SugaredLogger
	return &zapLogger{
		sugaredLogger: logger.Sugar(),
	}
}

// NewDevelopmentLogger returns a logger configured for development.
// Shows debug logs and outputs human-readable logs to console.
func NewDevelopmentLogger() Logger {
	return newLogger(false)
}

// NewProductionLogger returns a logger configured for production.
// Hides debug logs and outputs JSON logs for machines or log aggregators.
func NewProductionLogger() Logger {
	return newLogger(true)
}

// Default returns the default application-wide logger.
// It initializes the logger once using sync.Once (thread-safe).
//
// Based on the APP_ENV environment variable:
//   - If APP_ENV="production", use production logger
//   - Otherwise, use development logger
//
// Example:
// os.Setenv("APP_ENV", "production")
// logger := logging.Default()
func Default() Logger {
	defaultLoggerOnce.Do(func() {
		if os.Getenv("APP_ENV") == "production" {
			defaultLogger = NewProductionLogger()
		} else {
			defaultLogger = NewDevelopmentLogger()
		}
	})
	return defaultLogger
}

// The following methods implement the Logger interface
// They simply delegate to zap.SugaredLogger's methods.

// Debug logs a message at DebugLevel.
// Use for verbose output useful during development or troubleshooting.
func (l *zapLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.sugaredLogger.Debugw(msg, keysAndValues...)
}

// Info logs a message at InfoLevel.
// Use for general information about app behavior.
func (l *zapLogger) Info(msg string, keysAndValues ...interface{}) {
	l.sugaredLogger.Infow(msg, keysAndValues...)
}

// Warn logs a message at WarnLevel.
// Use to highlight a potential issue that isn't blocking.
func (l *zapLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.sugaredLogger.Warnw(msg, keysAndValues...)
}

// Error logs a message at ErrorLevel.
// Use when something went wrong but the app can recover.
func (l *zapLogger) Error(msg string, keysAndValues ...interface{}) {
	l.sugaredLogger.Errorw(msg, keysAndValues...)
}

// Fatal logs a message at FatalLevel and exits the program.
func (l *zapLogger) Fatal(msg string, keysAndValues ...interface{}) {
	l.sugaredLogger.Fatalw(msg, keysAndValues...)
}

// With returns a new Logger with additional key-value context.
// These keys will appear in all future logs from this logger.
//
// Example:
// userLogger := logger.With("userID", 42)
// userLogger.Info("Order placed", "orderID", 1001)
//
// Output:
// {"time":"...","level":"INFO","msg":"Order placed","userID":42,"orderID":1001}
func (l *zapLogger) With(keysAndValues ...interface{}) Logger {
	return &zapLogger{
		sugaredLogger: l.sugaredLogger.With(keysAndValues...),
	}
}
