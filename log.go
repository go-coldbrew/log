package log

import (
	"context"
	"log/slog"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/go-coldbrew/log/loggers"
)

// SupportPackageIsVersion1 is a compile-time assertion constant.
// Downstream packages reference this to enforce version compatibility.
const SupportPackageIsVersion1 = true

var defaultHandler atomic.Pointer[Handler]

// cbLogger wraps a Handler and provides ColdBrew's convenience methods.
type cbLogger struct {
	handler *Handler
}

func (l *cbLogger) SetLevel(level loggers.Level) {
	l.handler.SetLevel(level)
}

func (l *cbLogger) GetLevel() loggers.Level {
	return l.handler.GetLevel()
}

func (l *cbLogger) Debug(ctx context.Context, args ...any) {
	l.Log(ctx, loggers.DebugLevel, 1, args...)
}

func (l *cbLogger) Info(ctx context.Context, args ...any) {
	l.Log(ctx, loggers.InfoLevel, 1, args...)
}

func (l *cbLogger) Warn(ctx context.Context, args ...any) {
	l.Log(ctx, loggers.WarnLevel, 1, args...)
}

func (l *cbLogger) Error(ctx context.Context, args ...any) {
	l.Log(ctx, loggers.ErrorLevel, 1, args...)
}

func (l *cbLogger) Log(ctx context.Context, level loggers.Level, skip int, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}

	slogLevel := ToSlogLevel(level)

	if !l.handler.Enabled(ctx, slogLevel) {
		return
	}

	// Parse args using ColdBrew's convention:
	// odd-length: first arg is message, rest are key-value pairs
	// even-length: scan for explicit "msg" key
	var msg string
	msgIdx := -1
	if len(args) == 1 {
		msg = stringKey(args[0])
	} else if len(args) > 1 {
		if len(args)%2 != 0 {
			msg = stringKey(args[0])
			args = args[1:]
		} else {
			for i := 0; i < len(args)-1; i += 2 {
				if k, ok := args[i].(string); ok && k == loggers.MessageKey {
					msg = stringKey(args[i+1])
					msgIdx = i
					break
				}
			}
		}
	}

	var pcs [1]uintptr
	runtime.Callers(skip+2, pcs[:])

	var attrBuf [8]slog.Attr
	attrs := attrBuf[:0]

	for i := 0; i < len(args)-1; i += 2 {
		if i == msgIdx {
			continue
		}
		attrs = append(attrs, toAttr(stringKey(args[i]), args[i+1]))
	}
	if len(args) > 1 && len(args)%2 != 0 {
		attrs = append(attrs, slog.Any("!BADKEY", args[len(args)-1]))
	}

	record := slog.NewRecord(time.Now(), slogLevel, msg, pcs[0])
	record.AddAttrs(attrs...)

	_ = l.handler.Handle(ctx, record)
}

// handler returns the underlying Handler (unexported to keep Logger interface clean).
func (l *cbLogger) handler_() *Handler {
	return l.handler
}

// getOrInitHandler returns the global Handler, lazily initializing it if needed.
func getOrInitHandler() *Handler {
	h := defaultHandler.Load()
	if h == nil {
		newHandler := NewHandler()
		defaultHandler.CompareAndSwap(nil, newHandler)
		h = defaultHandler.Load()
	}
	return h
}

// SetDefault sets the global ColdBrew handler and also calls slog.SetDefault
// so that native slog.InfoContext/slog.ErrorContext calls automatically get
// ColdBrew context fields injected.
func SetDefault(h *Handler) {
	if h == nil {
		return
	}
	defaultHandler.Store(h)
	slog.SetDefault(slog.New(h))
}

// GetHandler returns the global ColdBrew Handler.
func GetHandler() *Handler {
	return getOrInitHandler()
}

// GetLogger returns the global logger.
// If the global logger is not set, it will create a new one with the default Handler.
func GetLogger() Logger {
	return &cbLogger{handler: getOrInitHandler()}
}

// SetLogger sets the global logger.
//
// Deprecated: Use SetDefault with a *Handler instead. SetLogger is kept for
// backward compatibility with code that implements the Logger interface.
func SetLogger(l Logger) {
	if l == nil {
		return
	}
	// If the logger wraps a Handler, use it directly.
	type handlerProvider interface{ handler_() *Handler }
	if hl, ok := l.(handlerProvider); ok {
		SetDefault(hl.handler_())
		return
	}
	// Legacy path: wrap the BaseLogger in a compatibility adapter.
	h := newHandlerFromBaseLogger(l)
	defaultHandler.Store(h)
}

// NewLogger creates a new logger with a provided BaseLogger.
//
// Deprecated: Use NewHandler or NewHandlerWithInner instead. NewLogger is kept
// for backward compatibility.
func NewLogger(bl loggers.BaseLogger) Logger { //nolint:staticcheck // backward compatibility
	h := newHandlerFromBaseLogger(bl)
	return &cbLogger{handler: h}
}

// AddToContext adds log fields to the provided context.
// Any info added here will be included in all logs that use the returned context.
// This works with both ColdBrew's log functions and native slog.InfoContext.
func AddToContext(ctx context.Context, key string, value any) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return loggers.AddToLogContext(ctx, key, value)
}

// AddAttrsToContext adds typed slog.Attr fields to the context.
// Each Attr is stored keyed by its own Key, and emitted without interface boxing.
// Use this with slog.LogAttrs for the zero-boxing logging path:
//
//	ctx = log.AddAttrsToContext(ctx,
//	    slog.String("trace_id", id),
//	    slog.Int("user_id", uid),
//	)
//	slog.LogAttrs(ctx, slog.LevelInfo, "handled", slog.Int("status", 200))
func AddAttrsToContext(ctx context.Context, attrs ...slog.Attr) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	for _, a := range attrs {
		if a.Key != "" {
			ctx = loggers.AddToLogContext(ctx, a.Key, a)
		}
	}
	return ctx
}

// baseLoggerAdapter wraps a BaseLogger as a slog.Handler for backward compatibility.
type baseLoggerAdapter struct {
	bl loggers.BaseLogger //nolint:staticcheck // backward compatibility adapter
}

func newHandlerFromBaseLogger(bl loggers.BaseLogger) *Handler { //nolint:staticcheck // backward compatibility
	levelVar := &slog.LevelVar{}
	levelVar.Set(ToSlogLevel(bl.GetLevel()))
	return &Handler{
		inner:    &baseLoggerAdapter{bl: bl},
		levelVar: levelVar,
		opts:     loggers.GetDefaultOptions(),
	}
}

func (a *baseLoggerAdapter) Enabled(_ context.Context, level slog.Level) bool {
	return a.bl.GetLevel() >= FromSlogLevel(level)
}

func (a *baseLoggerAdapter) Handle(ctx context.Context, record slog.Record) error {
	args := make([]any, 0, 1+record.NumAttrs()*2)
	args = append(args, record.Message)
	record.Attrs(func(attr slog.Attr) bool {
		args = append(args, attr.Key, attr.Value.Any())
		return true
	})
	a.bl.Log(ctx, FromSlogLevel(record.Level), 0, args...)
	return nil
}

func (a *baseLoggerAdapter) WithAttrs(_ []slog.Attr) slog.Handler {
	return a
}

func (a *baseLoggerAdapter) WithGroup(_ string) slog.Handler {
	return a
}
