package log

import (
	"context"

	"github.com/go-coldbrew/log/loggers"
)

// Logger interface is implemnted by the log implementation to provide the log methods to the application code.
type Logger interface {
	loggers.BaseLogger
	// Debug logs a message at level Debug.
	// ctx is used to extract the request id and other context information.
	Debug(ctx context.Context, args ...interface{})
	// Info logs a message at level Info.
	// ctx is used to extract the request id and other context information.
	Info(ctx context.Context, args ...interface{})
	// Warn logs a message at level Warn.
	// ctx is used to extract the request id and other context information.
	Warn(ctx context.Context, args ...interface{})
	// Error logs a message at level Error.
	// ctx is used to extract the request id and other context information.
	Error(ctx context.Context, args ...interface{})
}
