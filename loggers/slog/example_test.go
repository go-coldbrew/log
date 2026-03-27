package slog_test

import (
	"context"

	"github.com/go-coldbrew/log"
	"github.com/go-coldbrew/log/loggers"
	cbslog "github.com/go-coldbrew/log/loggers/slog"
)

// This example shows how to use the slog backend with JSON output.
// The slog backend is the default and recommended backend.
func Example() {
	// Create a slog-backed logger with JSON output
	logger := cbslog.NewLogger(
		loggers.WithJSONLogs(true),
		loggers.WithCallerInfo(true),
	)

	// Set as the global logger
	log.SetLogger(log.NewLogger(logger))

	// Log normally — output goes through slog's JSONHandler
	ctx := context.Background()
	log.Info(ctx, "msg", "service started", "port", 8080)
}

// This example shows how to configure the slog backend with text output
// and a custom log level.
func Example_textOutput() {
	logger := cbslog.NewLogger(
		loggers.WithJSONLogs(false),
		loggers.WithLevel(loggers.DebugLevel),
	)
	log.SetLogger(log.NewLogger(logger))

	ctx := context.Background()
	log.Debug(ctx, "msg", "debug message", "detail", "verbose")
}
