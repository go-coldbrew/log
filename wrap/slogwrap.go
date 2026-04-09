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
	l log.Logger
	// preformatted holds key-value pairs from WithAttrs, already resolved
	// with the groupPrefix that was active at the time WithAttrs was called.
	// This ensures attrs are not retroactively re-prefixed by later WithGroup calls.
	preformatted []any
	groups       []string
	groupPrefix  string // cached strings.Join(groups, ".") + "."
}

// Enabled reports whether the handler handles records at the given level.
// Respects per-request level overrides set via log.OverrideLogLevel.
// ColdBrew levels are inverted (Error=0 < Warn=1 < Info=2 < Debug=3),
// so >= means "configured level is at least as verbose as the message level."
func (h *slogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	cbLevel := h.l.GetLevel()
	if override, found := log.GetOverridenLogLevel(ctx); found {
		cbLevel = override
	}
	return cbLevel >= fromSlogLevel(level)
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

	// Use odd-leading form: message as first arg, then key-value pairs.
	// This is the universal convention across all backends (zap, gokit, slog).
	args := make([]any, 0, 1+len(h.preformatted)+record.NumAttrs()*2)
	args = append(args, record.Message)

	// Append pre-resolved attrs (keys already include their frozen group prefix).
	args = append(args, h.preformatted...)

	// Append record attrs with the current group prefix.
	record.Attrs(func(a slog.Attr) bool {
		args = appendAttr(args, h.groupPrefix, a, 0)
		return true
	})

	h.l.Log(ctx, cbLevel, slogHandlerSkip, args...)
	return nil
}

func (h *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	// Pre-resolve attrs with the current groupPrefix so they're frozen
	// and won't be affected by future WithGroup calls.
	var resolved []any
	for _, a := range attrs {
		resolved = appendAttr(resolved, h.groupPrefix, a, 0)
	}
	newPreformatted := make([]any, len(h.preformatted), len(h.preformatted)+len(resolved))
	copy(newPreformatted, h.preformatted)
	newPreformatted = append(newPreformatted, resolved...)
	return &slogHandler{l: h.l, preformatted: newPreformatted, groups: h.groups, groupPrefix: h.groupPrefix}
}

func (h *slogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	newGroups := make([]string, len(h.groups), len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups = append(newGroups, name)
	return &slogHandler{
		l:            h.l,
		preformatted: h.preformatted,
		groups:       newGroups,
		groupPrefix:  strings.Join(newGroups, ".") + ".",
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

// maxGroupDepth is the maximum nesting depth for slog group attributes.
// Beyond this depth, groups are replaced with a placeholder to prevent
// memory exhaustion from pathological input.
const maxGroupDepth = 10

// GroupDepthExceededPlaceholder is the value used when slog group nesting
// exceeds maxGroupDepth.
const GroupDepthExceededPlaceholder = "[nested group depth exceeded]"

// appendAttr flattens an slog.Attr into key-value pairs, applying a group prefix.
// depth tracks the current group nesting level to cap unbounded recursion.
func appendAttr(args []any, groupPrefix string, a slog.Attr, depth int) []any {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return args
	}

	key := a.Key
	if groupPrefix != "" {
		key = groupPrefix + key
	}

	if a.Value.Kind() == slog.KindGroup {
		if depth >= maxGroupDepth {
			return append(args, key, GroupDepthExceededPlaceholder)
		}
		groupAttrs := a.Value.Group()
		innerPrefix := groupPrefix
		if a.Key != "" {
			innerPrefix = groupPrefix + a.Key + "."
		}
		for _, ga := range groupAttrs {
			args = appendAttr(args, innerPrefix, ga, depth+1)
		}
		return args
	}

	return append(args, key, a.Value.Any())
}
