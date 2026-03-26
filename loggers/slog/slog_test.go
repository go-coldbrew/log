package slog

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/go-coldbrew/log/loggers"
)

// newBufferedLogger creates a logger writing to buf. Pass handlerOpts to
// customize the slog.HandlerOptions (e.g., for wire-compat ReplaceAttr).
// If handlerOpts is nil, a plain handler with no ReplaceAttr is used.
func newBufferedLogger(buf *bytes.Buffer, handlerOpts *slog.HandlerOptions, opts ...loggers.Option) loggers.BaseLogger {
	opt := loggers.GetDefaultOptions()
	for _, f := range opts {
		f(&opt)
	}

	levelVar := &slog.LevelVar{}
	levelVar.Set(toSlogLevel(opt.Level))

	if handlerOpts == nil {
		handlerOpts = &slog.HandlerOptions{AddSource: false, Level: levelVar}
	} else {
		handlerOpts.Level = levelVar
	}

	var handler slog.Handler
	if opt.JSONLogs {
		handler = slog.NewJSONHandler(buf, handlerOpts)
	} else {
		handler = slog.NewTextHandler(buf, handlerOpts)
	}

	return &logger{
		handler:  handler,
		levelVar: levelVar,
		opt:      opt,
	}
}

func wireCompatHandlerOpts(opt loggers.Options) *slog.HandlerOptions {
	return &slog.HandlerOptions{
		AddSource: false,
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
}

func newWireCompatLogger(buf *bytes.Buffer, opts ...loggers.Option) loggers.BaseLogger {
	opt := loggers.GetDefaultOptions()
	for _, f := range opts {
		f(&opt)
	}
	return newBufferedLogger(buf, wireCompatHandlerOpts(opt), opts...)
}

func TestNewLogger(t *testing.T) {
	l := NewLogger()
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
	var _ loggers.BaseLogger = l
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name     string
		level    loggers.Level
		logLevel loggers.Level
		expect   bool
	}{
		{"error at info level", loggers.ErrorLevel, loggers.InfoLevel, true},
		{"info at info level", loggers.InfoLevel, loggers.InfoLevel, true},
		{"debug at info level", loggers.DebugLevel, loggers.InfoLevel, false},
		{"warn at error level", loggers.WarnLevel, loggers.ErrorLevel, false},
		{"debug at debug level", loggers.DebugLevel, loggers.DebugLevel, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := newBufferedLogger(&buf, nil, loggers.WithLevel(tt.logLevel))
			l.Log(context.Background(), tt.level, 1, "msg", "test message")
			got := buf.String()
			if tt.expect && got == "" {
				t.Errorf("expected output but got none")
			}
			if !tt.expect && got != "" {
				t.Errorf("expected no output but got: %s", got)
			}
		})
	}
}

func TestContextFields(t *testing.T) {
	var buf bytes.Buffer
	l := newBufferedLogger(&buf, nil, loggers.WithJSONLogs(true), loggers.WithCallerInfo(false))

	ctx := context.Background()
	ctx = loggers.AddToLogContext(ctx, "request_id", "abc-123")
	ctx = loggers.AddToLogContext(ctx, "service", "test-svc")

	l.Log(ctx, loggers.InfoLevel, 1, "msg", "hello")

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nraw: %s", err, buf.String())
	}

	if m["request_id"] != "abc-123" {
		t.Errorf("expected request_id=abc-123, got %v", m["request_id"])
	}
	if m["service"] != "test-svc" {
		t.Errorf("expected service=test-svc, got %v", m["service"])
	}
}

func TestCallerInfo(t *testing.T) {
	var buf bytes.Buffer
	l := newBufferedLogger(&buf, nil, loggers.WithJSONLogs(true), loggers.WithCallerInfo(true))

	// skip=0 because we're calling the backend directly (no wrapper layer).
	l.Log(context.Background(), loggers.InfoLevel, 0, "msg", "with caller")

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, buf.String())
	}

	caller, ok := m["caller"].(string)
	if !ok || caller == "" {
		t.Errorf("expected caller field, got %v", m["caller"])
	}
	if !strings.Contains(caller, "slog_test.go") {
		t.Errorf("expected caller to contain slog_test.go, got %s", caller)
	}
}

func TestCallerInfoDisabled(t *testing.T) {
	var buf bytes.Buffer
	l := newBufferedLogger(&buf, nil, loggers.WithJSONLogs(true), loggers.WithCallerInfo(false))

	l.Log(context.Background(), loggers.InfoLevel, 1, "msg", "no caller")

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, buf.String())
	}

	if _, ok := m["caller"]; ok {
		t.Errorf("expected no caller field, but got one: %v", m["caller"])
	}
}

func TestSetLevel(t *testing.T) {
	var buf bytes.Buffer
	l := newBufferedLogger(&buf, nil, loggers.WithLevel(loggers.ErrorLevel))

	l.Log(context.Background(), loggers.InfoLevel, 1, "msg", "should not appear")
	if buf.Len() > 0 {
		t.Errorf("expected no output at info level with error-only logger, got: %s", buf.String())
	}

	l.SetLevel(loggers.DebugLevel)
	if l.GetLevel() != loggers.DebugLevel {
		t.Errorf("expected DebugLevel after SetLevel, got %v", l.GetLevel())
	}

	l.Log(context.Background(), loggers.InfoLevel, 1, "msg", "should appear now")
	if buf.Len() == 0 {
		t.Error("expected output after SetLevel to DebugLevel")
	}
}

func TestJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	l := newBufferedLogger(&buf, nil, loggers.WithJSONLogs(true), loggers.WithCallerInfo(false))

	l.Log(context.Background(), loggers.InfoLevel, 1, "msg", "json test", "key", "value")

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("expected valid JSON, got: %s", buf.String())
	}
	if m["msg"] != "json test" {
		t.Errorf("expected msg=json test, got %v", m["msg"])
	}
	if m["key"] != "value" {
		t.Errorf("expected key=value, got %v", m["key"])
	}
}

func TestTextOutput(t *testing.T) {
	var buf bytes.Buffer
	l := newBufferedLogger(&buf, nil, loggers.WithJSONLogs(false), loggers.WithCallerInfo(false))

	l.Log(context.Background(), loggers.InfoLevel, 1, "msg", "text test", "key", "value")

	out := buf.String()
	if !strings.Contains(out, "text test") {
		t.Errorf("expected output to contain 'text test', got: %s", out)
	}
	if !strings.Contains(out, "key=value") {
		t.Errorf("expected output to contain 'key=value', got: %s", out)
	}
}

func TestSingleArgMessage(t *testing.T) {
	var buf bytes.Buffer
	l := newBufferedLogger(&buf, nil, loggers.WithJSONLogs(true), loggers.WithCallerInfo(false))

	l.Log(context.Background(), loggers.InfoLevel, 1, "hello world")

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("expected valid JSON, got: %s", buf.String())
	}
	if m["msg"] != "hello world" {
		t.Errorf("expected msg='hello world', got %v", m["msg"])
	}
}

func TestNilContext(t *testing.T) {
	var buf bytes.Buffer
	l := newBufferedLogger(&buf, nil, loggers.WithJSONLogs(true), loggers.WithCallerInfo(false))

	l.Log(nil, loggers.InfoLevel, 1, "msg", "nil ctx") //nolint:staticcheck // testing nil context handling
	if buf.Len() == 0 {
		t.Error("expected output with nil context")
	}
}

func TestReentryGuard(t *testing.T) {
	var buf bytes.Buffer
	l := newBufferedLogger(&buf, nil, loggers.WithJSONLogs(true), loggers.WithCallerInfo(false))

	ctx := context.WithValue(context.Background(), slogBackendKey{}, true)
	l.Log(ctx, loggers.InfoLevel, 1, "msg", "should be dropped")

	if buf.Len() > 0 {
		t.Errorf("expected no output with re-entry guard, got: %s", buf.String())
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
		got := toSlogLevel(tt.cb)
		if got != tt.slog {
			t.Errorf("toSlogLevel(%v) = %v, want %v", tt.cb, got, tt.slog)
		}
	}
}

func TestNewLoggerWithHandler(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	l := NewLoggerWithHandler(handler, loggers.WithCallerInfo(false))

	l.Log(context.Background(), loggers.InfoLevel, 1, "msg", "custom handler")

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("expected valid JSON, got: %s", buf.String())
	}
	if m["msg"] != "custom handler" {
		t.Errorf("expected msg='custom handler', got %v", m["msg"])
	}
}

func TestWireCompatibility_TimestampKey(t *testing.T) {
	var buf bytes.Buffer
	l := newWireCompatLogger(&buf, loggers.WithCallerInfo(false))

	l.Log(context.Background(), loggers.InfoLevel, 1, "msg", "wire test")

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, buf.String())
	}

	if _, ok := m["@timestamp"]; !ok {
		t.Errorf("expected @timestamp key, got keys: %v", keys(m))
	}
	if _, ok := m["time"]; ok {
		t.Errorf("unexpected 'time' key — should be renamed to @timestamp")
	}
}

func TestWireCompatibility_LevelValues(t *testing.T) {
	tests := []struct {
		level    loggers.Level
		expected string
	}{
		{loggers.DebugLevel, "debug"},
		{loggers.InfoLevel, "info"},
		{loggers.WarnLevel, "warning"},
		{loggers.ErrorLevel, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			var buf bytes.Buffer
			l := newWireCompatLogger(&buf, loggers.WithCallerInfo(false), loggers.WithLevel(loggers.DebugLevel))
			l.Log(context.Background(), tt.level, 1, "msg", "level test")

			var m map[string]any
			if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
				t.Fatalf("failed to parse JSON: %v\nraw: %s", err, buf.String())
			}
			if m["level"] != tt.expected {
				t.Errorf("expected level=%q, got %q", tt.expected, m["level"])
			}
		})
	}
}

func TestWireCompatibility_LevelKey(t *testing.T) {
	var buf bytes.Buffer
	l := newWireCompatLogger(&buf, loggers.WithCallerInfo(false),
		loggers.WithLevelFieldName("severity"))

	l.Log(context.Background(), loggers.InfoLevel, 1, "msg", "custom level key")

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, buf.String())
	}

	if _, ok := m["severity"]; !ok {
		t.Errorf("expected 'severity' key, got keys: %v", keys(m))
	}
	if _, ok := m["level"]; ok {
		t.Errorf("unexpected 'level' key — should be renamed to 'severity'")
	}
}

func TestWireCompatibility_CustomTimestampKey(t *testing.T) {
	var buf bytes.Buffer
	l := newWireCompatLogger(&buf, loggers.WithCallerInfo(false),
		loggers.WithTimestampFieldName("ts"))

	l.Log(context.Background(), loggers.InfoLevel, 1, "msg", "custom ts key")

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, buf.String())
	}

	if _, ok := m["ts"]; !ok {
		t.Errorf("expected 'ts' key, got keys: %v", keys(m))
	}
}

func keys(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
