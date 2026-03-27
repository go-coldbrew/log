package log_test

import (
	"context"
	"io"
	stdlogpkg "log"
	"log/slog"
	"os"
	"testing"

	"github.com/go-coldbrew/log"
	"github.com/go-coldbrew/log/loggers"
	"github.com/go-coldbrew/log/loggers/gokit"
	"github.com/go-coldbrew/log/loggers/logrus"
	cbslog "github.com/go-coldbrew/log/loggers/slog"
	"github.com/go-coldbrew/log/loggers/stdlog"
	"github.com/go-coldbrew/log/loggers/zap"
	"github.com/go-coldbrew/log/wrap"
)

// Common options: JSON output, no caller info (to isolate serialization cost).
var (
	jsonNoCaller = []loggers.Option{
		loggers.WithJSONLogs(true),
		loggers.WithCallerInfo(false),
	}
	jsonWithCaller = []loggers.Option{
		loggers.WithJSONLogs(true),
		loggers.WithCallerInfo(true),
	}
)

// discardStdout redirects os.Stdout to /dev/null for the duration of a
// benchmark. This is necessary for backends (gokit, zap, logrus, stdlog)
// that unconditionally write to os.Stdout.
func discardStdout(b *testing.B) {
	b.Helper()
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatalf("failed to open /dev/null: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = devNull
	b.Cleanup(func() {
		os.Stdout = origStdout
		devNull.Close()
	})
}

func setupLogger(b *testing.B, backend loggers.BaseLogger) {
	b.Helper()
	log.SetLogger(log.NewLogger(backend))
	b.ResetTimer()
}

// --- Backend benchmarks: log.Info() with each backend ---

func BenchmarkBackend_Gokit_JSON(b *testing.B) {
	discardStdout(b)
	setupLogger(b, gokit.NewLogger(jsonNoCaller...))
	ctx := context.Background()
	for b.Loop() {
		log.Info(ctx, "msg", "benchmark message", "key1", "value1", "key2", 42)
	}
}

func BenchmarkBackend_Zap_JSON(b *testing.B) {
	discardStdout(b)
	setupLogger(b, zap.NewLogger(jsonNoCaller...))
	ctx := context.Background()
	for b.Loop() {
		log.Info(ctx, "msg", "benchmark message", "key1", "value1", "key2", 42)
	}
}

func BenchmarkBackend_Logrus_JSON(b *testing.B) {
	discardStdout(b)
	setupLogger(b, logrus.NewLogger(jsonNoCaller...))
	ctx := context.Background()
	for b.Loop() {
		log.Info(ctx, "msg", "benchmark message", "key1", "value1", "key2", 42)
	}
}

func BenchmarkBackend_Slog_JSON(b *testing.B) {
	handler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	setupLogger(b, cbslog.NewLoggerWithHandler(handler, jsonNoCaller...))
	ctx := context.Background()
	for b.Loop() {
		log.Info(ctx, "msg", "benchmark message", "key1", "value1", "key2", 42)
	}
}

func BenchmarkBackend_Slog_Text(b *testing.B) {
	handler := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	setupLogger(b, cbslog.NewLoggerWithHandler(handler, loggers.WithJSONLogs(false), loggers.WithCallerInfo(false)))
	ctx := context.Background()
	for b.Loop() {
		log.Info(ctx, "msg", "benchmark message", "key1", "value1", "key2", 42)
	}
}

func BenchmarkBackend_Stdlog(b *testing.B) {
	discardStdout(b)
	origWriter := stdlogpkg.Writer()
	stdlogpkg.SetOutput(io.Discard)
	b.Cleanup(func() { stdlogpkg.SetOutput(origWriter) })
	setupLogger(b, stdlog.NewLogger())
	ctx := context.Background()
	for b.Loop() {
		log.Info(ctx, "msg", "benchmark message", "key1", "value1", "key2", 42)
	}
}

// --- Backend benchmarks with caller info ---

func BenchmarkBackend_Gokit_JSON_Caller(b *testing.B) {
	discardStdout(b)
	setupLogger(b, gokit.NewLogger(jsonWithCaller...))
	ctx := context.Background()
	for b.Loop() {
		log.Info(ctx, "msg", "benchmark message", "key1", "value1", "key2", 42)
	}
}

func BenchmarkBackend_Slog_JSON_Caller(b *testing.B) {
	handler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	setupLogger(b, cbslog.NewLoggerWithHandler(handler, jsonWithCaller...))
	ctx := context.Background()
	for b.Loop() {
		log.Info(ctx, "msg", "benchmark message", "key1", "value1", "key2", 42)
	}
}

// --- Backend benchmarks with context fields ---

func BenchmarkBackend_Gokit_JSON_CtxFields(b *testing.B) {
	discardStdout(b)
	setupLogger(b, gokit.NewLogger(jsonNoCaller...))
	ctx := context.Background()
	ctx = loggers.AddToLogContext(ctx, "trace_id", "abc-123")
	ctx = loggers.AddToLogContext(ctx, "service", "bench-svc")
	ctx = loggers.AddToLogContext(ctx, "request_id", "req-456")
	for b.Loop() {
		log.Info(ctx, "msg", "benchmark message", "key1", "value1", "key2", 42)
	}
}

func BenchmarkBackend_Slog_JSON_CtxFields(b *testing.B) {
	handler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	setupLogger(b, cbslog.NewLoggerWithHandler(handler, jsonNoCaller...))
	ctx := context.Background()
	ctx = loggers.AddToLogContext(ctx, "trace_id", "abc-123")
	ctx = loggers.AddToLogContext(ctx, "service", "bench-svc")
	ctx = loggers.AddToLogContext(ctx, "request_id", "req-456")
	for b.Loop() {
		log.Info(ctx, "msg", "benchmark message", "key1", "value1", "key2", 42)
	}
}

func BenchmarkBackend_Zap_JSON_CtxFields(b *testing.B) {
	discardStdout(b)
	setupLogger(b, zap.NewLogger(jsonNoCaller...))
	ctx := context.Background()
	ctx = loggers.AddToLogContext(ctx, "trace_id", "abc-123")
	ctx = loggers.AddToLogContext(ctx, "service", "bench-svc")
	ctx = loggers.AddToLogContext(ctx, "request_id", "req-456")
	for b.Loop() {
		log.Info(ctx, "msg", "benchmark message", "key1", "value1", "key2", 42)
	}
}

// --- Frontend benchmarks: slog.Info() through the bridge ---

func BenchmarkFrontend_SlogBridge_GokitBackend(b *testing.B) {
	discardStdout(b)
	log.SetLogger(log.NewLogger(gokit.NewLogger(jsonNoCaller...)))
	sl := wrap.ToSlogLogger(log.GetLogger())
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		sl.InfoContext(ctx, "benchmark message", "key1", "value1", "key2", 42)
	}
}

func BenchmarkFrontend_SlogBridge_SlogBackend(b *testing.B) {
	handler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	log.SetLogger(log.NewLogger(cbslog.NewLoggerWithHandler(handler, jsonNoCaller...)))
	sl := wrap.ToSlogLogger(log.GetLogger())
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		sl.InfoContext(ctx, "benchmark message", "key1", "value1", "key2", 42)
	}
}

func BenchmarkFrontend_SlogBridge_WithAttrs(b *testing.B) {
	handler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	log.SetLogger(log.NewLogger(cbslog.NewLoggerWithHandler(handler, jsonNoCaller...)))
	sl := wrap.ToSlogLogger(log.GetLogger()).With("service", "bench-svc", "version", "1.0")
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		sl.InfoContext(ctx, "benchmark message", "key1", "value1")
	}
}

func BenchmarkFrontend_SlogBridge_WithGroup(b *testing.B) {
	handler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	log.SetLogger(log.NewLogger(cbslog.NewLoggerWithHandler(handler, jsonNoCaller...)))
	sl := wrap.ToSlogLogger(log.GetLogger()).WithGroup("request")
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		sl.InfoContext(ctx, "benchmark message", "method", "GET", "path", "/api/v1")
	}
}

// --- Frontend benchmark: native slog (baseline, no ColdBrew overhead) ---

func BenchmarkFrontend_NativeSlog_JSON(b *testing.B) {
	handler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	sl := slog.New(handler)
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		sl.InfoContext(ctx, "benchmark message", "key1", "value1", "key2", 42)
	}
}

func BenchmarkFrontend_NativeSlog_Text(b *testing.B) {
	handler := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	sl := slog.New(handler)
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		sl.InfoContext(ctx, "benchmark message", "key1", "value1", "key2", 42)
	}
}

// --- Frontend benchmark: go-kit wrapped (existing pattern) ---

func BenchmarkFrontend_GoKitWrap(b *testing.B) {
	discardStdout(b)
	log.SetLogger(log.NewLogger(gokit.NewLogger(jsonNoCaller...)))
	gk := wrap.ToGoKitLogger(log.GetLogger())
	b.ResetTimer()
	for b.Loop() {
		_ = gk.Log("msg", "benchmark message", "key1", "value1", "key2", 42)
	}
}

// --- Filtered (disabled level) benchmarks ---

func BenchmarkFiltered_Gokit(b *testing.B) {
	discardStdout(b)
	setupLogger(b, gokit.NewLogger(jsonNoCaller...))
	log.SetLevel(loggers.ErrorLevel)
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		log.Debug(ctx, "msg", "should be filtered")
	}
}

func BenchmarkFiltered_Slog(b *testing.B) {
	handler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	setupLogger(b, cbslog.NewLoggerWithHandler(handler, jsonNoCaller...))
	log.SetLevel(loggers.ErrorLevel)
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		log.Debug(ctx, "msg", "should be filtered")
	}
}

func BenchmarkFiltered_NativeSlog(b *testing.B) {
	handler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})
	sl := slog.New(handler)
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		sl.DebugContext(ctx, "should be filtered")
	}
}