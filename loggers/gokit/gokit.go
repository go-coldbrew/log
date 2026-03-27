// Deprecated: Package gokit provides BaseLogger implementation for go-kit/log.
// The go-kit/log library is in maintenance mode and no longer actively developed.
// Use the slog backend (loggers/slog) instead, which is the default and recommended backend.
package gokit

import (
	"context"
	"fmt"
	stdlog "log"
	"os"

	"github.com/go-coldbrew/log/loggers"
	"github.com/go-kit/log"
)

type logger struct {
	logger log.Logger
	level  loggers.Level
	opt    loggers.Options
}

func (l *logger) Log(ctx context.Context, level loggers.Level, skip int, args ...any) {
	// Batch all extra fields (level, caller, context) into a single slice
	// and call log.With() once instead of N separate wrapper allocations.
	extra := make([]any, 0, 8)
	extra = append(extra, l.opt.LevelFieldName, level.String())

	if l.opt.CallerInfo {
		_, file, line := loggers.FetchCallerInfo(skip+1, l.opt.CallerFileDepth)
		extra = append(extra, l.opt.CallerFieldName, fmt.Sprintf("%s:%d", file, line))
	}

	ctxFields := loggers.FromContext(ctx)
	if ctxFields != nil {
		ctxFields.Range(func(k, v any) bool {
			extra = append(extra, k, v)
			return true
		})
	}

	lgr := log.With(l.logger, extra...)
	if len(args) == 1 {
		_ = lgr.Log("msg", args[0])
	} else {
		_ = lgr.Log(args...)
	}
}

func (l *logger) SetLevel(level loggers.Level) {
	l.level = level
}

func (l *logger) GetLevel() loggers.Level {
	return l.level
}

// NewLogger returns a base logger impl for go-kit log
func NewLogger(options ...loggers.Option) loggers.BaseLogger {
	// default options
	opt := loggers.GetDefaultOptions()

	// read options
	for _, f := range options {
		f(&opt)
	}

	l := logger{}
	writer := log.NewSyncWriter(os.Stdout)

	// check for json or logfmt
	if opt.JSONLogs {
		l.logger = log.NewJSONLogger(writer)
	} else {
		l.logger = log.NewLogfmtLogger(writer)
	}

	l.logger = log.With(l.logger, opt.TimestampFieldName, log.DefaultTimestamp)

	l.level = opt.Level
	l.opt = opt

	if opt.ReplaceStdLogger {
		stdlog.SetFlags(stdlog.LUTC)
		stdlog.SetOutput(log.NewStdlibAdapter(l.logger, log.TimestampKey(opt.TimestampFieldName)))
	}
	return &l
}
