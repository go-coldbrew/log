package log

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/go-coldbrew/log/loggers"
)

type countingLogger struct {
	count atomic.Int64
	level loggers.Level
}

func (c *countingLogger) Log(_ context.Context, _ loggers.Level, _ int, _ ...any) {
	c.count.Add(1)
}
func (c *countingLogger) SetLevel(l loggers.Level) { c.level = l }
func (c *countingLogger) GetLevel() loggers.Level  { return c.level }

// Tests OverrideLogLevel
func TestOverrideLogLevel(t *testing.T) {
	// Set the log level to debug
	ctx := context.Background()
	_, found := GetOverridenLogLevel(ctx)

	if found {
		t.Error("Expected not to find the log level in the context")
	}

	ctx = OverrideLogLevel(ctx, loggers.DebugLevel)

	// Get the log level from the context
	level, found := GetOverridenLogLevel(ctx)
	if !found {
		t.Error("Expected to find the log level in the context")
	}
	if level != loggers.DebugLevel {
		t.Error("Expected the log level to be debug")
	}
}

// TestOverrideLogLevelCausesLogging verifies that a per-request override
// causes a message to be logged even when the global level would filter it.
func TestOverrideLogLevelCausesLogging(t *testing.T) {
	cl := &countingLogger{level: loggers.ErrorLevel}
	l := NewLogger(cl)
	SetLogger(l)

	ctx := context.Background()

	// Without override: debug message should be filtered (global level is Error).
	l.Log(ctx, loggers.DebugLevel, 0, "msg", "should be filtered")
	if cl.count.Load() != 0 {
		t.Error("expected debug message to be filtered at Error level")
	}

	// With override: debug message should pass through.
	ctx = OverrideLogLevel(ctx, loggers.DebugLevel)
	l.Log(ctx, loggers.DebugLevel, 0, "msg", "should pass through")
	if cl.count.Load() != 1 {
		t.Errorf("expected 1 log call with override, got %d", cl.count.Load())
	}

	// Info without override should still be filtered.
	plainCtx := context.Background()
	l.Log(plainCtx, loggers.InfoLevel, 0, "msg", "still filtered")
	if cl.count.Load() != 1 {
		t.Errorf("expected info to be filtered without override, got %d calls", cl.count.Load())
	}
}
