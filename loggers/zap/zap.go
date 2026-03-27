// Package zap provides a BaseLogger implementation for uber/zap
package zap

import (
	"context"
	"fmt"

	"github.com/go-coldbrew/log/loggers"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type logger struct {
	logger *zap.SugaredLogger
	opt    loggers.Options
	cfg    zap.Config
}

// COLBREW_CALL_STACK_SIZE number stack frame involved between the logger call from application to zap call.
const COLBREW_CALL_STACK_SIZE = 3

func (l *logger) Log(ctx context.Context, level loggers.Level, skip int, args ...any) {

	var msg string
	// If there are odd number of elements in args, first will be treated as a message and rest will
	// be key value pair to log in json format.
	if len(args)%2 != 0 {
		msg = fmt.Sprint(args[0])
		args = args[1:]
	}
	ctxFields := loggers.FromContext(ctx)
	if ctxFields != nil {
		ctxFields.Range(func(k, v any) bool { args = append(args, k, v); return true })
	}

	// Use structured-logging variants (Infow, Debugw, etc.) to pass fields inline
	// instead of logger.With() which creates a new SugaredLogger wrapper per call.
	switch level {
	case loggers.DebugLevel:
		l.logger.Debugw(msg, args...)
	case loggers.InfoLevel:
		l.logger.Infow(msg, args...)
	case loggers.WarnLevel:
		l.logger.Warnw(msg, args...)
	default:
		l.logger.Errorw(msg, args...)
	}
}

func (l *logger) GetLevel() loggers.Level {
	return l.opt.Level
}

func (l *logger) SetLevel(level loggers.Level) {
	l.opt.Level = level
	l.cfg.Level.SetLevel(toZapLevel(level))
}

func toZapLevel(level loggers.Level) zapcore.Level {

	switch level {
	case loggers.DebugLevel:
		return zapcore.DebugLevel
	case loggers.InfoLevel:
		return zap.InfoLevel
	case loggers.WarnLevel:
		return zap.WarnLevel
	case loggers.ErrorLevel:
		return zap.ErrorLevel
	default:
		return zapcore.ErrorLevel
	}
}

func NewLogger(options ...loggers.Option) loggers.BaseLogger {

	opt := loggers.GetDefaultOptions()
	for _, f := range options {
		f(&opt)
	}

	zapCfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(toZapLevel(opt.Level)),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},

		EncoderConfig: zapcore.EncoderConfig{
			MessageKey: "message",

			LevelKey:    opt.LevelFieldName,
			EncodeLevel: zapcore.CapitalLevelEncoder,

			TimeKey:    opt.TimestampFieldName,
			EncodeTime: zapcore.ISO8601TimeEncoder,

			CallerKey:    opt.CallerFieldName,
			EncodeCaller: zapcore.FullCallerEncoder,
		},
	}

	if opt.JSONLogs {
		zapCfg.Encoding = "json"
	} else {
		zapCfg.Encoding = "console"
	}
	l, err := zapCfg.Build()
	if err != nil {
		l, _ = zap.NewProduction()
	}
	l = l.WithOptions(zap.AddCallerSkip(COLBREW_CALL_STACK_SIZE))

	return &logger{
		logger: l.Sugar(),
		opt:    opt,
		cfg:    zapCfg,
	}

}
