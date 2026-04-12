/*
Package log provides structured logging for ColdBrew microservices.

It uses a custom slog.Handler that automatically injects per-request context
fields (added via [AddToContext] or [AddAttrsToContext]) into every log record.

# Quick Start

Use the package-level functions for simple logging:

	log.Info(ctx, "msg", "order processed", "order_id", "ORD-123")
	log.Error(ctx, "msg", "connection failed", "host", "db.internal")

# Native slog Support

After calling [SetDefault], native slog calls automatically get ColdBrew
context fields:

	log.SetDefault(log.NewHandler())
	ctx := context.Background()
	ctx = log.AddToContext(ctx, "trace_id", "abc-123")
	slog.InfoContext(ctx, "request handled", "status", 200) // includes trace_id

# Adding Context Fields

Use [AddAttrsToContext] to add typed slog.Attr fields, or [AddToContext] for
untyped key-value pairs. Both are included in all subsequent logs for that
request:

	ctx := context.Background()
	ctx = log.AddAttrsToContext(ctx,
	    slog.String("trace_id", id),
	    slog.Int("user_id", uid),
	)

[AddAttrsToContext] stores each slog.Attr in the context. At log time, the
Handler recovers the typed Attr and emits it directly. Context storage goes
through an any-typed API internally (one boxing per field per request), but
the Attr's type information is preserved for emission.

# High-Performance Logging

Combine [AddAttrsToContext] with [slog.LogAttrs] for the lowest-overhead path.
Per-call attributes passed to [slog.LogAttrs] avoid interface boxing entirely:

	slog.LogAttrs(ctx, slog.LevelInfo, "request handled",
	    slog.Int("status", 200),
	    slog.Duration("latency", elapsed),
	)

# Custom Handlers

Use [NewHandlerWithInner] to compose ColdBrew's context injection with any
slog.Handler:

	multi := slogmulti.Fanout(jsonHandler, textHandler)
	h := log.NewHandlerWithInner(multi)
	log.SetDefault(h)

ColdBrew interceptors automatically add grpcMethod, trace ID, and HTTP path
to the context, so these fields appear in all service logs.
*/
package log
