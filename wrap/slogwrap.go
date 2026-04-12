package wrap

import (
	"context"
	"log/slog"
	"strings"

	"github.com/go-coldbrew/log"
)

// ToSlogHandler wraps a ColdBrew log.Logger as an slog.Handler.
//
// Deprecated: With the slog-native architecture, use log.GetHandler() directly
// as it already implements slog.Handler with context field injection.
// This function is kept for backward compatibility with custom Logger implementations.
func ToSlogHandler(l log.Logger) slog.Handler {
	return &slogHandler{l: l}
}

// ToSlogLogger wraps a ColdBrew log.Logger as an *slog.Logger.
//
// Deprecated: With the slog-native architecture, the global slog.Logger
// is already configured via log.SetDefault. Use slog.Default() or
// slog.New(log.GetHandler()) instead.
func ToSlogLogger(l log.Logger) *slog.Logger {
	return slog.New(ToSlogHandler(l))
}

// slogBridgeKey is a context sentinel used to prevent infinite loops
// when both the slog handler bridge and slog backend are active.
type slogBridgeKey struct{}

// slogHandler implements slog.Handler, routing slog log calls into
// a ColdBrew log.Logger. This is the legacy bridge for custom Logger
// implementations that don't expose a Handler.
type slogHandler struct {
	l            log.Logger
	preformatted []any
	groups       []string
	groupPrefix  string
}

// Enabled reports whether the handler handles records at the given level.
func (h *slogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	cbLevel := h.l.GetLevel()
	if override, found := log.GetOverridenLogLevel(ctx); found {
		cbLevel = override
	}
	return cbLevel >= log.FromSlogLevel(level)
}

const slogHandlerSkip = 6

func (h *slogHandler) Handle(ctx context.Context, record slog.Record) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Value(slogBridgeKey{}) != nil {
		return nil
	}
	ctx = context.WithValue(ctx, slogBridgeKey{}, true)

	cbLevel := log.FromSlogLevel(record.Level)

	args := make([]any, 0, 1+len(h.preformatted)+record.NumAttrs()*2)
	args = append(args, record.Message)
	args = append(args, h.preformatted...)

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

const maxGroupDepth = 10

const groupDepthExceededPlaceholder = "[nested group depth exceeded]"

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
			return append(args, key, groupDepthExceededPlaceholder)
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
