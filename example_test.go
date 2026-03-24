package log_test

import (
	"context"

	"github.com/go-coldbrew/log"
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
