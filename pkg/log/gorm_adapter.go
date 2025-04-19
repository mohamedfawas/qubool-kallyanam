package log

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

// GormLogAdapter adapts zap logger to be used with gorm
type GormLogAdapter struct {
	logger *zap.Logger
}

// NewGormLogAdapter creates a new gorm log adapter
func NewGormLogAdapter(logger *zap.Logger) *GormLogAdapter {
	return &GormLogAdapter{
		logger: logger.With(zap.String("component", "gorm")),
	}
}

// Printf implements gorm logger interface
func (l *GormLogAdapter) Printf(format string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

func (l *GormLogAdapter) Info(ctx interface{}, format string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

func (l *GormLogAdapter) Warn(ctx interface{}, format string, args ...interface{}) {
	l.logger.Warn(fmt.Sprintf(format, args...))
}

func (l *GormLogAdapter) Error(ctx interface{}, format string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, args...))
}

func (l *GormLogAdapter) Trace(ctx interface{}, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := []zap.Field{
		zap.Duration("elapsed", elapsed),
		zap.String("sql", sql),
		zap.Int64("rows", rows),
	}

	if err != nil {
		fields = append(fields, zap.Error(err))
		l.logger.Error("gorm query error", fields...)
		return
	}

	if elapsed > time.Second {
		l.logger.Warn("slow query", fields...)
		return
	}

	l.logger.Debug("sql query", fields...)
}
