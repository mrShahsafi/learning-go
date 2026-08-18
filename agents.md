# Go Learning Journal — Senior Patterns & Tricky Notes

This file is the living index of everything we practice together.
Each lesson has its own directory under `lessons/`. New topics are added here as we cover them.

---

## About You (Preferences)

- Coming from **Python / Django / DRF** background.
- Building a real product: **Coordeck** (Go/Gin · ozzo-validation · RFC 9457 error envelopes · PostgreSQL 15).
- You think contract-first and already lean toward explicit over magical — that's the Go mindset.
- You prefer lessons that are **practical and grounded in your actual stack**, not toy examples.
- You like **visual / tabular comparisons** (Django vs Go side-by-side).
- You want each lesson as a **separate directory with code files**, not just prose.

---

## Lessons Covered

| # | Topic | Directory | Key Takeaway |
|---|-------|-----------|--------------|
| 00 | Go Syntax Cheat Sheet | [`lessons/00-go-syntax-cheatsheet/`](lessons/00-go-syntax-cheatsheet/README.md) | Reference doc, not a lesson · come back here whenever you forget syntax |
| 01 | Django → Go/Gin: What Actually Bites You | [`lessons/01-django-to-go-gaps/`](lessons/01-django-to-go-gaps/README.md) | Errors are values · no ORM sync · real goroutines · no settings.py |
| 02 | Dependency Injection Without a Framework | [`lessons/02-dependency-injection/`](lessons/02-dependency-injection/README.md) | Config→Repo→Service→Handler chain · interface for testability · never use package globals |
| 03 | ozzo-validation: Custom Rules, Nested Structs, RFC 9457 | [`lessons/03-ozzo-validation/`](lessons/03-ozzo-validation/README.md) | Bind then validate (two steps) · custom Rule interface · pointer = optional nested · flattenErrors for RFC 9457 |
| 04 | pgx Transaction Patterns | [`lessons/04-pgx-transactions/`](lessons/04-pgx-transactions/README.md) | `WithTx` closure · `defer Rollback` is always safe · `Querier` interface for pool/tx polymorphism · passing tx explicitly (no thread-local) |
| 05 | Structured Logging with slog | [`lessons/05-structured-logging/`](lessons/05-structured-logging/README.md) | `log/slog` ignores ctx unless you wrap the `Handler` · custom `contextHandler` injects `request_id` · package-level `slog.Info` bypasses ctx entirely — always use `*Context` variants |
| 06 | Interface-Based Testing | [`lessons/06-interface-testing/`](lessons/06-interface-testing/README.md) | `fakeRepo` (real map-backed implementation) replaces `unittest.mock.patch` · table-driven tests replace `pytest.mark.parametrize` · fresh fake per subtest — no automatic transaction rollback like Django's `TestCase` · `httptest` + real `gin.Engine` replaces `APITestCase` |
| 07 | Context Propagation | [`lessons/07-context-propagation/`](lessons/07-context-propagation/README.md) | `ctx` is a live cancellation signal, not a data bag · must be passed explicitly through every layer or the chain silently breaks · `context.WithTimeout` only tightens, never loosens · `context.WithoutCancel` (not `Background()`) for fire-and-forget work that must outlive the request |
| 08 | Graceful Shutdown | [`lessons/08-graceful-shutdown/`](lessons/08-graceful-shutdown/README.md) | You are gunicorn now · `signal.NotifyContext` catches SIGTERM · `srv.Shutdown` needs a FRESH deadline ctx, never the already-canceled signal ctx · `http.ErrServerClosed` is success, not an error · teardown in reverse startup order: drain → workers → pool |
| 09 | Module Layout for a Production API | [`lessons/09-module-layout/`](lessons/09-module-layout/README.md) | `internal/` is compiler-enforced, not just convention · dependency arrows point one way: domain → repository → service → handler → main · `go build` refuses import cycles outright — no Python-style lazy-import escape hatch · `cmd/api/main.go` is tiny because it's the only package allowed to see every layer |
| 10 | Middleware Patterns in Gin | [`lessons/10-gin-middleware/`](lessons/10-gin-middleware/README.md) | Middleware is an ordered `HandlerFunc` chain · `c.Next()` continues · `c.Abort()` still needs `return` · timeouts work through cooperative context cancellation · reuse Gin recovery |

---

## Next Topics Queue

Suggested deep-dives based on your stack and where Go seniors invest their attention:

11. **Concurrency Patterns for API Services** — bounded goroutines, `errgroup`, worker pools, and avoiding goroutine leaks when requests are canceled.

---

## Teaching Style Notes

- Start with the **Django equivalent** when introducing a concept.
- Always use **your actual stack** in code examples (Gin, pgx, ozzo-validation).
- Include a **comparison table** at the end of each lesson.
- Create real `.go` files in the lesson directory — not just prose — so you can run them.
- Point out **the single trap** that bites Django developers hardest in each topic.

---

## How to Use This File

When starting a new session, say:
> "Open agents.md and teach me the next topic in the queue."

I'll pick the top item from the Next Topics Queue, create a new `lessons/NN-topic/` directory with runnable Go files, and update this index.
