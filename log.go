package log

import (
	"context"
	"sync"

	"github.com/go-coldbrew/log/loggers"
	"github.com/go-coldbrew/log/loggers/gokit"
)

var (
	defaultLogger Logger
	mu            *sync.Mutex
	once          *sync.Once
)

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
	if defaultLogger == nil {
		once.Do(func() {
			defaultLogger = NewLogger(gokit.NewLogger())
		})
	}
	return defaultLogger
}

// SetLogger sets the global logger
func SetLogger(l Logger) {
	if l != nil {
		mu.Lock()
		defer mu.Unlock()
		defaultLogger = l
	}
}
