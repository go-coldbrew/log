package wrap

import (
	"context"
	"log/slog"
	"strings"

	"github.com/go-coldbrew/log"
	"github.com/go-coldbrew/log/loggers"
)

// slogBridgeKey is a context sentinel used to prevent infinite loops
// when both the slog handler bridge and slog backend are active.
type slogBridgeKey struct{}

// slogHandler implements slog.Handler, routing slog log calls into
// a ColdBrew log.Logger. This allows third-party code and new code
// using slog natively to flow through ColdBrew's logging pipeline.
type slogHandler struct {
	l           log.Logger
	attrs       []slog.Attr
	groups      []string
	groupPrefix string // cached strings.Join(groups, ".") + "."
}

// Enabled reports whether the handler handles records at the given level.
// ColdBrew levels are inverted (Error=0 < Warn=1 < Info=2 < Debug=3),
// so >= means "configured level is at least as verbose as the message level."
func (h *slogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return h.l.GetLevel() >= fromSlogLevel(level)
}

// The skip value accounts for the call stack between Handle and the actual caller:
// baseLog.Log → wrapper.Log → Handle → slog.(*Logger).log → slog.Info → caller
const slogHandlerSkip = 6

func (h *slogHandler) Handle(ctx context.Context, record slog.Record) error {
	if ctx == nil {
		ctx = context.Background()
	}
	// Re-entry guard: prevent infinite loops with the slog backend.
	if ctx.Value(slogBridgeKey{}) != nil {
		return nil
	}
	ctx = context.WithValue(ctx, slogBridgeKey{}, true)

	cbLevel := fromSlogLevel(record.Level)

	args := make([]any, 0, 2+len(h.attrs)*2+record.NumAttrs()*2)
	args = append(args, "msg", record.Message)

	for _, a := range h.attrs {
		args = appendAttr(args, h.groupPrefix, a)
	}

	record.Attrs(func(a slog.Attr) bool {
		args = appendAttr(args, h.groupPrefix, a)
		return true
	})

	h.l.Log(ctx, cbLevel, slogHandlerSkip, args...)
	return nil
}

func (h *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	newAttrs := make([]slog.Attr, len(h.attrs), len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	newAttrs = append(newAttrs, attrs...)
	return &slogHandler{l: h.l, attrs: newAttrs, groups: h.groups, groupPrefix: h.groupPrefix}
}

func (h *slogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	newGroups := make([]string, len(h.groups), len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups = append(newGroups, name)
	return &slogHandler{
		l:           h.l,
		attrs:       h.attrs,
		groups:      newGroups,
		groupPrefix: strings.Join(newGroups, ".") + ".",
	}
}

// ToSlogHandler wraps a ColdBrew log.Logger as an slog.Handler.
func ToSlogHandler(l log.Logger) slog.Handler {
	return &slogHandler{l: l}
}

// ToSlogLogger wraps a ColdBrew log.Logger as an *slog.Logger.
func ToSlogLogger(l log.Logger) *slog.Logger {
	return slog.New(ToSlogHandler(l))
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

// appendAttr flattens an slog.Attr into key-value pairs, applying a group prefix.
func appendAttr(args []any, groupPrefix string, a slog.Attr) []any {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return args
	}

	key := a.Key
	if groupPrefix != "" {
		key = groupPrefix + key
	}

	if a.Value.Kind() == slog.KindGroup {
		groupAttrs := a.Value.Group()
		innerPrefix := groupPrefix
		if a.Key != "" {
			innerPrefix = groupPrefix + a.Key + "."
		}
		for _, ga := range groupAttrs {
			args = appendAttr(args, innerPrefix, ga)
		}
		return args
	}

	return append(args, key, a.Value.Any())
}
