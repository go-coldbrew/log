package loggers_test

import (
	"context"

	"github.com/go-coldbrew/log"
	"github.com/go-coldbrew/log/loggers"
)

// AddToLogContext adds structured fields at the lower level,
// useful when building custom logger integrations or interceptors.
func ExampleAddToLogContext() {
	ctx := context.Background()

	// Add structured fields that propagate through the interceptor chain
	ctx = loggers.AddToLogContext(ctx, "service", "payment-gateway")
	ctx = loggers.AddToLogContext(ctx, "trace_id", "abc-def-123")

	// These fields appear in every log line that uses this context
	log.Info(ctx, "msg", "charge initiated", "amount", 1999, "currency", "USD")
}
