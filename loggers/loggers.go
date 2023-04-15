/*Package loggers provides loggers implementation for log package
 */
package loggers

import (
	"context"
	"fmt"
	"runtime"
	"strings"
)

// Level type
type Level uint32

// Convert the Level to a string. E.g.  ErrorLevel becomes "error".
func (level Level) String() string {
	switch level {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warning"
	case ErrorLevel:
		return "error"
	}

	return "unknown"
}

// ParseLevel takes a string level and returns the log level constant.
func ParseLevel(lvl string) (Level, error) {
	switch strings.ToLower(lvl) {
	case "error":
		return ErrorLevel, nil
	case "warn", "warning":
		return WarnLevel, nil
	case "info":
		return InfoLevel, nil
	case "debug":
		return DebugLevel, nil
	}

	var l Level
	return l, fmt.Errorf("not a valid log Level: %q", lvl)
}

// AllLevels A constant exposing all logging levels
var AllLevels = []Level{
	ErrorLevel,
	WarnLevel,
	InfoLevel,
	DebugLevel,
}

// These are the different logging levels. You can set the logging level to log
// on your instance of logger, obtained with `logs.New()`.
const (
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	// Commonly used for hooks to send errors to an error tracking service.
	ErrorLevel = iota
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel level. General operational entries about what's going on inside the
	// application.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel
)

// BaseLogger is the interface that needs to be implemented by client loggers
type BaseLogger interface {
	// Log logs a message at the given level. The args are key-value pairs.
	// The key must be a string, while the value can be of any type.
	// The message is logged with the given level.
	// Level is the level of the log message.
	// Skip is the number of stack frames to skip before getting the file name and line number.
	// If skip is 0, the file and line of the caller of Log is logged.
	// ctx is the context of the log message. It is used to pass the context fields to the logger
	Log(ctx context.Context, level Level, skip int, args ...interface{})
	// SetLevel sets the level of the logger
	SetLevel(level Level)
	// GetLevel gets the level of the logger
	GetLevel() Level
}

// Options contain all common options for BaseLoggers
type Options struct {
	// ReplaceStdLogger replaces std logger with this logger
	ReplaceStdLogger bool
	// JSONLogs enables/disables json logs
	JSONLogs bool
	// Level is the level of the logger
	Level Level
	// TimestampFieldName is the name of the timestamp field in logs
	TimestampFieldName string
	// LevelFieldName is the name of the level field in logs
	LevelFieldName string
	// CallerInfo enables/disables adding caller info to logs
	CallerInfo bool
	// CallerFileDepth is the depth of file to use in caller info
	CallerFileDepth int
	// CallerFieldName is the name of callerinfo field
	CallerFieldName string
}

// GetDefaultOptions fetches loggers default options
func GetDefaultOptions() Options {
	return DefaultOptions
}

// DefaultOptions stores all default options in loggers package
// These options are used by all loggers
var (
	DefaultOptions = Options{
		ReplaceStdLogger:   false,
		JSONLogs:           true,
		Level:              InfoLevel,
		TimestampFieldName: "@timestamp",
		LevelFieldName:     "level",
		CallerInfo:         true,
		CallerFileDepth:    2,
		CallerFieldName:    "caller",
	}
)

// Option defines an option for BaseLogger
type Option func(*Options)

// WithReplaceStdLogger enables/disables replacing std logger
func WithReplaceStdLogger(replaceStdLogger bool) Option {
	return func(o *Options) {
		o.ReplaceStdLogger = replaceStdLogger
	}
}

// WithJSONLogs enables/disables json logs
func WithJSONLogs(json bool) Option {
	return func(o *Options) {
		o.JSONLogs = json
	}
}

// WithTimestampFieldName sets the name of the time stamp field in logs
func WithTimestampFieldName(name string) Option {
	return func(o *Options) {
		if name != "" {
			o.TimestampFieldName = name
		}
	}
}

// WithLevelFieldName sets the name of the level field in logs
func WithLevelFieldName(name string) Option {
	return func(o *Options) {
		if name != "" {
			o.LevelFieldName = name
		}
	}
}

// WithCallerInfo enables/disables adding caller info to logs
func WithCallerInfo(callerInfo bool) Option {
	return func(o *Options) {
		o.CallerInfo = callerInfo
	}
}

// WithCallerFileDepth sets the depth of file to use in caller info
func WithCallerFileDepth(depth int) Option {
	return func(o *Options) {
		if depth > 0 {
			o.CallerFileDepth = depth
		}
	}
}

// WithCallerFieldName sets the name of callerinfo field
func WithCallerFieldName(name string) Option {
	return func(o *Options) {
		name = strings.TrimSpace(name)
		if name != "" {
			o.CallerFieldName = name
		}
	}
}

// FetchCallerInfo fetches function name, file name and line number from stack
// skip is the number of stack frames to ascend, with 0 identifying the caller of FetchCallerInfo.
// depth is the depth of file to use in caller info
func FetchCallerInfo(skip int, depth int) (function string, file string, line int) {
	if depth <= 0 {
		depth = 2
	}
	if skip < 0 {
		skip = 0
	}
	pc, file, line, _ := runtime.Caller(skip + 1)
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			depth--
			if depth == 0 {
				file = file[i+1:]
				break
			}
		}
	}
	return runtime.FuncForPC(pc).Name(), file, line
}
