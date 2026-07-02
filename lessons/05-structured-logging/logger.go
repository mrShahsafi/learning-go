package main

import (
	"context"
	"log/slog"
	"os"
)

// contextHandler wraps any slog.Handler and injects request-scoped attributes
// pulled from ctx onto every record. slog does NOT do this for you — Handle
// receives ctx but the built-in handlers ignore it entirely.
type contextHandler struct {
	slog.Handler
}

func newContextHandler(h slog.Handler) *contextHandler {
	return &contextHandler{Handler: h}
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if reqID, ok := RequestIDFromContext(ctx); ok {
		r.AddAttrs(slog.String("request_id", reqID))
	}
	return h.Handler.Handle(ctx, r)
}

// NewLogger builds a JSON slog.Logger whose every *Context call (InfoContext,
// ErrorContext, ...) auto-attaches request_id when one is present on ctx.
func NewLogger() *slog.Logger {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return slog.New(newContextHandler(jsonHandler))
}
