# Lesson 08 — Graceful Shutdown

## The Django Equivalent

You have almost certainly never written shutdown code for Django — because
**gunicorn wrote it for you**. When you `kill -TERM` a gunicorn master (or
Kubernetes rolls a pod), gunicorn stops accepting connections, lets workers
finish their in-flight requests up to `--graceful-timeout` (default 30s),
then SIGKILLs anything still running:

```bash
# gunicorn config you've probably cargo-culted
gunicorn app.wsgi --graceful-timeout 30 --timeout 60
```

Your Django code never sees any of this. There's no `on_shutdown` hook in a
view, no "drain" phase you implement — the process supervisor owns the
lifecycle, and your app is just a function it calls.

---

## The Go Reality

There is no gunicorn. Your binary **is** the master process, the worker,
and the supervisor all at once. If you do nothing, Ctrl+C / SIGTERM kills
the process instantly — mid-request, mid-transaction, mid-write:

- In-flight requests get their TCP connection ripped away (client sees a
  connection reset, not a response).
- A pgx transaction that was about to commit just… doesn't (Postgres rolls
  it back when the connection drops, so at least you're not corrupt — but
  the client who got a 200-in-progress never learns what happened).
- Background goroutines vanish without running their cleanup.

Kubernetes makes this non-optional: every deploy sends SIGTERM to your old
pods. **Without graceful shutdown, every deploy drops live traffic.**

The stdlib gives you the two halves and you wire them together in `main`:

1. **`signal.NotifyContext`** — turns SIGINT/SIGTERM into a canceled
   `context.Context`. This is the same `ctx` machinery from
   [lesson 07](../07-context-propagation/README.md), now applied to the
   *process* lifetime instead of a *request* lifetime.
2. **`srv.Shutdown(ctx)`** — stops accepting new connections, waits for
   in-flight requests to finish, and returns when they're done or when the
   ctx you gave it expires — whichever comes first.

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

go srv.ListenAndServe()   // blocks forever → its own goroutine

<-ctx.Done()              // park here until SIGTERM/SIGINT

shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
srv.Shutdown(shutdownCtx) // drain in-flight requests, max 10s
```

---

## The Three Traps for Django Developers

### Trap 1 — Passing the already-canceled signal ctx into `Shutdown`
```go
// WRONG — ctx is the signal ctx, which is ALREADY canceled by the time
// we get here. Shutdown sees a dead ctx and gives up immediately:
// zero drain time, in-flight requests killed. Looks graceful, isn't.
<-ctx.Done()
srv.Shutdown(ctx)

// RIGHT — a FRESH ctx whose only job is "how long am I willing to wait"
shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
srv.Shutdown(shutdownCtx)
```
This is the single most common graceful-shutdown bug in Go codebases, and
it's invisible in testing because idle servers shut down instantly either
way. It only bites under real traffic.

### Trap 2 — Treating `http.ErrServerClosed` as an error
`ListenAndServe` **always** returns a non-nil error. When `Shutdown` is
called, it returns `http.ErrServerClosed` — that's the success case:

```go
if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
    errCh <- err // a REAL error, e.g. port already in use
}
```
Log-and-exit on every non-nil error and every clean shutdown gets reported
as a crash — the Go flavor of Django's "why is there a stack trace on
normal reload."

### Trap 3 — Tearing down in the wrong order
Startup order: pool → workers → server. Teardown must be the **reverse**:

```
1. srv.Shutdown()   — stop new traffic, drain in-flight requests
2. stop workers      — nothing is producing new background work now
3. pool.Close()      — nothing can possibly touch the DB anymore
```

Close the pool first and every still-draining request errors with
"pool is closed" — you did the drain dance and still returned 500s.
(`handler.go`'s `FakePool` demonstrates exactly this failure if you reorder
the teardown in `server.go`.) In Django you never see this because
gunicorn's worker teardown and psycopg's connection cleanup are sequenced
for you.

---

## Full Runnable Example

| File | What it shows |
|------|---------------|
| `main.go` | Tiny: `signal.NotifyContext` → `run(ctx)` — the shape a production `cmd/api/main.go` should have |
| `server.go` | `run()` owning the full lifecycle: ordered startup, `ListenAndServe` in a goroutine with `ErrServerClosed` handling, ordered teardown with a fresh drain deadline, worker stopped via ctx + `WaitGroup` |
| `handler.go` | `/slow` (5s fake query) to demonstrate draining, plus a `FakePool` whose `Close()` makes teardown-order bugs visible |

Run it:
```bash
go run .           # PORT=9090 go run .  if 8080 is taken

# Terminal 2: start a slow request
curl -w '\n%{http_code}\n' localhost:8080/slow

# Terminal 1: while it's running, hit Ctrl+C ONCE.
# Watch the order in the logs:
#   shutdown signal received, draining requests
#   slow request finished          <- in-flight request completed!
#   audit flusher: final flush, exiting
#   db pool closed
#   clean exit
# ...and terminal 2 still gets its 200.
```

Then try the failure mode: comment out the `signal.NotifyContext` wiring
(replace `ctx` with `context.Background()` in `main.go`) and Ctrl+C
mid-request — curl reports `connection reset by peer` instead of a 200.

---

## Comparison Table

| | Django (gunicorn) | Go |
|---|---|---|
| Who handles SIGTERM | gunicorn master process | You, via `signal.NotifyContext` |
| Draining in-flight requests | `--graceful-timeout 30` flag | `srv.Shutdown(ctxWithTimeout)` |
| "Time's up" hard kill | gunicorn SIGKILLs workers | `Shutdown` returns `DeadlineExceeded` → you call `srv.Close()` |
| Clean-shutdown sentinel | n/a (invisible to app code) | `ListenAndServe` returns `http.ErrServerClosed` — not an error |
| Background worker teardown | Celery `warm shutdown` handles it | Your ctx + `sync.WaitGroup`, sequenced by hand |
| Teardown ordering | Supervisor + libraries sequence it | You: drain → stop workers → close pool (reverse of startup) |
| Deploy safety | Free | Every deploy drops traffic until you write this code |

---

## The Single Trap That Bites Django Developers Hardest

> **Nothing shuts your server down gracefully until you write the code —
> and the code has a booby trap: `srv.Shutdown` needs a *fresh* deadline
> ctx, not the signal ctx that woke you up.** Django developers have never
> owned a process lifecycle; gunicorn's `--graceful-timeout` was doing this
> invisibly for years. In Go the whole sequence — catch the signal, drain
> with a deadline, stop workers, close the pool, in that order — is yours.
> Get the ctx wrong (reuse the already-canceled signal ctx) and you ship
> something that *looks* graceful, passes every idle-server test, and still
> drops every in-flight request on deploy day.
