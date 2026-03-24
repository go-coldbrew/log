package log_test

import (
	"context"

	"github.com/go-coldbrew/log"
	"github.com/go-coldbrew/log/loggers"
)

func ExampleInfo() {
	ctx := context.Background()
	log.Info(ctx, "msg", "order processed", "order_id", "ORD-123", "items", 3)
}

func ExampleAddToContext() {
	ctx := context.Background()

	// Add per-request fields to context — these appear in all subsequent log lines
	ctx = log.AddToContext(ctx, "request_id", "abc-123")
	ctx = log.AddToContext(ctx, "user_id", "user-42")

	// All logs using this context now include request_id and user_id
	log.Info(ctx, "msg", "processing request", "step", "validation")
	log.Info(ctx, "msg", "request complete", "status", "ok", "duration_ms", 42)
}

func ExampleError() {
	ctx := context.Background()
	log.Error(ctx, "msg", "database connection failed", "host", "db.internal", "port", 5432, "retry_in", "5s")
}

// AddToLogContext from the loggers package adds fields at the lower level,
// useful when building custom logger integrations or interceptors.
func ExampleAddToLogContext() {
	ctx := context.Background()

	// Add structured fields that propagate through the interceptor chain
	ctx = loggers.AddToLogContext(ctx, "service", "payment-gateway")
	ctx = loggers.AddToLogContext(ctx, "trace_id", "abc-def-123")

	// These fields appear in every log line that uses this context
	log.Info(ctx, "msg", "charge initiated", "amount", 1999, "currency", "USD")
}
