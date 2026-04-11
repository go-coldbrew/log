package log

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/go-coldbrew/log/loggers"
)

// Handler implements slog.Handler with automatic ColdBrew context field injection.
// Any fields added via loggers.AddToLogContext (or log.AddToContext) are automatically
// included in every log record processed by this handler.
//
// Handler is composable — it can wrap any slog.Handler as its inner handler,
// and it can itself be wrapped by other slog.Handler implementations (e.g., slog-multi).
// WithAttrs and WithGroup return new *Handler instances that preserve context injection.
type Handler struct {
	inner       slog.Handler
	levelVar    *slog.LevelVar
	opts        loggers.Options
	callerCache sync.Map // pc (uintptr) → "file:line" (string)
}

var _ slog.Handler = (*Handler)(nil)

// NewHandler creates a new Handler with the default inner handler (slog.JSONHandler
// or slog.TextHandler based on options).
func NewHandler(options ...loggers.Option) *Handler {
	opt := applyOptions(options)

	levelVar := &slog.LevelVar{}
	levelVar.Set(ToSlogLevel(opt.Level))

	handlerOpts := &slog.HandlerOptions{
		AddSource: false,
		Level:     levelVar,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Key = opt.TimestampFieldName
			}
			if a.Key == slog.LevelKey {
				a.Key = opt.LevelFieldName
				if lvl, ok := a.Value.Any().(slog.Level); ok {
					a.Value = slog.StringValue(FromSlogLevel(lvl).String())
				}
			}
			return a
		},
	}

	var inner slog.Handler
	if opt.JSONLogs {
		inner = slog.NewJSONHandler(os.Stdout, handlerOpts)
	} else {
		inner = slog.NewTextHandler(os.Stdout, handlerOpts)
	}

	return &Handler{
		inner:    inner,
		levelVar: levelVar,
		opts:     opt,
	}
}

// NewHandlerWithInner creates a new Handler wrapping the provided slog.Handler.
// Use this to compose ColdBrew's context injection with custom handlers
// (e.g., slog-multi for fan-out, sampling handlers, or custom formatters).
//
// Example:
//
//	multi := slogmulti.Fanout(jsonHandler, textHandler)
//	h := log.NewHandlerWithInner(multi)
//	log.SetDefault(h)
func NewHandlerWithInner(inner slog.Handler, options ...loggers.Option) *Handler {
	if inner == nil {
		panic("log: NewHandlerWithInner called with nil inner handler")
	}
	opt := applyOptions(options)

	levelVar := &slog.LevelVar{}
	levelVar.Set(ToSlogLevel(opt.Level))

	return &Handler{
		inner:    inner,
		levelVar: levelVar,
		opts:     opt,
	}
}

func applyOptions(options []loggers.Option) loggers.Options {
	opt := loggers.GetDefaultOptions()
	for _, f := range options {
		f(&opt)
	}
	return opt
}

// Inner returns the wrapped slog.Handler.
func (h *Handler) Inner() slog.Handler {
	return h.inner
}

// Enabled reports whether the handler handles records at the given level.
// It checks both the configured level and any per-request level override
// set via OverrideLogLevel. This means per-request debug logging works
// even for native slog.DebugContext calls.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	cbLevel := FromSlogLevel(h.levelVar.Level())
	msgLevel := FromSlogLevel(level)

	// Fast path: base level permits this message.
	if cbLevel >= msgLevel {
		return true
	}

	// Per-request override takes precedence over both ColdBrew's level and the
	// inner handler's level — this is what makes OverrideLogLevel work.
	if ctx != nil {
		if override, found := GetOverridenLogLevel(ctx); found {
			return override >= msgLevel
		}
	}

	return false
}

// Handle processes the log record, injecting ColdBrew context fields and caller info,
// then delegates to the inner handler.
func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	if ctx == nil {
		ctx = context.Background()
	}

	// Inject caller info if configured.
	if h.opts.CallerInfo && record.PC != 0 {
		callerStr := h.cachedCallerInfo(record.PC)
		record.AddAttrs(slog.String(h.opts.CallerFieldName, callerStr))
	}

	// Inject context fields from AddToLogContext.
	ctxFields := loggers.FromContext(ctx)
	if ctxFields != nil {
		ctxFields.Range(func(k, v any) bool {
			record.AddAttrs(toAttr(stringKey(k), v))
			return true
		})
	}

	return h.inner.Handle(ctx, record)
}

// cloneWithInner returns a new Handler sharing level and options but with a different inner handler.
func (h *Handler) cloneWithInner(inner slog.Handler) *Handler {
	return &Handler{
		inner:    inner,
		levelVar: h.levelVar,
		opts:     h.opts,
	}
}

// WithAttrs returns a new Handler with the given attributes pre-applied.
// The returned handler preserves ColdBrew context field injection.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return h.cloneWithInner(h.inner.WithAttrs(attrs))
}

// WithGroup returns a new Handler with the given group name.
// The returned handler preserves ColdBrew context field injection.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return h.cloneWithInner(h.inner.WithGroup(name))
}

// SetLevel changes the log level dynamically.
// If the inner handler supports SetLevel (e.g., the BaseLogger adapter),
// the level change is propagated.
func (h *Handler) SetLevel(level loggers.Level) {
	h.levelVar.Set(ToSlogLevel(level))

	type levelSetter interface{ SetLevel(loggers.Level) }
	if inner, ok := h.inner.(levelSetter); ok {
		inner.SetLevel(level)
	}
}

// GetLevel returns the current log level.
func (h *Handler) GetLevel() loggers.Level {
	return FromSlogLevel(h.levelVar.Level())
}

// cachedCallerInfo returns a "file:line" string for the given program counter,
// using a per-handler cache to avoid repeated frame resolution.
func (h *Handler) cachedCallerInfo(pc uintptr) string {
	if v, ok := h.callerCache.Load(pc); ok {
		return v.(string)
	}
	frames := runtime.CallersFrames([]uintptr{pc})
	f, _ := frames.Next()
	file := f.File
	depth := h.opts.CallerFileDepth
	if depth <= 0 {
		depth = 2
	}
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			depth--
			if depth == 0 {
				file = file[i+1:]
				break
			}
		}
	}
	s := file + ":" + strconv.Itoa(f.Line)
	actual, _ := h.callerCache.LoadOrStore(pc, s)
	return actual.(string)
}

func stringKey(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

// toAttr converts a key-value pair into an slog.Attr.
// When val is an slog.Attr, it is returned as-is (using its own key).
// When val is an slog.Value, the provided key is used.
func toAttr(key string, val any) slog.Attr {
	switch v := val.(type) {
	case slog.Attr:
		return v
	case slog.Value:
		return slog.Attr{Key: key, Value: v}
	case time.Duration:
		return slog.String(key, v.String())
	default:
		return slog.Any(key, val)
	}
}

// ToSlogLevel converts a ColdBrew log level to an slog.Level.
func ToSlogLevel(level loggers.Level) slog.Level {
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

// FromSlogLevel converts an slog.Level to a ColdBrew log level.
func FromSlogLevel(level slog.Level) loggers.Level {
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
