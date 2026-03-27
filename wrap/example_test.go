package wrap_test

import (
	"context"
	"log/slog"

	"github.com/go-coldbrew/log"
	"github.com/go-coldbrew/log/wrap"
)

// This example shows how to route slog calls through ColdBrew's logger.
// Third-party libraries using slog.Info() will flow through your ColdBrew
// logging pipeline (context fields, level overrides, interceptors).
func ExampleToSlogLogger() {
	// Set up slog's default to route through ColdBrew
	sl := wrap.ToSlogLogger(log.GetLogger())
	slog.SetDefault(sl)

	// Now slog calls flow through ColdBrew's pipeline
	ctx := context.Background()
	slog.InfoContext(ctx, "request processed", "method", "GET", "status", 200)
}

// This example shows how to use slog's WithGroup and With methods
// through the ColdBrew bridge.
func ExampleToSlogLogger_withGroup() {
	sl := wrap.ToSlogLogger(log.GetLogger())

	// Groups produce dot-separated keys (e.g., "request.method")
	reqLogger := sl.WithGroup("request").With("trace_id", "abc-123")

	ctx := context.Background()
	reqLogger.InfoContext(ctx, "handled", "method", "POST", "path", "/api/orders")
}
