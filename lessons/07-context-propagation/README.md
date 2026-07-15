# Lesson 07 — Context Propagation

## The Django Equivalent

Django (WSGI, and mostly ASGI too) gives you no first-class way to know the
client hung up, and no built-in "cancel this DB query, the caller left."
`request` is just a data bag — there's no `request.cancelled()` you check
mid-view, and Postgres queries run via `psycopg2`/Django's ORM run to
completion regardless of whether anyone is still listening.

The closest Django gets is Celery for the "don't block the response on slow
work" half of this lesson:

```python
# views.py
def create_order(request):
    order = Order.objects.create(product_id=request.data["product_id"])
    write_audit_log.delay(f"order created: {order.product_id}")  # fire-and-forget
    return JsonResponse({"status": "accepted"}, status=201)
```

`delay()` hands the job to a broker (Redis/RabbitMQ) — a separate process
runs it. That's a bigger hammer than what Go needs for the same problem.

---

## The Go Reality

Every `*gin.Context` carries a real `context.Context` at `c.Request.Context()`.
It is **not** a data bag — it's a live cancellation signal wired to the
underlying TCP connection. The moment the client disconnects (closes the
tab, hits Ctrl+C, the load balancer times out), `net/http` cancels that
context automatically. No middleware, no polling, no Celery broker required.

The catch: that signal only reaches your database query, your outbound HTTP
call, or your goroutine if **you physically pass `ctx` as the first
parameter through every layer** — handler → service → repo — exactly like
`[[lesson 04's pgx.Tx|../04-pgx-transactions/README.md]]` had to be threaded
explicitly, and exactly like `[[lesson 05's request_id|../05-structured-logging/README.md]]`
had to be threaded explicitly. `ctx` is the third thing on this list that
Go refuses to make ambient. If any function in the chain swaps in
`context.Background()`, the whole chain above it stops mattering.

Real `pgx` (used in `[[lesson 04|../04-pgx-transactions/README.md]]`) behaves
exactly like `ReportRepo.BuildRevenueReport` in this lesson: every
`Query`/`Exec`/`QueryRow` call selects on `ctx.Done()` internally, and
cancellation sends a real Postgres `CancelRequest` to abort the query
server-side — not just client-side. A canceled request that reaches the DB
layer stops wasting a connection and CPU on a query nobody will read.

---

## The Three Traps for Django Developers

### Trap 1 — Forgetting to pass `ctx` silently disables all of this
```go
// WRONG — query runs to completion even if the client left 10s ago
func (r *ReportRepo) BuildRevenueReport(_ context.Context, accountID int64) (string, error) {
    return r.pool.QueryRow(context.Background(), sql, accountID) // ...
}
```
There's no compiler error, no panic — just a query that keeps running,
holding a connection, doing work for a client that's gone. This is the same
failure shape as forgetting to pass `ctx` to a logger call in lesson 05:
silent, structural, and invisible until someone asks "why is our connection
pool exhausted under load."

### Trap 2 — Reusing the request's `ctx` for background work kills it early
See `OrderService.CreateOrder` in `service.go`. Gin cancels
`c.Request.Context()` the instant the handler function returns — that's
*before* a fire-and-forget goroutine spawned inside it has any chance to
finish. Passing that same `ctx` into `AuditLogRepo.Write` would abort the
audit write mid-flight almost every time.

```go
// WRONG — this goroutine's ctx dies the moment CreateOrder returns
go s.auditRepo.Write(ctx, event)

// RIGHT — same request_id/trace values, but cancellation stripped
detached := context.WithoutCancel(ctx)
go s.auditRepo.Write(detached, event)
```

The instinct coming from Django is to reach for `context.Background()`
instead — but that also throws away every value already attached to `ctx`
(request ID, trace ID, anything a `[[lesson 05|../05-structured-logging/README.md]]`-style
logger reads back out). `context.WithoutCancel` (stdlib since Go 1.21) is
the correct tool: values survive, cancellation doesn't.

### Trap 3 — `context.WithTimeout` only ever tightens, never loosens
```go
ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
defer cancel()
```
If the incoming `ctx` already has a deadline 200ms from now, asking for a
1-second timeout does **not** grant you a full second — the earlier deadline
still wins. This trips people who expect `WithTimeout` to "set" a deadline
the way you'd set a Python variable; it actually composes with whatever
deadline is already in force. `GetReportFast` in this lesson enforces its own
1s SLA on top of the request's ctx — try it against a request whose own
deadline is already shorter and the shorter one wins.

Also note: **always call the returned `cancel()`**, even on the success path
(`defer cancel()` right after `WithTimeout`/`WithCancel`). Skipping it leaks
the timer until the parent context itself is done — a `go vet`-flagged
mistake that's easy to make once and repeat everywhere.

---

## Full Runnable Example

| File | What it shows |
|------|---------------|
| `repo.go` | `ReportRepo` — a query that selects on `ctx.Done()`, mirroring real pgx. `AuditLogRepo` — a write slow enough to outlive a request |
| `service.go` | `GetRevenueReport` (plain propagation), `GetRevenueReportWithSLA` (`context.WithTimeout` layered on top), `CreateOrder` (`context.WithoutCancel` for fire-and-forget) |
| `handler.go` | Gin handlers wiring `c.Request.Context()` into each service call |
| `main.go` | Route wiring |

Run it:
```bash
go run .

# 1. Plain propagation — Ctrl+C within 3s aborts the query server-side
curl localhost:8080/reports/1
# (hit Ctrl+C before it finishes — server log shows "context canceled")

# 2. Service-level SLA — always fails at ~1s, the query itself takes 3s
curl -w '\n%{http_code}\n' localhost:8080/reports/1/fast

# 3. Detached background work — response is instant, audit log prints ~500ms later
curl -X POST localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"product_id":"sku-1"}'
```

---

## Comparison Table

| | Django | Go |
|---|---|---|
| "Client disconnected" signal | Not exposed — view runs to completion regardless | `c.Request.Context()` canceled automatically by `net/http` |
| Propagating cancellation to the DB | Not applicable — ORM query always runs to completion | Pass `ctx` through every layer; pgx aborts server-side via `CancelRequest` |
| Per-call deadline shorter than the caller's | Not idiomatic — would need manual `signal.alarm` or threading tricks | `context.WithTimeout(ctx, d)` — tightens, never loosens |
| Fire-and-forget background work | Celery `.delay()` — separate broker + worker process | `go func(){}()` with `context.WithoutCancel(ctx)` — same process, values kept, cancellation dropped |
| Detach without losing request-scoped values | N/A (threadlocal survives independently) | `context.WithoutCancel` — NOT `context.Background()`, which also drops values |

---

## The Single Trap That Bites Django Developers Hardest

> **`ctx` is a live cancellation signal, not a data bag — and Go will never
> propagate it for you.** Django never asks you to think about "is the
> client still there," so there's no instinct to thread a cancellation
> token through handler → service → repo. In Go, every layer that drops
> `ctx` (substitutes `context.Background()`, forgets the parameter, or reuses
> a request's `ctx` for work that must outlive the request) breaks the chain
> silently — no error, just wasted queries or goroutines killed too early.
> The fix is always the same: pass `ctx` explicitly, everywhere, and reach
> for `context.WithoutCancel` — never `context.Background()` — the moment
> work needs to survive past the request that started it.
