package log

import (
	"context"
	"sync/atomic"

	"github.com/go-coldbrew/log/loggers"
	cbslog "github.com/go-coldbrew/log/loggers/slog"
)

// SupportPackageIsVersion1 is a compile-time assertion constant.
// Downstream packages reference this to enforce version compatibility.
const SupportPackageIsVersion1 = true

var defaultLogger atomic.Pointer[Logger]

type logger struct {
	baseLog loggers.BaseLogger
}

func (l *logger) SetLevel(level loggers.Level) {
	l.baseLog.SetLevel(level)
}

func (l *logger) GetLevel() loggers.Level {
	return l.baseLog.GetLevel()
}

func (l *logger) Debug(ctx context.Context, args ...any) {
	l.Log(ctx, loggers.DebugLevel, 1, args...)
}

func (l *logger) Info(ctx context.Context, args ...any) {
	l.Log(ctx, loggers.InfoLevel, 1, args...)
}

func (l *logger) Warn(ctx context.Context, args ...any) {
	l.Log(ctx, loggers.WarnLevel, 1, args...)
}

func (l *logger) Error(ctx context.Context, args ...any) {
	l.Log(ctx, loggers.ErrorLevel, 1, args...)
}

func (l *logger) Log(ctx context.Context, level loggers.Level, skip int, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	logLevel := l.GetLevel()
	if logLevel >= level {
		l.baseLog.Log(ctx, level, skip+1, args...)
		return
	}
	// Only check override if base level would filter this out.
	// Most requests have no override, so this avoids a context lookup on the hot path.
	if overridenLogLevel, found := GetOverridenLogLevel(ctx); found && overridenLogLevel >= level {
		l.baseLog.Log(ctx, level, skip+1, args...)
	}
}

// NewLogger creates a new logger with a provided BaseLogger
// The default logger is slog logger
func NewLogger(log loggers.BaseLogger) Logger {
	l := new(logger)
	l.baseLog = log
	return l
}

// GetLogger returns the global logger
// If the global logger is not set, it will create a new one with slog logger
func GetLogger() Logger {
	l := defaultLogger.Load()
	if l == nil {
		// If the default logger is not set, create a new one with slog logger
		slogLogger := cbslog.NewLogger()
		newLogger := NewLogger(slogLogger)
		defaultLogger.CompareAndSwap(nil, &newLogger)
	}
	return *defaultLogger.Load()
}

// SetLogger sets the global logger
func SetLogger(l Logger) {
	if l != nil {
		defaultLogger.Store(&l)
	}
}

// AddToContext adds log fields to the provided context.
// Any info added here will be included in all logs that use the returned context.
// This is the preferred entry point for adding contextual logging fields and is implemented
// internally using loggers.AddToLogContext.
func AddToContext(ctx context.Context, key string, value any) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return loggers.AddToLogContext(ctx, key, value)
}
