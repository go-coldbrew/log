package log

import (
	"context"

	"github.com/go-coldbrew/log/loggers"
)

// Logger is the interface for ColdBrew's structured logging.
// It extends loggers.BaseLogger with convenience methods for each log level.
type Logger interface {
	loggers.BaseLogger //nolint:staticcheck // intentional use for backward compatibility
	// Debug logs a message at level Debug.
	// ctx is used to extract per-request context fields.
	Debug(ctx context.Context, args ...any)
	// Info logs a message at level Info.
	// ctx is used to extract per-request context fields.
	Info(ctx context.Context, args ...any)
	// Warn logs a message at level Warn.
	// ctx is used to extract per-request context fields.
	Warn(ctx context.Context, args ...any)
	// Error logs a message at level Error.
	// ctx is used to extract per-request context fields.
	Error(ctx context.Context, args ...any)
}
