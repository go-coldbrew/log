package gokit_test

import (
	"context"

	"github.com/go-coldbrew/log"
	"github.com/go-coldbrew/log/loggers"
	"github.com/go-coldbrew/log/loggers/gokit"
)

// This example shows how to use the gokit backend.
//
// Deprecated: Use the slog backend instead (loggers/slog). The go-kit/log
// library is in maintenance mode and no longer actively developed.
func Example() {
	logger := gokit.NewLogger(
		loggers.WithJSONLogs(true),
		loggers.WithCallerInfo(true),
	)
	log.SetLogger(log.NewLogger(logger))

	ctx := context.Background()
	log.Info(ctx, "msg", "request handled", "method", "GET", "path", "/api/v1")
}
