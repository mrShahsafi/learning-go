# Lesson 06 — Interface-Based Testing

## The Django Equivalent

Django testing leans on the framework to give you isolation for free:

```python
# test_users.py
from django.test import TestCase
from rest_framework.test import APITestCase
from unittest.mock import patch

class UserServiceTests(TestCase):
    def test_get_user(self):
        user = User.objects.create(email="amir@coordeck.com", name="Amir")
        got = get_user(user.id)
        self.assertEqual(got.email, "amir@coordeck.com")

class UserAPITests(APITestCase):
    def test_create_user(self):
        resp = self.client.post("/users", {"email": "a@b.com", "name": "A"})
        self.assertEqual(resp.status_code, 201)
```

Two different escape hatches do the isolation work:

- `TestCase` wraps every test in a DB transaction and rolls it back — so tests
  don't see each other's data, but you're still hitting a real (test) database.
- `unittest.mock.patch` swaps out a function/attribute at import time when you
  want to avoid the DB entirely — often reaching for `@patch("myapp.services.db.query")`.

Go has neither transactional test rollback nor monkey-patching. Instead, the
**interface is the seam** — and you get isolation by injecting a different
*implementation*, not by patching internals.

---

## The Go Way: fakeRepo, Not Mocks

From [lesson 02](../02-dependency-injection/README.md) you already know
`UserService` depends on the `UserRepo` interface, not a concrete `pgxUserRepo`.
That single decision is the entire testing strategy:

```go
type UserRepo interface {
    FindByID(ctx context.Context, id int64) (*User, error)
    Create(ctx context.Context, u *User) error
    FindByEmail(ctx context.Context, email string) (*User, error)
}
```

In tests, you construct `NewUserService(NewFakeUserRepo())` instead of
`NewUserService(NewUserRepo(dsn))`. `fakeUserRepo` is a real, working
implementation backed by a map — not a mock object that records calls and
plays back stubbed return values. It:

- Actually enforces "duplicate email fails", because that logic lives in the
  fake too, not in a mocking framework's `.return_value` chain.
- Has no assertions about *how* it was called (`.assert_called_once_with(...)`)
  — you assert on the *outcome* (`GetUser` returns the right user, or an error).

This is the senior Go instinct: prefer a small real fake over a mocking
library. There's no `gomock`/`testify/mock` in this lesson on purpose —
you don't need one for most interfaces this small.

---

## Table-Driven Tests Replace `pytest.mark.parametrize`

`user_service_test.go` has `TestUserService_CreateUser` running five cases
through one loop:

```go
tests := []struct {
    name    string
    email   string
    userN   string
    wantErr bool
}{
    {name: "valid input", email: "alice@example.com", userN: "Alice", wantErr: false},
    {name: "empty email", email: "", userN: "Alice", wantErr: true},
    // ...
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        svc := NewUserService(NewFakeUserRepo()) // fresh fake per case
        got, err := svc.CreateUser(context.Background(), tt.email, tt.userN)
        // assert against tt.wantErr
    })
}
```

`t.Run(tt.name, ...)` gives each row its own named subtest — shows up in
`go test -v` as `TestUserService_CreateUser/empty_email`, and `go test -run
TestUserService_CreateUser/empty_email` runs just that one. This is Go's
answer to `@pytest.mark.parametrize("email,name,expect_err", [...])`.

**The trap:** a fresh `NewFakeUserRepo()` is created *inside* each subtest,
not once above the loop. If you hoist it above the loop, "duplicate email"
tests from earlier cases silently pollute later ones — there's no automatic
transaction rollback to save you like Django's `TestCase` gives you.

---

## Testing the HTTP Layer with `httptest`

`user_handler_test.go` is the analogue of `APITestCase` + `self.client.post(...)`:

```go
router, _ := newTestRouter()   // real gin.Engine, fakeUserRepo behind it

req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
req.Header.Set("Content-Type", "application/json")
w := httptest.NewRecorder()

router.ServeHTTP(w, req)       // full round trip: routing, binding, JSON encode

if w.Code != http.StatusCreated { ... }
```

No test client abstraction, no fixtures loaded from a JSON file — `httptest`
gives you a real `http.ResponseWriter` recorder and you drive Gin's actual
`ServeHTTP` router. This exercises routing, `ShouldBindJSON`, your handler,
the service, and the fake repo — the same layers that would run in production,
minus the real database.

---

## The Files in This Lesson

| File | What it teaches |
|------|----------------|
| `user_repo.go` | `UserRepo` interface + `pgxUserRepo` stub + `fakeUserRepo` (the seam) |
| `user_service.go` | Business logic with enough branches to be worth testing (`CreateUser` validation) |
| `user_service_test.go` | Table-driven unit tests against `fakeUserRepo` — no DB, no mocks |
| `user_handler.go` | Gin handlers, unchanged by the testing strategy below them |
| `user_handler_test.go` | `httptest` + a real `gin.Engine` — full HTTP-layer tests |
| `main.go` | Wires the real app (still using `fakeUserRepo` so it runs without Postgres) |

## Run It

```bash
go test ./... -v      # run every test, see each subtest name
go run .              # start the demo server on :8080
```

---

## Django vs Go: Testing

| Concern | Django / DRF | Go |
|---|---|---|
| Isolating from the DB | `unittest.mock.patch` on the ORM/service | Inject a `fakeRepo` implementing the same interface |
| Per-test data isolation | `TestCase` wraps each test in a rolled-back transaction | Construct a fresh fake per test/subtest — no framework magic |
| Parametrized cases | `@pytest.mark.parametrize(...)` | Table-driven test: `[]struct{...}` + `for` loop + `t.Run` |
| HTTP-layer tests | `APITestCase` + `self.client.post(...)` | `httptest.NewRecorder()` + real `router.ServeHTTP()` |
| Asserting call behavior | `mock.assert_called_once_with(...)` | Not idiomatic — assert on outcomes, not call internals |
| Running one case | `pytest -k test_name` | `go test -run TestName/subtest_name` |
