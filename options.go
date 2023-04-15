package log

import (
	"context"

	"github.com/go-coldbrew/log/loggers"
	"github.com/go-coldbrew/options"
)

const logLevelKey = "OverrideLogLevel"

// OverrideLogLevel allows the default log level to be overridden from request context
// This is useful when you want to override the log level for a specific request
// For example, you can set the log level to debug for a specific request
// while the default log level is set to info
func OverrideLogLevel(ctx context.Context, level loggers.Level) context.Context {
	return options.AddToOptions(ctx, logLevelKey, level.String())
}

// GetOverridenLogLevel fetches overriden log level from context
// If no log level is overriden, it returns false
// If log level is overriden, it returns the log level and true
func GetOverridenLogLevel(ctx context.Context) (loggers.Level, bool) {
	opt := options.FromContext(ctx)
	if opt == nil {
		return 0, false
	}
	if val, found := opt.Get(logLevelKey); found {
		if level, ok := val.(string); ok {
			l, err := loggers.ParseLevel(level)
			if err != nil {
				return 0, false
			}
			return l, true
		}
	}
	return 0, false
}
