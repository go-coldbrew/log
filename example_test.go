package log_test

import (
	"context"

	"github.com/go-coldbrew/log"
)

func ExampleInfo() {
	ctx := context.Background()
	log.Info(ctx, "service started")
}

func ExampleAddToContext() {
	ctx := context.Background()

	// Add per-request fields to context — these appear in all log lines
	ctx = log.AddToContext(ctx, "request_id", "abc-123")
	ctx = log.AddToContext(ctx, "user_id", "user-42")

	// Subsequent log calls include the context fields
	log.Info(ctx, "processing request")
}

func ExampleError() {
	ctx := context.Background()
	log.Error(ctx, "database connection failed", "retrying in 5s")
}
