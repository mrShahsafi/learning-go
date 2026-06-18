# Lesson 01 — Django → Go/Gin: What Actually Bites You

**Stack context**: Go/Gin · ozzo-validation · RFC 9457 errors · PostgreSQL 15

---

## 1. Errors are values, not exceptions — everywhere

In Django/DRF you `raise`, and the framework catches. In Go, every function returns `(T, error)`, and if you don't check it, nothing complains — it just silently does the wrong thing.

```go
// The Go contract you must follow in every handler:
doc, err := repo.FindByID(id)
if err != nil {
    c.JSON(404, buildEnvelope("document.not_found", requestID))
    return  // ← this return is easy to forget and causes double-writes
}
c.JSON(200, doc)
```

**Pattern**: Define sentinel errors at the repo layer and map them in handlers.

```go
// repo layer
var ErrNotFound = errors.New("not.found")

// handler layer
if errors.Is(err, repo.ErrNotFound) {
    c.JSON(404, buildEnvelope("document.not_found", requestID))
    return
}
```

---

## 2. No ORM means your schema and structs can silently diverge

Django owns your schema — models → migrations → always in sync. In Go with pgx, your PostgreSQL schema and Go structs are **completely independent**. Add a column to the DB and forget the struct? No error — the data is just silently ignored.

**Fix**: use [sqlc](https://sqlc.dev) — it generates type-safe Go from your SQL queries, so your structs are guaranteed to match your schema at compile time. Pairs perfectly with contract-first philosophy (schema is source of truth, code is generated).

---

## 3. Middleware is a HandlerFunc, not a class

Django middleware is a class wired in `settings.py`. Gin middleware is just a function that calls `c.Next()`, applied per route group:

```go
func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        id := uuid.New().String()
        c.Set("request_id", id)       // your RFC 9457 field
        c.Header("X-Request-ID", id)
        c.Next()
    }
}

r.Use(RequestID())           // global
api := r.Group("/api/v1")
api.Use(AuthRequired())      // only authenticated routes
```

---

## 4. Real concurrency — no GIL protecting you

Every Gin request runs in its own goroutine simultaneously. In Django, the GIL serializes Python execution so accidental shared state often "works" badly but doesn't race. In Go, shared mutable state (in-memory caches, counters) will corrupt silently or panic.

- `pgxpool.Pool` is goroutine-safe — your DB calls are fine.
- Anything shared across requests needs a `sync.RWMutex` or `sync.Map`.

```bash
go test -race ./...   # run this regularly — catches races at test time, not production
```

---

## 5. No settings.py — config is a struct you build once

```go
type Config struct {
    DatabaseURL string `env:"DATABASE_URL,required"`
    JWTSecret   string `env:"JWT_SECRET,required"`
    Port        string `env:"PORT" envDefault:"8080"`
}
```

Use [caarlos0/env](https://github.com/caarlos0/env) to populate it from environment variables at startup. A bad env var panics on boot — not mid-request. Pass `cfg` down via dependency injection, never as a global.

---

## 6. Testing Gin handlers without a server

Django has `TestClient` built in. Gin uses the stdlib `httptest` package — no server needed:

```go
func TestCreateDocument(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := setupRouter()

    body := `{"title": "Q3 Report"}`
    req, _ := http.NewRequest("POST", "/api/v1/documents", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    assert.Equal(t, 201, w.Code)
}
```

**Senior tip**: write a small helper `assertMessageCode(t, w.Body, "document.validation.title.required")` — your RFC 9457 dot-notation codes become your test vocabulary, far more meaningful than asserting status codes alone.

---

## The meta-lesson

Go is Django with all the implicit magic removed. The patterns you've already chosen — explicit validation, structured error codes, contract-first specs — are exactly the Go philosophy. You're not fighting the language, you're already thinking in it.

---

## Django vs Go Comparison Table

| Concept | Django | Go/Gin |
|---|---|---|
| Error handling | `raise` / middleware catches | `(T, error)` return — you check |
| Schema sync | Models drive migrations | Schema and structs are independent |
| Middleware | Class in `settings.py` | `HandlerFunc` applied to route group |
| Concurrency | GIL serializes Python | Real goroutines — races are real |
| Config | `settings.py` | Env-populated struct, injected |
| Testing HTTP | `TestClient` | `httptest.NewRecorder()` |
