package main

import "context"

type ctxKey int

const requestIDKey ctxKey = iota

// WithRequestID stores a request ID on the context. Unexported key type
// prevents collisions with keys set by other packages.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func RequestIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey).(string)
	return id, ok
}
