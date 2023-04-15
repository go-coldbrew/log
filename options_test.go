package log

import (
	"context"
	"testing"

	"github.com/go-coldbrew/log/loggers"
)

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
