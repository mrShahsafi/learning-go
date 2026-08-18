# Lesson 10 — Middleware Patterns in Gin

## The Django Equivalent

Django middleware is a class or callable wrapped around the whole request:

```python
class AuthMiddleware:
    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        if request.headers.get("Authorization") != "Bearer dev-token":
            return JsonResponse({"title": "Unauthorized"}, status=401)

        response = self.get_response(request)
        return response
```

Returning a response stops the chain. Settings determine the global order,
and decorators or DRF permission classes usually handle route-specific work.

## The Go/Gin Reality

A Gin middleware is only a `gin.HandlerFunc`:

```go
type HandlerFunc func(*gin.Context)
```

The middleware and endpoint handlers form one ordered slice:

```text
request
  → recovery before c.Next()
    → auth before c.Next()
      → timeout before c.Next()
        → endpoint
      ← timeout after c.Next()
    ← auth after c.Next()
  ← recovery after c.Next()
response
```

`c.Next()` runs the remaining handlers now, then returns so middleware can do
post-processing. That is how logging middleware measures the endpoint and
then records its status. `c.Abort()` marks the chain so later handlers are
skipped.

Middleware can be attached at three scopes:

```go
router.Use(ProblemRecovery())                         // every route
private := router.Group("/private", Auth("token"))   // one route group
router.GET("/slow", Timeout(time.Second), handler)   // one route
```

### 1. Auth: Abort the Chain

`Auth` reads the bearer token at the HTTP trust boundary. Invalid input gets
an RFC 9457 response and the protected handler never runs:

```go
provided, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
if !ok || subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
    c.Header("Content-Type", "application/problem+json")
    c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
        "type":   "https://api.coordeck.com/problems/unauthorized",
        "title":  "Unauthorized",
        "status": http.StatusUnauthorized,
    })
    return
}
c.Next()
```

The example uses constant-time comparison because authentication values
should not use an ordinary string comparison. A production system would
usually validate a signed access token rather than compare one static token;
the chain control is the same.

### 2. Timeout: Propagate Cancellation

The timeout middleware tightens the request context and replaces the context
on the `*http.Request`:

```go
ctx, cancel := context.WithTimeout(c.Request.Context(), duration)
defer cancel()

c.Request = c.Request.WithContext(ctx)
c.Next()
```

This does not forcibly kill a handler. Go has no safe "stop this goroutine"
operation. The handler, service, and pgx calls must use the propagated context:

```go
select {
case <-c.Request.Context().Done():
    c.JSON(http.StatusGatewayTimeout, problem)
case result := <-service.DoWork(c.Request.Context()):
    c.JSON(http.StatusOK, result)
}
```

pgx already observes `ctx.Done()` when called with that context. If a layer
replaces it with `context.Background()`, the timeout chain is silently broken.

Do not implement a timeout by running `c.Next()` in another goroutine and
writing a `504` concurrently. `gin.Context` and its response writer are not
safe for concurrent writes. Propagate the deadline and make work cooperative.

### 3. Recovery: Reuse Gin's Native Middleware

Gin already provides `CustomRecovery`, so the lesson only supplies the RFC
9457 mapping:

```go
return gin.CustomRecovery(func(c *gin.Context, _ any) {
    c.Header("Content-Type", "application/problem+json")
    c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
        "type":   "https://api.coordeck.com/problems/internal-server-error",
        "title":  "Internal Server Error",
        "status": http.StatusInternalServerError,
    })
})
```

Recovery belongs near the outside of the stack so it can catch panics from
route middleware and endpoints. Gin logs the panic server-side; the response
deliberately omits it because secrets and stack traces must never reach the
client.

## The Single Trap That Bites Django Developers Hardest

> **`c.Abort()` stops later handlers, but it does not return from the current
> middleware function.** Django stops when middleware returns a response. In
> Gin, always keep `c.AbortWithStatusJSON(...)` and `return` together. Without
> the `return`, code below the abort still runs and can mutate state or write a
> second response even though the endpoint itself was skipped.

## Run It

```bash
go test ./...
go run .

curl localhost:8080/private/bookings
# 401 RFC 9457 response

curl -H 'Authorization: Bearer dev-token' localhost:8080/private/bookings
# {"bookings":["desk A","room B"]}

curl localhost:8080/slow
# 504 after the propagated deadline

curl localhost:8080/panic
# 500 RFC 9457 response; the server stays alive
```

## Files

| File | What it demonstrates |
|---|---|
| `middleware.go` | Isolated auth, timeout, and recovery middleware |
| `middleware_test.go` | Real `httptest` requests through a Gin engine |
| `main.go` | Global, route-group, and single-route middleware registration |

## Comparison Table

| | Django / DRF | Go / Gin |
|---|---|---|
| Middleware shape | Callable or class wrapping `get_response` | `func(*gin.Context)` |
| Continue chain | Call `get_response(request)` | Call `c.Next()` |
| Stop chain | Return a response | `c.Abort...()` and then `return` |
| Global registration | Ordered `MIDDLEWARE` setting | Ordered `router.Use(...)` calls |
| Route-specific behavior | Decorator, permission class, or view logic | Handler argument or `Group(...).Use(...)` |
| Request state | Attributes on `HttpRequest` | `gin.Context` for handler-local values; `Request.Context()` for cancellation |
| Timeout | Server/framework timeout plus cooperative I/O | `context.WithTimeout` propagated through every layer |
| Panic/exception recovery | Django exception middleware | `gin.Recovery()` or `gin.CustomRecovery...()` |
| Test client | Django/DRF test client | `httptest` plus a real `gin.Engine` |
