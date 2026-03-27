package stdlog_test

import (
	"context"

	"github.com/go-coldbrew/log"
	"github.com/go-coldbrew/log/loggers/stdlog"
)

// This example shows how to use the stdlog backend.
// It uses Go's standard "log" package for output — simple but no structured formatting.
func Example() {
	logger := stdlog.NewLogger()
	log.SetLogger(log.NewLogger(logger))

	ctx := context.Background()
	log.Info(ctx, "msg", "server started", "port", 8080)
}
