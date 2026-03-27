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

// toAttr creates an slog.Attr, handling time.Duration specially since slog
// serializes it as nanoseconds (int64) while gokit uses fmt.Sprint ("1s").
func toAttr(key string, val any) slog.Attr {
	if d, ok := val.(time.Duration); ok {
		return slog.String(key, d.String())
	}
	return slog.Any(key, val)
}

func (l *logger) Log(ctx context.Context, level loggers.Level, skip int, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}

	slogLevel := toSlogLevel(level)

	// Gate on our own levelVar first (handles SetLevel on custom handlers
	// whose internal level may not be wired to our levelVar).
	if slogLevel < l.levelVar.Level() {
		return
	}

	if !l.handler.Enabled(ctx, slogLevel) {
		return
	}

	// Re-entry guard: prevent infinite loops with the slog handler bridge.
	// Placed after level checks to avoid context allocation on filtered messages.
	if ctx.Value(slogBackendKey{}) != nil {
		return
	}
	ctx = context.WithValue(ctx, slogBackendKey{}, true)

	// Extract "msg" from args and build attrs directly — no intermediate slice.
	var msg string
	msgIdx := -1
	if len(args) == 1 {
		msg = stringKey(args[0])
	} else if len(args) > 1 {
		for i := 0; i < len(args)-1; i += 2 {
			if k, ok := args[i].(string); ok && k == loggers.MessageKey {
				msg = stringKey(args[i+1])
				msgIdx = i
				break
			}
		}
	}

	// Stack-allocated buffer avoids heap allocation for <=8 attrs (common case).
	var attrBuf [8]slog.Attr
	attrs := attrBuf[:0]

	if l.opt.CallerInfo {
		_, file, line := loggers.FetchCallerInfo(skip+1, l.opt.CallerFileDepth)
		attrs = append(attrs, slog.String(l.opt.CallerFieldName, fmt.Sprintf("%s:%d", file, line)))
	}

	ctxFields := loggers.FromContext(ctx)
	if ctxFields != nil {
		ctxFields.Range(func(k, v any) bool {
			attrs = append(attrs, toAttr(stringKey(k), v))
			return true
		})
	}

	// Build attrs directly from args, skipping the "msg" pair.
	for i := 0; i < len(args)-1; i += 2 {
		if i == msgIdx {
			continue
		}
		attrs = append(attrs, toAttr(stringKey(args[i]), args[i+1]))
	}
	if len(args) > 1 && len(args)%2 != 0 {
		attrs = append(attrs, slog.Any("!BADKEY", args[len(args)-1]))
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
// Note: SetLevel updates the internally tracked level. Both this level and
// the provided handler's own level filtering apply; the stricter one wins.
func NewLoggerWithHandler(handler slog.Handler, options ...loggers.Option) loggers.BaseLogger {
	opt := loggers.GetDefaultOptions()
	for _, f := range options {
		f(&opt)
	}

	levelVar := &slog.LevelVar{}
	levelVar.Set(toSlogLevel(opt.Level))

	if opt.ReplaceStdLogger {
		slog.SetDefault(slog.New(handler))
	}

	return &logger{
		handler:  handler,
		levelVar: levelVar,
		opt:      opt,
	}
}
