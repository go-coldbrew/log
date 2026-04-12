package log

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/go-coldbrew/log/loggers"
)

// parseJSON parses a JSON log line from a buffer and resets it.
func parseJSON(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse JSON log: %v\nraw: %s", err, buf.String())
	}
	buf.Reset()
	return m
}

// setDefaultForTest sets the global ColdBrew handler and restores the
// previous state (both defaultHandler and slog.Default) on cleanup.
func setDefaultForTest(t *testing.T, h *Handler) {
	t.Helper()
	prevSlog := slog.Default()
	prevHandler := defaultHandler.Load()
	SetDefault(h)
	t.Cleanup(func() {
		slog.SetDefault(prevSlog)
		defaultHandler.Store(prevHandler)
	})
}

func TestNativeSlog_ContextFieldInjection(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := NewHandlerWithInner(inner, loggers.WithCallerInfo(false))
	setDefaultForTest(t, h)

	ctx := context.Background()
	ctx = AddToContext(ctx, "trace_id", "abc-123")
	ctx = AddToContext(ctx, "service", "test-svc")

	slog.InfoContext(ctx, "hello world")

	m := parseJSON(t, &buf)
	if m["trace_id"] != "abc-123" {
		t.Errorf("expected trace_id=abc-123, got %v", m["trace_id"])
	}
	if m["service"] != "test-svc" {
		t.Errorf("expected service=test-svc, got %v", m["service"])
	}
	if m["msg"] != "hello world" {
		t.Errorf("expected msg=hello world, got %v", m["msg"])
	}
}

func TestNativeSlog_OverrideLogLevel(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := NewHandlerWithInner(inner, loggers.WithLevel(loggers.InfoLevel), loggers.WithCallerInfo(false))
	setDefaultForTest(t, h)

	// Without override: debug should be filtered.
	slog.DebugContext(context.Background(), "should be filtered")
	if buf.Len() > 0 {
		t.Errorf("expected debug to be filtered, got: %s", buf.String())
	}

	// With override: debug should pass through.
	ctx := OverrideLogLevel(context.Background(), loggers.DebugLevel)
	slog.DebugContext(ctx, "should appear")
	if buf.Len() == 0 {
		t.Error("expected debug message with override to appear")
	}
	m := parseJSON(t, &buf)
	if m["msg"] != "should appear" {
		t.Errorf("expected msg=should appear, got %v", m["msg"])
	}
}

func TestHandler_WithAttrsPreservesContextInjection(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := NewHandlerWithInner(inner, loggers.WithCallerInfo(false))

	wrapped := h.WithAttrs([]slog.Attr{slog.String("static_key", "static_val")})

	ctx := context.Background()
	ctx = AddToContext(ctx, "trace_id", "xyz-789")

	r := slog.Record{}
	r.Level = slog.LevelInfo
	r.Message = "test with attrs"

	err := wrapped.Handle(ctx, r)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	m := parseJSON(t, &buf)
	if m["static_key"] != "static_val" {
		t.Errorf("expected static_key=static_val, got %v", m["static_key"])
	}
	if m["trace_id"] != "xyz-789" {
		t.Errorf("expected trace_id=xyz-789 from context, got %v", m["trace_id"])
	}
}

func TestHandler_WithGroupPreservesContextInjection(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := NewHandlerWithInner(inner, loggers.WithCallerInfo(false))

	grouped := h.WithGroup("app")

	ctx := context.Background()
	ctx = AddToContext(ctx, "trace_id", "grp-456")

	r := slog.Record{}
	r.Level = slog.LevelInfo
	r.Message = "grouped message"
	r.AddAttrs(slog.String("action", "deploy"))

	err := grouped.Handle(ctx, r)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	m := parseJSON(t, &buf)
	if m["msg"] != "grouped message" {
		t.Errorf("expected msg=grouped message, got %v", m["msg"])
	}
	appGroup, _ := m["app"].(map[string]any)
	if appGroup == nil {
		t.Fatal("expected app group in output")
	}
	if appGroup["trace_id"] != "grp-456" {
		t.Errorf("expected app.trace_id=grp-456 from context, got %v", appGroup["trace_id"])
	}
}

func TestHandler_EmptyWithAttrs(t *testing.T) {
	inner := slog.NewJSONHandler(&bytes.Buffer{}, nil)
	h := NewHandlerWithInner(inner)
	h2 := h.WithAttrs(nil)
	if h2 != h {
		t.Error("expected same handler for empty WithAttrs")
	}
}

func TestHandler_EmptyWithGroup(t *testing.T) {
	inner := slog.NewJSONHandler(&bytes.Buffer{}, nil)
	h := NewHandlerWithInner(inner)
	h2 := h.WithGroup("")
	if h2 != h {
		t.Error("expected same handler for empty WithGroup")
	}
}

func TestHandler_Inner(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, nil)
	h := NewHandlerWithInner(inner)
	if h.Inner() != inner {
		t.Error("Inner() should return the inner handler")
	}
}

func TestHandler_SetGetLevel(t *testing.T) {
	h := NewHandler()
	h.SetLevel(loggers.DebugLevel)
	if h.GetLevel() != loggers.DebugLevel {
		t.Errorf("expected DebugLevel, got %v", h.GetLevel())
	}
	h.SetLevel(loggers.ErrorLevel)
	if h.GetLevel() != loggers.ErrorLevel {
		t.Errorf("expected ErrorLevel, got %v", h.GetLevel())
	}
}

func TestCBLogger_ArgParsing(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := NewHandlerWithInner(inner, loggers.WithCallerInfo(false))
	l := &cbLogger{handler: h}

	// Test odd-leading form: first arg is message.
	l.Info(context.Background(), "hello", "key1", "val1")
	m := parseJSON(t, &buf)
	if m["msg"] != "hello" {
		t.Errorf("expected msg=hello, got %v", m["msg"])
	}
	if m["key1"] != "val1" {
		t.Errorf("expected key1=val1, got %v", m["key1"])
	}

	// Test even-length with explicit "msg" key.
	l.Info(context.Background(), "msg", "explicit message", "key2", "val2")
	m = parseJSON(t, &buf)
	if m["msg"] != "explicit message" {
		t.Errorf("expected msg=explicit message, got %v", m["msg"])
	}
	if m["key2"] != "val2" {
		t.Errorf("expected key2=val2, got %v", m["key2"])
	}

	// Test single arg.
	l.Info(context.Background(), "just a message")
	m = parseJSON(t, &buf)
	if m["msg"] != "just a message" {
		t.Errorf("expected msg=just a message, got %v", m["msg"])
	}
}

func TestSetDefault_EnablesNativeSlog(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := NewHandlerWithInner(inner, loggers.WithCallerInfo(false))
	setDefaultForTest(t, h)

	slog.Info("native slog call", "key", "value")
	m := parseJSON(t, &buf)
	if m["msg"] != "native slog call" {
		t.Errorf("expected msg=native slog call, got %v", m["msg"])
	}
	if m["key"] != "value" {
		t.Errorf("expected key=value, got %v", m["key"])
	}
}

func TestToAttr_SlogAttr(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := NewHandlerWithInner(inner, loggers.WithCallerInfo(false))
	setDefaultForTest(t, h)

	ctx := AddAttrsToContext(context.Background(), slog.String("user_id", "42"))
	slog.InfoContext(ctx, "test")

	m := parseJSON(t, &buf)
	if m["user_id"] != "42" {
		t.Errorf("expected user_id=42, got %v", m["user_id"])
	}
}

func TestToAttr_SlogAttr_MapKeyIgnored(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := NewHandlerWithInner(inner, loggers.WithCallerInfo(false))
	setDefaultForTest(t, h)

	ctx := context.Background()
	ctx = loggers.AddToLogContext(ctx, "ignored_key", slog.String("real_key", "value"))
	slog.InfoContext(ctx, "test")

	m := parseJSON(t, &buf)
	if m["real_key"] != "value" {
		t.Errorf("expected real_key=value, got %v", m["real_key"])
	}
	if _, exists := m["ignored_key"]; exists {
		t.Errorf("did not expect ignored_key in output, got %v", m["ignored_key"])
	}
}

func TestToAttr_SlogValue(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := NewHandlerWithInner(inner, loggers.WithCallerInfo(false))
	setDefaultForTest(t, h)

	ctx := context.Background()
	ctx = loggers.AddToLogContext(ctx, "count", slog.IntValue(99))
	slog.InfoContext(ctx, "test")

	m := parseJSON(t, &buf)
	if m["count"] != float64(99) {
		t.Errorf("expected count=99, got %v", m["count"])
	}
}

func TestAddAttrsToContext_Multiple(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := NewHandlerWithInner(inner, loggers.WithCallerInfo(false))
	setDefaultForTest(t, h)

	ctx := AddAttrsToContext(context.Background(),
		slog.String("trace_id", "abc-123"),
		slog.Int("user_id", 42),
	)
	slog.InfoContext(ctx, "handled")

	m := parseJSON(t, &buf)
	if m["trace_id"] != "abc-123" {
		t.Errorf("expected trace_id=abc-123, got %v", m["trace_id"])
	}
	if m["user_id"] != float64(42) {
		t.Errorf("expected user_id=42, got %v", m["user_id"])
	}
}

func TestAddAttrsToContext_Overwrites(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := NewHandlerWithInner(inner, loggers.WithCallerInfo(false))
	setDefaultForTest(t, h)

	ctx := AddToContext(context.Background(), "trace_id", "old-value")
	ctx = AddAttrsToContext(ctx, slog.String("trace_id", "new-value"))
	slog.InfoContext(ctx, "test")

	m := parseJSON(t, &buf)
	if m["trace_id"] != "new-value" {
		t.Errorf("expected trace_id=new-value, got %v", m["trace_id"])
	}
}

func TestAddAttrsToContext_EmptyKey(t *testing.T) {
	ctx := AddAttrsToContext(context.Background(), slog.Attr{Key: "", Value: slog.StringValue("skip")})
	fields := loggers.FromContext(ctx)
	if fields != nil {
		found := false
		fields.Range(func(k, _ any) bool {
			if k == "" {
				found = true
			}
			return true
		})
		if found {
			t.Error("expected empty-key attr to be skipped")
		}
	}
}

func TestAddAttrsToContext_Nil(t *testing.T) {
	ctx := AddAttrsToContext(nil, slog.String("key", "val")) //nolint:staticcheck // testing nil context
	if ctx == nil {
		t.Error("expected non-nil context")
	}
}

func TestLevelMapping(t *testing.T) {
	tests := []struct {
		cb   loggers.Level
		slog slog.Level
	}{
		{loggers.DebugLevel, slog.LevelDebug},
		{loggers.InfoLevel, slog.LevelInfo},
		{loggers.WarnLevel, slog.LevelWarn},
		{loggers.ErrorLevel, slog.LevelError},
	}

	for _, tt := range tests {
		if got := ToSlogLevel(tt.cb); got != tt.slog {
			t.Errorf("ToSlogLevel(%v) = %v, want %v", tt.cb, got, tt.slog)
		}
		if got := FromSlogLevel(tt.slog); got != tt.cb {
			t.Errorf("FromSlogLevel(%v) = %v, want %v", tt.slog, got, tt.cb)
		}
	}
}
