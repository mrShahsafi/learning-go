# Lesson 02 — Dependency Injection Without a Framework

## The Django Equivalent

In Django you never think about DI — it's invisible:

```python
# views.py — just import and go, Django wired everything
from .models import User

def get_user(request, pk):
    user = User.objects.get(pk=pk)   # ORM talks to DB via connection from settings.py
    return JsonResponse({"id": user.id, "email": user.email})
```

The framework handles the wiring: `settings.py` → `DATABASES` → ORM → your view.  
You never pass a DB connection anywhere. It's global and implicit.

---

## The Go Equivalent

Go has **no settings.py, no INSTALLED_APPS, no AppConfig magic**.  
You build the wiring yourself — once, in `main.go`.

The pattern is a chain:

```
Config  →  DB pool  →  Repo  →  Service  →  Handler  →  Router
```

Each layer receives only what it needs via its constructor (`NewXxx(dep)`).  
Nothing is global. Nothing is implicit.

---

## The Files in This Lesson

| File | What it teaches |
|------|----------------|
| `config.go` | Replacing `settings.py` with an explicit `Config` struct |
| `user_repo.go` | The DB layer + the interface trick that enables testing |
| `user_service.go` | Business logic that depends on an interface, not a concrete type |
| `user_handler.go` | Gin handler that depends on a service, not a repo |
| `main.go` | The **one place** that wires everything together |

---

## Run It

```bash
cd lessons/02-dependency-injection
go run .
# In another terminal:
curl http://localhost:8080/users/1
curl http://localhost:8080/users/2
curl http://localhost:8080/users/99   # → 404
```

To use a real DB, change one line in `main.go`:
```go
// from:
repo := NewFakeUserRepo()
// to:
repo := NewUserRepo(cfg.DatabaseURL)
```

---

## The Interface Trick — Why It Matters

The single most important pattern in `user_repo.go` is this:

```go
// Define the contract
type UserRepo interface {
    FindByID(ctx context.Context, id int64) (*User, error)
}

// Two implementations of the same contract:
type pgxUserRepo struct { ... }   // production — talks to Postgres
type fakeUserRepo struct { ... }  // tests — uses a map in memory
```

`UserService` depends on `UserRepo` (the interface), not `pgxUserRepo` (the concrete type).  
This means in tests you can do:

```go
func TestGetUser_NotFound(t *testing.T) {
    repo := &fakeUserRepo{data: map[int64]*User{}}  // empty DB, no connection needed
    svc  := NewUserService(repo)
    _, err := svc.GetUser(context.Background(), 999)
    if err == nil {
        t.Fatal("expected error")
    }
}
```

No Docker. No test database. No mocking library. Just a plain struct.

---

## The Constructor Pattern

Every layer follows the same shape:

```go
type SomeLayer struct {
    dep *Dependency
}

func NewSomeLayer(dep *Dependency) *SomeLayer {
    return &SomeLayer{dep: dep}
}
```

This is Go's idiomatic DI — no framework, no reflection, no magic tags.  
The compiler enforces it: if you forget to pass `dep`, it won't build.

---

## Django vs Go Comparison

| Concern | Django | Go |
|---------|--------|----|
| Config | `settings.py` — global module, loaded by framework | `Config` struct — explicit, passed in `main()` |
| DB connection | `django.db.connection` — implicit global | `*pgxpool.Pool` — explicit field on repo struct |
| Wiring | `INSTALLED_APPS` + `AppConfig` — framework does it | `main.go` — you do it, once, in one place |
| Repo layer | `Model.objects` — baked into ORM | Plain struct with methods; you define the query |
| Service layer | Usually a `services.py` module with free functions | Struct with methods; deps injected via constructor |
| Testability | Requires `django.test.TestCase` + test DB setup | Swap in a `fakeRepo` struct — zero infrastructure |
| What's implicit | Almost everything (DB, ORM, middleware, signals) | Nothing — if it's needed, you pass it in |
| Circular import risk | Low (Django resolves for you) | Real risk if you wire in the wrong layer |

---

## The Single Trap for Django Developers

**Using package-level global variables instead of constructors.**

Django trains you to do this:
```python
# In any file, just import — it "just works"
from django.conf import settings
db_url = settings.DATABASES["default"]["NAME"]
```

In Go you might try:
```go
// ❌ DO NOT DO THIS
var DB *pgxpool.Pool  // package-level global in db.go

func init() {
    DB, _ = pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
}
```

This looks convenient but breaks badly:
- Tests can't swap out the DB — the global is set at `init()` time
- Parallel tests share mutable state and interfere with each other
- The `init()` order across packages is surprising and hard to control
- Errors during `init()` can only `panic` — no graceful handling

**Always pass dependencies explicitly. If it's needed, it's a constructor argument.**

---

## What's Next

With DI patterns solid, the natural follow-ups are:

- **Lesson 03** — Interface-based testing: writing the `fakeRepo` tests for real
- **Lesson 04** — pgx transaction patterns: passing `pgx.Tx` through the same DI chain
