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
| 01 | Django → Go/Gin: What Actually Bites You | [`lessons/01-django-to-go-gaps/`](lessons/01-django-to-go-gaps/README.md) | Errors are values · no ORM sync · real goroutines · no settings.py |
| 02 | Dependency Injection Without a Framework | [`lessons/02-dependency-injection/`](lessons/02-dependency-injection/README.md) | Config→Repo→Service→Handler chain · interface for testability · never use package globals |
| 03 | ozzo-validation: Custom Rules, Nested Structs, RFC 9457 | [`lessons/03-ozzo-validation/`](lessons/03-ozzo-validation/README.md) | Bind then validate (two steps) · custom Rule interface · pointer = optional nested · flattenErrors for RFC 9457 |

---

## Next Topics Queue

Suggested deep-dives based on your stack and where Go seniors invest their attention:

1. **pgx transaction patterns** — `BEGIN` / `ROLLBACK` in Gin handlers without boilerplate; the `func(tx pgx.Tx) error` closure pattern.
4. **Structured logging with slog** — replacing fmt.Println with Go 1.21 `log/slog`, attaching `request_id` to every log line via context.
5. **Interface-based testing** — defining repo interfaces so handlers are testable without a real DB; the `fakeRepo` pattern seniors use instead of mocks.
6. **Context propagation** — what `c.Request.Context()` is, why you pass it to every DB call, and how it enables graceful shutdown and timeout cancellation.
7. **Graceful shutdown** — `signal.NotifyContext` + `srv.Shutdown()` so in-flight requests finish before the process exits.
8. **Go module layout for a production API** — the `internal/` convention, why `cmd/api/main.go` is tiny, and how to split handlers / services / repos without circular imports.

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
