# Lesson 09 — Module Layout for a Production API

## The Django Equivalent

Django gives you an "app" — `myapp/models.py`, `views.py`, `serializers.py`,
`urls.py`, `admin.py` — and a project-level `urls.py` that stitches apps
together. Nothing stops `myapp/views.py` from importing `otherapp/models.py`,
which imports something that imports back into `myapp`. Python resolves the
cycle at import time (or doesn't, and you get the classic
`ImportError: cannot import name X from partially initialized module`), and
the usual fix is folding the import inside the function body so it runs
*after* both modules have finished loading. It's a workaround, not a design —
the dependency graph between your apps is whatever imports happen to exist,
enforced by nothing.

## The Go Reality

Go's build graph is a DAG, and the compiler refuses to build anything else.
There's no "import inside the function to dodge the cycle" escape hatch —
if package A imports B and B imports A, `go build` stops with:

```
package lesson09/module-layout/cmd/api
	imports lesson09/module-layout/internal/handler from main.go
	imports lesson09/module-layout/internal/service from booking_handler.go
	imports lesson09/module-layout/internal/repository from booking_service.go
	imports lesson09/module-layout/internal/service from booking_repo.go: import cycle not allowed
```

That forces a decision Django lets you defer forever: **which direction does
each dependency point?** The convention that answers it:

```
domain  →  (nothing)                  — plain types, zero deps
repository → domain                   — persistence, knows the data shape
service → domain, repository          — business logic, knows storage as an interface
handler → domain, service             — HTTP layer, knows business logic as an interface
main    → everything                  — wiring only, nothing imports main
```

Arrows point one way, inward to outward never happens, and `main` is the only
package allowed to know about the whole graph. This lesson's code is exactly
that shape:

```
cmd/api/main.go            — wiring: config → repo → service → handler → gin
internal/domain/           — Booking struct, no imports from this module
internal/repository/       — BookingRepository interface + in-memory impl
internal/service/          — BookingService: business rules, depends on the repo interface
internal/handler/          — Gin handlers, depend on the service interface
internal/config/           — env var loading
```

### Why `internal/`?

`internal/` isn't just a naming convention — the Go toolchain enforces it.
Any package rooted under a directory named `internal` can only be imported by
code whose import path shares the parent of that `internal` directory. If
Coordeck published this as a library, an outside module could `import
"lesson09/module-layout/cmd/api"` if it really wanted to, but it **cannot**
`import "lesson09/module-layout/internal/repository"` — the compiler blocks
it. Django has no equivalent: any app, in any project that happens to have
your code on its `PYTHONPATH`, can `import myapp.models` whether you meant it
to be public API or not.

### Why is `cmd/api/main.go` so small?

Because `main` is the *only* package that's allowed to see every layer at
once — so it's also the only place where the whole dependency chain gets
built. Everything else is called through an interface it received, not a
concrete type it constructed. Compare:

```go
// main.go — the entire lifecycle of the app, in wiring order
cfg := config.Load()
repo := repository.NewMemoryBookingRepo()
svc := service.NewBookingService(repo)   // svc only knows repo as an interface
h := handler.NewBookingHandler(svc)       // h only knows svc as an interface

r := gin.Default()
handler.RegisterRoutes(r, h)
r.Run(":" + cfg.Port)
```

If `main.go` starts growing business logic, that's the signal it's no longer
just wiring — pull that logic down into `service`.

---

## The Single Trap That Bites Django Developers Hardest

> **Django lets you paper over a circular dependency with a lazy import
> inside a function; Go's compiler has no such escape hatch — it will not
> build a cycle, period.** That means the package layout isn't a
> nice-to-have you can clean up later, it's a decision you're forced to make
> before the code compiles at all. Get the dependency direction backwards
> (say, `repository` importing `service` to call back into business logic)
> and you don't get subtly wrong behavior like in Python — you get a build
> failure naming the exact cycle, every time, for every developer, forever.
> The fix is always the same: introduce an interface at the lower layer and
> have the higher layer depend on it, never the reverse.

---

## Full Runnable Example

| File | What it shows |
|------|---------------|
| `cmd/api/main.go` | The only package that sees every layer — pure wiring, no logic |
| `internal/domain/booking.go` | The shared type, zero imports from this module |
| `internal/repository/booking_repo.go` | Defines `BookingRepository`, ships an in-memory impl — this is what `04-pgx-transactions` would swap for a real `pgxpool.Pool` |
| `internal/service/booking_service.go` | Business rule (`title` required) sitting between HTTP and storage |
| `internal/handler/booking_handler.go` | Gin handlers that only know `service.BookingService`, the interface |
| `internal/config/config.go` | Env var loading, isolated so nothing else needs `os.Getenv` scattered around |

Run it:
```bash
go run ./cmd/api          # PORT=9090 go run ./cmd/api  if 8080 is taken

curl -X POST localhost:8080/bookings -d '{"title":"desk A"}' -H 'Content-Type: application/json'
# {"id":"bk_1","title":"desk A"}

curl localhost:8080/bookings/bk_1
# {"id":"bk_1","title":"desk A"}
```

To see the compiler enforce the DAG yourself: add `import
"lesson09/module-layout/internal/service"` to `booking_repo.go`, reference
anything from it (e.g. `var _ = service.NewBookingService`), and run `go
build ./...`. It refuses immediately, naming the whole chain back to where
it closes the loop — that's the real output above, not a paraphrase.

---

## Comparison Table

| | Django | Go |
|---|---|---|
| Unit of organization | "app" (`myapp/`) — a folder convention only | package — a compiler-recognized unit with its own namespace |
| Circular imports | Allowed; papered over with function-local imports | Forbidden; `go build` fails naming the exact cycle |
| Enforcing "private" code | Nothing — any app can import any other app's internals | `internal/` — compiler-enforced, not just convention |
| Where wiring happens | Django's app registry + DI container (if you use one) | `cmd/api/main.go`, by hand, in explicit order |
| Dependency direction | Whatever imports happen to exist | Chosen up front: domain → repository → service → handler |
| Swapping an implementation | `settings.py` + app registry magic | Change what `main.go` passes to `NewXService(...)` — an interface swap |
