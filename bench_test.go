package log_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/go-coldbrew/log"
	"github.com/go-coldbrew/log/loggers"
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

func setupHandler(b *testing.B, inner slog.Handler, opts ...loggers.Option) {
	b.Helper()
	h := log.NewHandlerWithInner(inner, opts...)
	log.SetDefault(h)
	b.ResetTimer()
}

// --- Backend benchmarks: log.Info() with ColdBrew Handler ---

func BenchmarkBackend_Slog_JSON(b *testing.B) {
	inner := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	setupHandler(b, inner, jsonNoCaller...)
	ctx := context.Background()
	for b.Loop() {
		log.Info(ctx, "msg", "benchmark message", "key1", "value1", "key2", 42)
	}
}

func BenchmarkBackend_Slog_Text(b *testing.B) {
	inner := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	setupHandler(b, inner, loggers.WithJSONLogs(false), loggers.WithCallerInfo(false))
	ctx := context.Background()
	for b.Loop() {
		log.Info(ctx, "msg", "benchmark message", "key1", "value1", "key2", 42)
	}
}

// --- Backend benchmarks with caller info ---

func BenchmarkBackend_Slog_JSON_Caller(b *testing.B) {
	inner := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	setupHandler(b, inner, jsonWithCaller...)
	ctx := context.Background()
	for b.Loop() {
		log.Info(ctx, "msg", "benchmark message", "key1", "value1", "key2", 42)
	}
}

// --- Backend benchmarks with context fields ---

func BenchmarkBackend_Slog_JSON_CtxFields(b *testing.B) {
	inner := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	setupHandler(b, inner, jsonNoCaller...)
	ctx := context.Background()
	ctx = loggers.AddToLogContext(ctx, "trace_id", "abc-123")
	ctx = loggers.AddToLogContext(ctx, "service", "bench-svc")
	ctx = loggers.AddToLogContext(ctx, "request_id", "req-456")
	for b.Loop() {
		log.Info(ctx, "msg", "benchmark message", "key1", "value1", "key2", 42)
	}
}

// --- Backend benchmarks with typed context attrs ---

func BenchmarkBackend_Slog_JSON_CtxAttrs(b *testing.B) {
	inner := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	setupHandler(b, inner, jsonNoCaller...)
	ctx := context.Background()
	ctx = log.AddAttrsToContext(ctx,
		slog.String("trace_id", "abc-123"),
		slog.String("service", "bench-svc"),
		slog.String("request_id", "req-456"),
	)
	for b.Loop() {
		slog.InfoContext(ctx, "benchmark message", "key1", "value1", "key2", 42)
	}
}

// --- Native slog benchmarks: slog.InfoContext() through ColdBrew Handler ---

func BenchmarkFrontend_NativeSlog_ColdBrewHandler(b *testing.B) {
	inner := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := log.NewHandlerWithInner(inner, jsonNoCaller...)
	log.SetDefault(h)
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		slog.InfoContext(ctx, "benchmark message", "key1", "value1", "key2", 42)
	}
}

func BenchmarkFrontend_NativeSlog_ColdBrewHandler_CtxFields(b *testing.B) {
	inner := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := log.NewHandlerWithInner(inner, jsonNoCaller...)
	log.SetDefault(h)
	ctx := context.Background()
	ctx = loggers.AddToLogContext(ctx, "trace_id", "abc-123")
	ctx = loggers.AddToLogContext(ctx, "service", "bench-svc")
	ctx = loggers.AddToLogContext(ctx, "request_id", "req-456")
	b.ResetTimer()
	for b.Loop() {
		slog.InfoContext(ctx, "benchmark message", "key1", "value1", "key2", 42)
	}
}

// --- Baseline: native slog without ColdBrew (no overhead) ---

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

// --- Filtered (disabled level) benchmarks ---

func BenchmarkFiltered_Slog(b *testing.B) {
	inner := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	setupHandler(b, inner, jsonNoCaller...)
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
