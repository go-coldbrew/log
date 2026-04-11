/*
Package log provides structured logging for ColdBrew microservices.

It uses a custom slog.Handler that automatically injects per-request context
fields (added via [AddToContext] or [loggers.AddToLogContext]) into every log record.

# Quick Start

Use the package-level functions for simple logging:

	log.Info(ctx, "msg", "order processed", "order_id", "ORD-123")
	log.Error(ctx, "msg", "connection failed", "host", "db.internal")

# Native slog Support

After calling [SetDefault], native slog calls automatically get ColdBrew
context fields:

	log.SetDefault(log.NewHandler())
	ctx := log.AddToContext(ctx, "trace_id", "abc-123")
	slog.InfoContext(ctx, "request handled", "status", 200) // includes trace_id

# Typed Attrs (Zero-Boxing Path)

Use [AddAttrsToContext] with [slog.LogAttrs] for the highest performance path,
avoiding interface boxing for both context fields and per-call attributes:

	ctx = log.AddAttrsToContext(ctx,
	    slog.String("trace_id", id),
	    slog.Int("user_id", uid),
	)
	slog.LogAttrs(ctx, slog.LevelInfo, "request handled", slog.Int("status", 200))

# Custom Handlers

Use [NewHandlerWithInner] to compose ColdBrew's context injection with any
slog.Handler:

	// Fan-out to multiple destinations
	multi := slogmulti.Fanout(jsonHandler, textHandler)
	h := log.NewHandlerWithInner(multi)
	log.SetDefault(h)

# Contextual Logs

Use [AddToContext] to add per-request fields that appear in all subsequent logs:

	ctx = log.AddToContext(ctx, "request_id", "abc-123")
	ctx = log.AddToContext(ctx, "user_id", "user-42")
	log.Info(ctx, "msg", "processing request") // includes request_id and user_id

ColdBrew interceptors automatically add grpcMethod, trace ID, and HTTP path
to the context, so these fields appear in all service logs.
*/
package log
