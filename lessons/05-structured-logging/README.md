# Lesson 05 — Structured Logging with `log/slog`

## The Django Equivalent

Django logging is configuration, not code. You wire it once in `settings.py`
and call `logging.getLogger(__name__)` everywhere:

```python
# settings.py
LOGGING = {
    "version": 1,
    "handlers": {"console": {"class": "logging.StreamHandler", "formatter": "json"}},
    "formatters": {"json": {"()": "pythonjsonlogger.jsonlogger.JsonFormatter"}},
    "root": {"handlers": ["console"], "level": "INFO"},
}

# middleware.py — request ID via a threadlocal-backed contextvar
import uuid, logging
from asgiref.local import Local

_local = Local()
logger = logging.getLogger(__name__)

class RequestIDMiddleware:
    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        _local.request_id = str(uuid.uuid4())
        response = self.get_response(request)
        logger.info("request handled", extra={"request_id": _local.request_id})
        return response
```

Because Python's `contextvars`/threadlocal storage is implicit, every
`logger.info(...)` call downstream can be made to pick up `request_id`
automatically (via a custom `Filter`), without passing anything explicitly.

---

## The Go Reality

Go's `log/slog` (stdlib since 1.21) gives you structured, leveled, JSON-capable
logging — but it has **no implicit context propagation**. `slog.Handler.Handle`
receives a `context.Context` as its first argument, and the built-in handlers
(`slog.NewJSONHandler`, `slog.NewTextHandler`) **completely ignore it**. That
`ctx` parameter exists purely so *you* can build a handler that reads from it.

This is the senior-level trick: wrap a handler.

```go
// logger.go
type contextHandler struct {
    slog.Handler
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
    if reqID, ok := RequestIDFromContext(ctx); ok {
        r.AddAttrs(slog.String("request_id", reqID))
    }
    return h.Handler.Handle(ctx, r)
}
```

Now every call to `logger.InfoContext(ctx, ...)` / `ErrorContext` / `WarnContext`
automatically gets `request_id` attached — as long as `ctx` actually carries one.

---

## The Three Traps for Django Developers

### Trap 1 — Plain `slog.Info(...)` never sees your context
`slog.Info`, `slog.Error`, etc. are package-level convenience functions that
call the **default logger** with `context.Background()`. They cannot pick up
request-scoped data no matter how your handler is built.

```go
// WRONG — silently missing request_id, even with contextHandler installed
slog.Info("order created", "product_id", id)

// RIGHT — pass the request's ctx explicitly
h.logger.InfoContext(ctx, "order created", slog.String("product_id", id))
```
See the commented-out line in `handler.go` — it's left there deliberately to
show what the trap looks like in a diff.

### Trap 2 — `ctx` must physically flow through every layer
Django's `contextvars` are ambient — a `Filter` reaches into thread-local
state without any function signature changes. Go has no ambient state. `ctx`
must be threaded as an explicit first parameter through handler → service →
repo, same as `[[lesson 04's pgx.Tx|../04-pgx-transactions/README.md]]` had to
be threaded explicitly. If a function drops `ctx` (or substitutes
`context.Background()`), every log call downstream silently loses `request_id`.

### Trap 3 — `logger.With(...)` returns a *new* logger, it doesn't mutate
`logger.With(slog.String("x", "y"))` allocates and returns a new `*slog.Logger`;
the original is untouched. This is idiomatic for scoping attrs to a call chain
(e.g. `repoLogger := h.logger.With("repo", "orders")`), but it means "attach
this once and it applies everywhere" doesn't exist — that's exactly the gap
`contextHandler` fills for request-scoped data like `request_id`.

---

## Full Runnable Example

| File | What it shows |
|------|---------------|
| `context.go` | Typed context key + `WithRequestID` / `RequestIDFromContext` |
| `logger.go` | `contextHandler` — the handler wrapper that injects `request_id` from ctx |
| `middleware.go` | Gin middleware: generates request ID, stores it on `c.Request`'s context, logs on completion |
| `handler.go` | A handler using `InfoContext`/`WarnContext`/`ErrorContext` — plus the commented-out trap |
| `main.go` | Wiring: builds the logger, registers the middleware, starts Gin |

Run it:
```bash
go run .

# in another terminal
curl -X POST localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"product_id":"sku-1","quantity":3}'
```

Every log line for that request — the handler's `"order created"` line and the
middleware's `"request handled"` line — carries the **same** `request_id`,
without either one importing or knowing about the other.

---

## Comparison Table

| | Django | Go + `log/slog` |
|---|---|---|
| Structured output | `python-json-logger` (3rd party) | `slog.NewJSONHandler` (stdlib) |
| Request ID propagation | Implicit via `contextvars`/threadlocal | Explicit: must pass `ctx` to every `*Context` call |
| Attach request-scoped fields globally | `logging.Filter` reads threadlocal | Custom `slog.Handler` reads `ctx.Value` |
| Per-call structured fields | `logger.info(msg, extra={...})` | `logger.InfoContext(ctx, msg, slog.String(...))` |
| Scoped "sub-logger" | `logger.getChild(name)` | `logger.With(...)` — returns a new `*slog.Logger` |
| Log levels | `DEBUG/INFO/WARNING/ERROR/CRITICAL` | `Debug/Info/Warn/Error` (stdlib has no Critical) |

---

## The Single Trap That Bites Django Developers Hardest

> **`ctx` is not ambient — it's a value you pass, and if you drop it, your
> structured fields silently vanish.** There's no traceback, no error, no
> lint warning: the log line just prints without `request_id`, correlating to
> nothing. In Django, threadlocal state means you can forget to pass anything
> and logging still "just works." In Go, the discipline of always calling
> `logger.XContext(ctx, ...)` — never `logger.X(...)` or the package-level
> `slog.X(...)` — inside a request scope *is* the correctness guarantee.
