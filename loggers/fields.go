package loggers

import (
	"context"
	"sync"
)

type logsContext string

var (
	contextKey logsContext = "LogsContextKey"
)

// LogFields contains all fields that have to be added to logs.
// Uses RWMutex + map instead of sync.Map since LogFields is per-request
// and typically not shared across goroutines. A plain map with explicit
// locking is cheaper than sync.Map for this write-few-read-once pattern.
type LogFields struct {
	mu sync.RWMutex
	m  map[string]any
}

// Add or modify log fields
func (o *LogFields) Add(key string, value any) {
	if len(key) > 0 {
		o.mu.Lock()
		if o.m == nil {
			o.m = make(map[string]any, 2)
		}
		o.m[key] = value
		o.mu.Unlock()
	}
}

// Del deletes a log field entry
func (o *LogFields) Del(key string) {
	o.mu.Lock()
	if o.m != nil {
		delete(o.m, key)
	}
	o.mu.Unlock()
}

// Store is a sync.Map-compatible alias for Add.
func (o *LogFields) Store(key, value any) {
	if k, ok := key.(string); ok {
		o.Add(k, value)
	}
}

// Load retrieves a value by key.
func (o *LogFields) Load(key any) (any, bool) {
	k, ok := key.(string)
	if !ok {
		return nil, false
	}
	o.mu.RLock()
	if o.m == nil {
		o.mu.RUnlock()
		return nil, false
	}
	v, found := o.m[k]
	o.mu.RUnlock()
	return v, found
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
func (o *LogFields) Range(f func(key, value any) bool) {
	o.mu.RLock()
	if len(o.m) == 0 {
		o.mu.RUnlock()
		return
	}
	// Snapshot into a slice (cheaper than map copy for small field counts).
	type kv struct {
		k string
		v any
	}
	entries := make([]kv, 0, len(o.m))
	for k, v := range o.m {
		entries = append(entries, kv{k, v})
	}
	o.mu.RUnlock()
	for _, e := range entries {
		if !f(e.k, e.v) {
			break
		}
	}
}

// newLogFields creates a LogFields with an initialized map.
func newLogFields() *LogFields {
	return &LogFields{m: make(map[string]any, 2)}
}

// AddToLogContext adds log fields to context.
// Any info added here will be added to all logs using this context.
// If ctx is nil, context.Background() is used.
func AddToLogContext(ctx context.Context, key string, value any) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	data := FromContext(ctx)
	if data == nil {
		data = newLogFields()
		ctx = context.WithValue(ctx, contextKey, data)
	}
	data.Add(key, value)
	return ctx
}

// FromContext fetchs log fields from provided context
func FromContext(ctx context.Context) *LogFields {
	if ctx == nil {
		return nil
	}
	if h := ctx.Value(contextKey); h != nil {
		if logData, ok := h.(*LogFields); ok {
			return logData
		}
	}
	return nil
}
