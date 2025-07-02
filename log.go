package log

import (
	"context"
	"sync/atomic"

	"github.com/go-coldbrew/log/loggers"
	"github.com/go-coldbrew/log/loggers/gokit"
)

var defaultLogger atomic.Pointer[Logger]

func init() {
	mu = &sync.Mutex{}
	once = &sync.Once{}
}

type logger struct {
	baseLog loggers.BaseLogger
}

func (l *logger) SetLevel(level loggers.Level) {
	l.baseLog.SetLevel(level)
}

func (l *logger) GetLevel() loggers.Level {
	return l.baseLog.GetLevel()
}

func (l *logger) Debug(ctx context.Context, args ...interface{}) {
	l.Log(ctx, loggers.DebugLevel, 1, args...)
}

func (l *logger) Info(ctx context.Context, args ...interface{}) {
	l.Log(ctx, loggers.InfoLevel, 1, args...)
}

func (l *logger) Warn(ctx context.Context, args ...interface{}) {
	l.Log(ctx, loggers.WarnLevel, 1, args...)
}

func (l *logger) Error(ctx context.Context, args ...interface{}) {
	l.Log(ctx, loggers.ErrorLevel, 1, args...)
}

func (l *logger) Log(ctx context.Context, level loggers.Level, skip int, args ...interface{}) {
	if ctx == nil {
		ctx = context.Background()
	}
	logLevel := l.GetLevel()
	if overridenLogLevel, found := GetOverridenLogLevel(ctx); found {
		logLevel = overridenLogLevel
	}
	if logLevel >= level {
		l.baseLog.Log(ctx, level, skip+1, args...)
	}
}

// NewLogger creates a new logger with a provided BaseLogger
// The default logger is gokit logger
func NewLogger(log loggers.BaseLogger) Logger {
	l := new(logger)
	l.baseLog = log
	return l
}

// GetLogger returns the global logger
// If the global logger is not set, it will create a new one with gokit logger
func GetLogger() Logger {
	l := defaultLogger.Load()
	if l == nil {
		// If the default logger is not set, create a new one with gokit logger
		gokitLogger := gokit.NewLogger()
		newLogger := NewLogger(gokitLogger)
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
