package loggers

import (
	"context"

	"github.com/go-coldbrew/options"
)

// LogFields contains all fields that have to be added to logs.
// It wraps *options.Options to share the same RequestContext storage,
// eliminating a separate context.WithValue allocation per request.
// LogFields should be obtained via FromContext or AddToLogContext;
// the zero value is safe but acts as a no-op.
type LogFields struct {
	inner *options.Options
}

// Add adds or modifies a log field.
func (o *LogFields) Add(key string, value any) {
	if o.inner != nil && len(key) > 0 {
		o.inner.Add(key, value)
	}
}

// Del deletes a log field entry.
func (o *LogFields) Del(key string) {
	if o.inner != nil {
		o.inner.Del(key)
	}
}

// Store is a sync.Map-compatible alias for Add.
func (o *LogFields) Store(key, value any) {
	if k, ok := key.(string); ok {
		o.Add(k, value)
	}
}

// Load retrieves a value by key.
func (o *LogFields) Load(key any) (any, bool) {
	if o.inner == nil {
		return nil, false
	}
	return o.inner.Load(key)
}

// Delete is a sync.Map-compatible alias for Del.
func (o *LogFields) Delete(key any) {
	if k, ok := key.(string); ok {
		o.Del(k)
	}
}

// Range calls f sequentially for each key and value in the map.
// If f returns false, Range stops the iteration.
// The callback may safely call Add/Del on the same LogFields instance.
// Uses a slice snapshot for efficient iteration over small field counts.
func (o *LogFields) Range(f func(key, value any) bool) {
	if o.inner != nil {
		o.inner.RangeSlice(f)
	}
}

// wrapAsLogFields wraps an *options.Options as a *LogFields.
func wrapAsLogFields(opts *options.Options) *LogFields {
	return &LogFields{inner: opts}
}

// AddToLogContext adds log fields to context.
// Any info added here will be added to all logs using this context.
// If ctx is nil, context.Background() is used.
func AddToLogContext(ctx context.Context, key string, value any) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return options.AddToLogFields(ctx, key, value)
}

// FromContext fetches log fields from provided context.
func FromContext(ctx context.Context) *LogFields {
	if ctx == nil {
		return nil
	}
	if opts := options.LogFieldsFromContext(ctx); opts != nil {
		return wrapAsLogFields(opts)
	}
	return nil
}
