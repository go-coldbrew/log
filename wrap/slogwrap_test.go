package wrap

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/go-coldbrew/log"
	"github.com/go-coldbrew/log/loggers"
	cbslog "github.com/go-coldbrew/log/loggers/slog"
)

// captureLogger is a mock BaseLogger that records log calls for inspection.
type captureLogger struct {
	mu      sync.Mutex
	entries []capturedEntry
	level   loggers.Level
}

type capturedEntry struct {
	Level loggers.Level
	Args  []any
}

func (c *captureLogger) Log(_ context.Context, level loggers.Level, _ int, args ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = append(c.entries, capturedEntry{Level: level, Args: args})
}

func (c *captureLogger) SetLevel(level loggers.Level) { c.level = level }
func (c *captureLogger) GetLevel() loggers.Level       { return c.level }

func (c *captureLogger) lastEntry() capturedEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) == 0 {
		return capturedEntry{}
	}
	return c.entries[len(c.entries)-1]
}

func (c *captureLogger) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

func newCaptureLogger(level loggers.Level) (*captureLogger, log.Logger) {
	cl := &captureLogger{level: level}
	return cl, log.NewLogger(cl)
}

func argsContain(args []any, key, value string) bool {
	// Bridge uses odd-leading form: args[0] is message, pairs start at index 1.
	start := 0
	if len(args)%2 != 0 {
		start = 1
	}
	for i := start; i < len(args)-1; i += 2 {
		k, _ := args[i].(string)
		if k == key && fmt.Sprint(args[i+1]) == value {
			return true
		}
	}
	return false
}

func TestToSlogHandler(t *testing.T) {
	_, l := newCaptureLogger(loggers.DebugLevel)
	h := ToSlogHandler(l)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	var _ slog.Handler = h
}

func TestToSlogLogger(t *testing.T) {
	_, l := newCaptureLogger(loggers.DebugLevel)
	sl := ToSlogLogger(l)
	if sl == nil {
		t.Fatal("expected non-nil slog.Logger")
	}
}

func TestHandleBasic(t *testing.T) {
	cl, l := newCaptureLogger(loggers.DebugLevel)
	h := ToSlogHandler(l)

	var record slog.Record
	record.Level = slog.LevelInfo
	record.Message = "hello"
	record.AddAttrs(slog.String("key", "value"))

	err := h.Handle(context.Background(), record)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	entry := cl.lastEntry()
	if entry.Level != loggers.InfoLevel {
		t.Errorf("expected InfoLevel, got %v", entry.Level)
	}
	// Message is in leading position (odd-leading form).
	if len(entry.Args) == 0 || fmt.Sprint(entry.Args[0]) != "hello" {
		t.Errorf("expected leading message 'hello', got %v", entry.Args)
	}
	if !argsContain(entry.Args, "key", "value") {
		t.Errorf("expected key=value in args, got %v", entry.Args)
	}
}

func TestEnabled(t *testing.T) {
	_, l := newCaptureLogger(loggers.InfoLevel)
	h := ToSlogHandler(l)

	if !h.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("expected Info to be enabled at InfoLevel")
	}
	if !h.Enabled(context.Background(), slog.LevelError) {
		t.Error("expected Error to be enabled at InfoLevel")
	}
	if h.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("expected Debug to be disabled at InfoLevel")
	}
}

func TestWithAttrs(t *testing.T) {
	cl, l := newCaptureLogger(loggers.DebugLevel)
	h := ToSlogHandler(l)

	h2 := h.WithAttrs([]slog.Attr{slog.String("service", "test-svc")})

	record := slog.Record{}
	record.Level = slog.LevelInfo
	record.Message = "with attrs"

	_ = h2.Handle(context.Background(), record)

	entry := cl.lastEntry()
	if !argsContain(entry.Args, "service", "test-svc") {
		t.Errorf("expected service=test-svc from WithAttrs, got %v", entry.Args)
	}
}

func TestWithAttrsEmpty(t *testing.T) {
	_, l := newCaptureLogger(loggers.DebugLevel)
	h := ToSlogHandler(l)
	h2 := h.WithAttrs(nil)
	if h2 != h {
		t.Error("expected same handler for empty WithAttrs")
	}
}

func TestWithGroup(t *testing.T) {
	cl, l := newCaptureLogger(loggers.DebugLevel)
	h := ToSlogHandler(l)

	h2 := h.WithGroup("request")

	record := slog.Record{}
	record.Level = slog.LevelInfo
	record.Message = "grouped"
	record.AddAttrs(slog.String("id", "abc-123"))

	_ = h2.Handle(context.Background(), record)

	entry := cl.lastEntry()
	if !argsContain(entry.Args, "request.id", "abc-123") {
		t.Errorf("expected request.id=abc-123, got %v", entry.Args)
	}
}

func TestWithGroupEmpty(t *testing.T) {
	_, l := newCaptureLogger(loggers.DebugLevel)
	h := ToSlogHandler(l)
	h2 := h.WithGroup("")
	if h2 != h {
		t.Error("expected same handler for empty WithGroup")
	}
}

func TestNestedGroups(t *testing.T) {
	cl, l := newCaptureLogger(loggers.DebugLevel)
	h := ToSlogHandler(l)

	h2 := h.WithGroup("http").WithGroup("request")

	record := slog.Record{}
	record.Level = slog.LevelInfo
	record.Message = "nested"
	record.AddAttrs(slog.String("method", "GET"))

	_ = h2.Handle(context.Background(), record)

	entry := cl.lastEntry()
	if !argsContain(entry.Args, "http.request.method", "GET") {
		t.Errorf("expected http.request.method=GET, got %v", entry.Args)
	}
}

func TestWithGroupAndAttrs(t *testing.T) {
	cl, l := newCaptureLogger(loggers.DebugLevel)
	h := ToSlogHandler(l)

	h2 := h.WithGroup("app").WithAttrs([]slog.Attr{slog.String("version", "1.0")})

	record := slog.Record{}
	record.Level = slog.LevelInfo
	record.Message = "versioned"
	record.AddAttrs(slog.String("action", "deploy"))

	_ = h2.Handle(context.Background(), record)

	entry := cl.lastEntry()
	if !argsContain(entry.Args, "app.version", "1.0") {
		t.Errorf("expected app.version=1.0, got %v", entry.Args)
	}
	if !argsContain(entry.Args, "app.action", "deploy") {
		t.Errorf("expected app.action=deploy, got %v", entry.Args)
	}
}

func TestLevelMapping(t *testing.T) {
	tests := []struct {
		slogLevel slog.Level
		cbLevel   loggers.Level
	}{
		{slog.LevelDebug, loggers.DebugLevel},
		{slog.LevelInfo, loggers.InfoLevel},
		{slog.LevelWarn, loggers.WarnLevel},
		{slog.LevelError, loggers.ErrorLevel},
	}

	for _, tt := range tests {
		got := fromSlogLevel(tt.slogLevel)
		if got != tt.cbLevel {
			t.Errorf("fromSlogLevel(%v) = %v, want %v", tt.slogLevel, got, tt.cbLevel)
		}
	}
}

func TestLevelMappingNonStandard(t *testing.T) {
	// Levels between standard values should map to the lower bucket.
	if fromSlogLevel(slog.LevelInfo+2) != loggers.InfoLevel {
		t.Error("expected Info+2 to map to InfoLevel")
	}
	if fromSlogLevel(slog.LevelDebug-4) != loggers.DebugLevel {
		t.Error("expected Debug-4 to map to DebugLevel")
	}
}

func TestReentryGuard(t *testing.T) {
	cl, l := newCaptureLogger(loggers.DebugLevel)
	h := ToSlogHandler(l)

	// Simulate re-entry by setting the sentinel key.
	ctx := context.WithValue(context.Background(), slogBridgeKey{}, true)

	record := slog.Record{}
	record.Level = slog.LevelInfo
	record.Message = "should be dropped"

	err := h.Handle(ctx, record)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if cl.count() > 0 {
		t.Error("expected no log entries with re-entry guard")
	}
}

func TestGroupAttrFlattening(t *testing.T) {
	cl, l := newCaptureLogger(loggers.DebugLevel)
	h := ToSlogHandler(l)

	record := slog.Record{}
	record.Level = slog.LevelInfo
	record.Message = "group attr"
	record.AddAttrs(slog.Group("server",
		slog.String("host", "localhost"),
		slog.Int("port", 8080),
	))

	_ = h.Handle(context.Background(), record)

	entry := cl.lastEntry()

	// Find the args as strings.
	var pairs []string
	start := 0
	if len(entry.Args)%2 != 0 {
		start = 1
	}
	for i := start; i < len(entry.Args)-1; i += 2 {
		k, _ := entry.Args[i].(string)
		pairs = append(pairs, k)
	}

	found := false
	for _, p := range pairs {
		if strings.HasPrefix(p, "server.") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected server.* prefixed keys, got pairs: %v", pairs)
	}
}

func TestNilContext(t *testing.T) {
	cl, l := newCaptureLogger(loggers.DebugLevel)
	h := ToSlogHandler(l)

	record := slog.Record{}
	record.Level = slog.LevelInfo
	record.Message = "nil ctx"

	// Should not panic.
	err := h.Handle(nil, record) //nolint:staticcheck // testing nil context handling
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if cl.count() == 0 {
		t.Error("expected log entry with nil context")
	}
}

func TestImmutability(t *testing.T) {
	cl, l := newCaptureLogger(loggers.DebugLevel)
	h := ToSlogHandler(l)

	h1 := h.WithAttrs([]slog.Attr{slog.String("a", "1")})
	h2 := h.WithAttrs([]slog.Attr{slog.String("b", "2")})

	// h1 and h2 should be independent — log from each and check.
	var r slog.Record
	r.Level = slog.LevelInfo
	r.Message = "h1"
	_ = h1.Handle(context.Background(), r)
	e1 := cl.lastEntry()

	r.Message = "h2"
	_ = h2.Handle(context.Background(), r)
	e2 := cl.lastEntry()

	if !argsContain(e1.Args, "a", "1") || argsContain(e1.Args, "b", "2") {
		t.Errorf("h1 should have a=1 only, got %v", e1.Args)
	}
	if !argsContain(e2.Args, "b", "2") || argsContain(e2.Args, "a", "1") {
		t.Errorf("h2 should have b=2 only, got %v", e2.Args)
	}
}

// Regression: attrs added before WithGroup must NOT be prefixed by that group.
func TestWithAttrsThenWithGroup(t *testing.T) {
	cl, l := newCaptureLogger(loggers.DebugLevel)
	h := ToSlogHandler(l)

	// Add attr first, then group — "service" should stay at top level.
	h2 := h.WithAttrs([]slog.Attr{slog.String("service", "api")}).WithGroup("req")

	var r slog.Record
	r.Level = slog.LevelInfo
	r.Message = "test"
	r.AddAttrs(slog.String("id", "123"))
	_ = h2.Handle(context.Background(), r)

	entry := cl.lastEntry()
	// "service" was added before the group — must NOT be prefixed.
	if !argsContain(entry.Args, "service", "api") {
		t.Errorf("expected top-level service=api, got %v", entry.Args)
	}
	if argsContain(entry.Args, "req.service", "api") {
		t.Errorf("service should NOT be prefixed with req., got %v", entry.Args)
	}
	// "id" is a record attr — should be prefixed with the group.
	if !argsContain(entry.Args, "req.id", "123") {
		t.Errorf("expected req.id=123, got %v", entry.Args)
	}
}

func TestWithGroupThenWithAttrs(t *testing.T) {
	cl, l := newCaptureLogger(loggers.DebugLevel)
	h := ToSlogHandler(l)

	// Group first, then attr — "version" should be prefixed.
	h2 := h.WithGroup("app").WithAttrs([]slog.Attr{slog.String("version", "2.0")})

	var r slog.Record
	r.Level = slog.LevelInfo
	r.Message = "test"
	r.AddAttrs(slog.String("action", "deploy"))
	_ = h2.Handle(context.Background(), r)

	entry := cl.lastEntry()
	if !argsContain(entry.Args, "app.version", "2.0") {
		t.Errorf("expected app.version=2.0, got %v", entry.Args)
	}
	if !argsContain(entry.Args, "app.action", "deploy") {
		t.Errorf("expected app.action=deploy, got %v", entry.Args)
	}
}

func TestAppendAttr_DeepNesting(t *testing.T) {
	cl, l := newCaptureLogger(loggers.DebugLevel)
	h := ToSlogHandler(l)

	// Build nesting deeper than maxGroupDepth to verify the cap.
	testDepth := maxGroupDepth + 5
	attr := slog.String("leaf", "deep")
	for i := testDepth - 1; i >= 0; i-- {
		attr = slog.Group(fmt.Sprintf("g%d", i), attr)
	}

	var r slog.Record
	r.Level = slog.LevelInfo
	r.Message = "deep nesting test"
	r.AddAttrs(attr)
	_ = h.Handle(context.Background(), r)

	entry := cl.lastEntry()

	// Groups beyond maxGroupDepth should be capped with the placeholder.
	// The leaf value "deep" should NOT appear at all since it's beyond the cap.
	foundPlaceholder := false
	foundLeafValue := false
	start := 0
	if len(entry.Args)%2 != 0 {
		start = 1
	}
	for i := start; i < len(entry.Args)-1; i += 2 {
		v := fmt.Sprint(entry.Args[i+1])
		if v == groupDepthExceededPlaceholder {
			foundPlaceholder = true
		}
		if v == "deep" {
			foundLeafValue = true
		}
	}

	if !foundPlaceholder {
		t.Errorf("expected depth-exceeded placeholder in args, got: %v", entry.Args)
	}
	if foundLeafValue {
		t.Errorf("leaf value should not appear when nesting exceeds maxGroupDepth (%d), got: %v", maxGroupDepth, entry.Args)
	}
}

// TestReentryGuardIntegration verifies that having both the slog backend AND
// the slog bridge active simultaneously does not cause an infinite loop.
func TestReentryGuardIntegration(t *testing.T) {
	handler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	log.SetLogger(log.NewLogger(cbslog.NewLoggerWithHandler(handler, loggers.WithCallerInfo(false))))

	sl := ToSlogLogger(log.GetLogger())

	// This would infinite-loop without re-entry guards:
	// sl.Info → bridge.Handle → log.Logger.Log → slog backend.Log → (guard breaks)
	sl.InfoContext(context.Background(), "should not loop", "key", "value")

	// Reverse: ColdBrew log through slog backend while bridge is slog default.
	slog.SetDefault(sl)
	log.Info(context.Background(), "msg", "reverse direction", "key", "value")

	// If we got here without hanging, the guards work.
}
