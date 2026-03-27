package zap_test

import (
	"context"

	"github.com/go-coldbrew/log"
	"github.com/go-coldbrew/log/loggers"
	"github.com/go-coldbrew/log/loggers/zap"
)

// This example shows how to use the zap backend with JSON output.
func Example() {
	logger := zap.NewLogger(
		loggers.WithJSONLogs(true),
		loggers.WithCallerInfo(true),
	)
	log.SetLogger(log.NewLogger(logger))

	ctx := context.Background()
	log.Info(ctx, "msg", "request handled", "method", "POST", "latency_ms", 12)
}
