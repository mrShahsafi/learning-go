# Lesson 04 — pgx Transaction Patterns

## The Django Equivalent

Django makes transactions feel effortless:

```python
# Django: decorator wraps the whole view in BEGIN/COMMIT/ROLLBACK
from django.db import transaction

@transaction.atomic
def transfer_funds(request):
    src = Account.objects.select_for_update().get(id=request.data["from"])
    dst = Account.objects.select_for_update().get(id=request.data["to"])
    src.balance -= request.data["amount"]
    dst.balance += request.data["amount"]
    src.save()
    dst.save()
    return JsonResponse({"status": "ok"})
```

`@transaction.atomic` handles `BEGIN`, `COMMIT`, and `ROLLBACK` for you.
Any exception → automatic rollback. You never write SQL transaction keywords.

---

## The Go Reality

With `pgx` there is **no decorator**. You call `pool.Begin()`, pass the `tx`
down the call chain, and `defer tx.Rollback()` — that's the pattern.

Done naively it turns into boilerplate in every handler:

```go
// naive — copy-pasted boilerplate in every handler
tx, err := pool.Begin(ctx)
if err != nil { ... }
defer tx.Rollback(ctx)

// ... do work ...

if err := tx.Commit(ctx); err != nil { ... }
```

The senior move is a **`withTx` helper** that accepts a closure. One function,
no boilerplate scattered across handlers.

---

## Core Pattern: `withTx` Closure

```go
// db/tx.go
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
    tx, err := pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx) // no-op if Commit was called — safe to always defer

    if err := fn(tx); err != nil {
        return err // Rollback fires via defer
    }
    return tx.Commit(ctx)
}
```

**Why `defer tx.Rollback` is always safe:**
After a successful `Commit`, Rollback returns `ErrTxClosed` — which we ignore.
If `fn` panics, the defer fires and cleans up the connection. Zero risk.

---

## The Three Traps for Django Developers

### Trap 1 — You must pass `tx` down the call chain
Django's ORM uses a thread-local connection; your repo just calls the ORM and
the active transaction is implicit. In Go there is no thread-local. You must
pass `tx` (as a `pgx.Tx`) to every function that needs to run inside the same
transaction.

```go
// Wrong: repo uses pool — separate connection, outside the transaction
func (r *UserRepo) CreateUser(ctx context.Context, u User) error {
    _, err := r.pool.Exec(ctx, "INSERT INTO users ...", ...)
    return err
}

// Right: repo accepts a pgx.Tx (or a pgx.Querier interface)
func (r *UserRepo) CreateUser(ctx context.Context, q pgx.Tx, u User) error {
    _, err := q.Exec(ctx, "INSERT INTO users ...", ...)
    return err
}
```

### Trap 2 — Forgetting `defer tx.Rollback` causes connection leaks
If you return early without Rollback, the transaction stays open and the
connection is held. Under load this exhausts `pgxpool` and every request hangs.

### Trap 3 — Checking the `Rollback` error
`tx.Rollback` after a successful `Commit` always returns an error
(`ErrTxClosed`). If you log all rollback errors you'll spam your logs.
Either ignore it or check `!errors.Is(err, pgx.ErrTxClosed)`.

---

## Interface Trick: `pgx.Tx` and `*pgxpool.Pool` share a common interface

Both implement `pgx.Querier`-compatible methods (`Exec`, `Query`, `QueryRow`).
Define a repo that accepts an interface so it works both in and out of a transaction:

```go
// Querier is satisfied by *pgxpool.Pool, pgx.Tx, and *pgxpool.Conn
type Querier interface {
    Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
```

Repo uses `Querier`; in normal handlers pass `pool`, in transactional handlers
pass `tx`. Same repo, zero duplication.

---

## Full Runnable Example

See the files in this directory:

| File | What it shows |
|------|---------------|
| `db.go` | `WithTx` helper |
| `repo.go` | Repo that accepts a `Querier` interface |
| `handler.go` | Gin handler using `WithTx` + the repo |
| `main.go` | Wiring everything together |
| `problem.go` | The naive boilerplate version (before refactor) |

Run with:
```bash
# Requires a local Postgres. See main.go for DSN env var.
go run .
```

---

## Comparison Table

| | Django | Go + pgx |
|---|---|---|
| Start transaction | `@transaction.atomic` decorator | `pool.Begin(ctx)` |
| Auto-rollback on error | Yes (decorator catches exceptions) | `defer tx.Rollback(ctx)` |
| Pass TX to repo | Implicit (thread-local) | Explicit: pass `pgx.Tx` or `Querier` |
| Savepoints | `transaction.savepoint()` | `tx.SavePoint(ctx, name)` |
| Read-only TX | Not built-in | `pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})` |
| Connection leak risk | None (framework manages) | High if you skip `defer Rollback` |

---

## The Single Trap That Bites Django Developers Hardest

> **You MUST pass `tx` explicitly.** There is no implicit transaction context.
> If your repo uses `pool.Exec(...)` instead of `tx.Exec(...)`, it runs on a
> separate connection outside your transaction — the ROLLBACK does nothing to it.
> This is the silent bug: tests pass, errors are logged, but half the writes
> survive when they should have been rolled back.
