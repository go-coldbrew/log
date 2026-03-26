// Package slog provides a BaseLogger implementation for log/slog.
package slog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/go-coldbrew/log/loggers"
)

// slogBackendKey is a context sentinel used to prevent infinite loops
// when both the slog backend and slog handler bridge are active.
type slogBackendKey struct{}

type logger struct {
	handler  slog.Handler
	levelVar *slog.LevelVar
	opt      loggers.Options
}

func stringKey(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func (l *logger) Log(ctx context.Context, level loggers.Level, skip int, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	// Re-entry guard: prevent infinite loops with the slog handler bridge.
	if ctx.Value(slogBackendKey{}) != nil {
		return
	}
	ctx = context.WithValue(ctx, slogBackendKey{}, true)

	slogLevel := toSlogLevel(level)

	// Gate on our own levelVar first (handles SetLevel on custom handlers
	// whose internal level may not be wired to our levelVar).
	if slogLevel < l.levelVar.Level() {
		return
	}

	if !l.handler.Enabled(ctx, slogLevel) {
		return
	}

	var msg string
	var kvArgs []any
	if len(args) == 1 {
		msg = stringKey(args[0])
	} else if len(args) > 1 {
		for i := 0; i < len(args)-1; i += 2 {
			if k, ok := args[i].(string); ok && k == "msg" {
				msg = stringKey(args[i+1])
				kvArgs = append(kvArgs, args[:i]...)
				kvArgs = append(kvArgs, args[i+2:]...)
				break
			}
		}
		if kvArgs == nil {
			kvArgs = args
		}
	}

	attrCap := len(kvArgs)/2 + 1
	if l.opt.CallerInfo {
		attrCap++
	}
	attrs := make([]slog.Attr, 0, attrCap)

	if l.opt.CallerInfo {
		_, file, line := loggers.FetchCallerInfo(skip+1, l.opt.CallerFileDepth)
		attrs = append(attrs, slog.String(l.opt.CallerFieldName, fmt.Sprintf("%s:%d", file, line)))
	}

	ctxFields := loggers.FromContext(ctx)
	if ctxFields != nil {
		ctxFields.Range(func(k, v any) bool {
			attrs = append(attrs, slog.Any(stringKey(k), v))
			return true
		})
	}

	for i := 0; i < len(kvArgs)-1; i += 2 {
		attrs = append(attrs, slog.Any(stringKey(kvArgs[i]), kvArgs[i+1]))
	}
	if len(kvArgs)%2 != 0 {
		attrs = append(attrs, slog.Any("!BADKEY", kvArgs[len(kvArgs)-1]))
	}

	var pcs [1]uintptr
	runtime.Callers(skip+2, pcs[:])
	record := slog.NewRecord(time.Now(), slogLevel, msg, pcs[0])
	record.AddAttrs(attrs...)

	_ = l.handler.Handle(ctx, record)
}

func (l *logger) SetLevel(level loggers.Level) {
	l.levelVar.Set(toSlogLevel(level))
}

func (l *logger) GetLevel() loggers.Level {
	return fromSlogLevel(l.levelVar.Level())
}

func toSlogLevel(level loggers.Level) slog.Level {
	switch level {
	case loggers.DebugLevel:
		return slog.LevelDebug
	case loggers.InfoLevel:
		return slog.LevelInfo
	case loggers.WarnLevel:
		return slog.LevelWarn
	case loggers.ErrorLevel:
		return slog.LevelError
	default:
		return slog.LevelError
	}
}

func fromSlogLevel(level slog.Level) loggers.Level {
	switch {
	case level >= slog.LevelError:
		return loggers.ErrorLevel
	case level >= slog.LevelWarn:
		return loggers.WarnLevel
	case level >= slog.LevelInfo:
		return loggers.InfoLevel
	default:
		return loggers.DebugLevel
	}
}

// NewLogger returns a BaseLogger implementation backed by log/slog.
func NewLogger(options ...loggers.Option) loggers.BaseLogger {
	opt := loggers.GetDefaultOptions()
	for _, f := range options {
		f(&opt)
	}

	levelVar := &slog.LevelVar{}
	levelVar.Set(toSlogLevel(opt.Level))

	handlerOpts := &slog.HandlerOptions{
		AddSource: false,
		Level:     levelVar,
		// Wire-compatible output: remap "time" → opt.TimestampFieldName (default "@timestamp")
		// and uppercase slog levels → lowercase ColdBrew levels (e.g., "INFO" → "info").
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Key = opt.TimestampFieldName
			}
			if a.Key == slog.LevelKey {
				a.Key = opt.LevelFieldName
				if lvl, ok := a.Value.Any().(slog.Level); ok {
					a.Value = slog.StringValue(fromSlogLevel(lvl).String())
				}
			}
			return a
		},
	}

	var handler slog.Handler
	if opt.JSONLogs {
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	}

	if opt.ReplaceStdLogger {
		slog.SetDefault(slog.New(handler))
	}

	return &logger{
		handler:  handler,
		levelVar: levelVar,
		opt:      opt,
	}
}

// NewLoggerWithHandler returns a BaseLogger implementation backed by the
// provided slog.Handler. Use this when you need a custom handler
// (e.g., for testing or custom output formats).
// Note: SetLevel updates the internally tracked level but the provided handler's
// own level filtering takes precedence for actual output control.
func NewLoggerWithHandler(handler slog.Handler, options ...loggers.Option) loggers.BaseLogger {
	opt := loggers.GetDefaultOptions()
	for _, f := range options {
		f(&opt)
	}

	levelVar := &slog.LevelVar{}
	levelVar.Set(toSlogLevel(opt.Level))

	return &logger{
		handler:  handler,
		levelVar: levelVar,
		opt:      opt,
	}
}
