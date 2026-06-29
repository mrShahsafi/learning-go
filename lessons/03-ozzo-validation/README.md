# Lesson 03 — ozzo-validation: Custom Validators, Nested Structs, RFC 9457

## The Django Equivalent

In DRF, serializers do two things in one step: deserialize JSON and validate it.

```python
# serializers.py
class AddressSerializer(serializers.Serializer):
    street = serializers.CharField(required=True)
    city   = serializers.CharField(required=True)
    zip    = serializers.CharField(validators=[validate_zip])

class CreateUserSerializer(serializers.Serializer):
    email    = serializers.EmailField(required=True)
    password = serializers.CharField(validators=[validate_strong_password])
    name     = serializers.CharField(min_length=2, max_length=100)
    address  = AddressSerializer(required=False)

# views.py
def create_user(request):
    s = CreateUserSerializer(data=request.data)
    if not s.is_valid():          # deserialize + validate in one call
        return Response(s.errors, status=422)
    # s.validated_data is now safe
```

---

## The Go Split: Deserialize First, Then Validate

Go does this in **two explicit steps**. This is the most important thing to internalize:

```
ShouldBindJSON(&req)   →  shape + JSON syntax only (no domain rules)
req.Validate()         →  domain rules (required, format, custom)
```

**The trap**: forgetting `req.Validate()` sends unvalidated data straight to your service.
There is no framework safety net.

---

## How ozzo-validation Works

### Basic field validation

```go
import validation "github.com/go-ozzo/ozzo-validation/v4"
import "github.com/go-ozzo/ozzo-validation/v4/is"

func (r CreateUserRequest) Validate() error {
    return validation.ValidateStruct(&r,
        validation.Field(&r.Email, validation.Required, is.Email),
        validation.Field(&r.Name,  validation.Required, validation.Length(2, 100)),
    )
}
```

`ValidateStruct` takes field pointers — it figures out the field name for the error map automatically.

### The error type

```go
err := req.Validate()

// err is a validation.Errors — which is just map[string]error
errs := err.(validation.Errors)
// errs["email"] = "must be a valid email address"
// errs["name"]  = "the length must be between 2 and 100"
```

This map is what you put in the RFC 9457 `extensions.errors` field.

---

## Custom Validators (`validators.go`)

Implement the `validation.Rule` interface — one method:

```go
type StrongPassword struct{}

func (StrongPassword) Validate(value interface{}) error {
    s, _ := value.(string)
    // your logic
    return errors.New("must be 8+ chars with one uppercase and one digit")
}
```

Use it exactly like a built-in rule:

```go
validation.Field(&r.Password, validation.Required, StrongPassword{})
```

The compile-time interface check is a Go senior habit:

```go
var _ validation.Rule = StrongPassword{} // build fails if interface changes
```

---

## Nested Struct Validation (`models.go`)

Make the nested struct implement `validation.Validatable`:

```go
type Address struct { Street, City, Zip string }

func (a Address) Validate() error {
    return validation.ValidateStruct(&a,
        validation.Field(&a.Street, validation.Required),
        validation.Field(&a.City,   validation.Required),
        validation.Field(&a.Zip,    validation.Required, validation.Match(zipRegex)),
    )
}
```

In the parent struct, use a **pointer** for optional nesting:

```go
type CreateUserRequest struct {
    Address *Address `json:"address"` // nil = not provided = skip validation
}

func (r CreateUserRequest) Validate() error {
    return validation.ValidateStruct(&r,
        validation.Field(&r.Address), // ozzo calls addr.Validate() automatically
    )
}
```

- `*Address = nil` → field skipped (optional)
- `*Address = &Address{...}` → ozzo calls `addr.Validate()`
- Errors come back nested: `{"address": {"zip": "must be a 5-digit US zip"}}`

---

## Mapping Errors to RFC 9457 (`problem.go`)

```go
func RespondValidationError(c *gin.Context, err error) {
    errs := err.(validation.Errors) // always type-assert first

    c.JSON(422, ProblemDetail{
        Type:   "https://coordeck.com/problems/validation-error",
        Title:  "Validation Error",
        Status: 422,
        Detail: "One or more fields failed validation.",
        Extensions: map[string]any{
            "errors": flattenErrors(errs),
        },
    })
}

func flattenErrors(errs validation.Errors) map[string]any {
    out := make(map[string]any, len(errs))
    for field, err := range errs {
        if nested, ok := err.(validation.Errors); ok {
            out[field] = flattenErrors(nested) // recurse for nested structs
        } else {
            out[field] = err.Error()
        }
    }
    return out
}
```

---

## Files in This Lesson

| File | What it teaches |
|------|----------------|
| `models.go` | Struct + `Validate()` method · nested struct pointer trick |
| `validators.go` | Custom `validation.Rule` + compile-time interface guard |
| `problem.go` | RFC 9457 `ProblemDetail` + `flattenErrors` for nested maps |
| `handler.go` | The two-step bind → validate pattern in a Gin handler |
| `main.go` | Wire up Gin |

---

## Run It

```bash
cd lessons/03-ozzo-validation
go run .
```

Then in another terminal, try each case:

```bash
# All fields missing → multiple errors
curl -s -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{}' | jq .

# Weak password → password error only
curl -s -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"email":"a@b.com","password":"weak","name":"Alice"}' | jq .

# Bad zip in nested address → nested error
curl -s -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"email":"a@b.com","password":"Strong1!","name":"Alice","address":{"street":"1 Main","city":"SF","zip":"bad"}}' | jq .

# All valid → 201
curl -s -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"email":"a@b.com","password":"Strong1!","name":"Alice","address":{"street":"1 Main","city":"SF","zip":"94105"}}' | jq .
```

---

## Django vs Go Comparison

| | Django DRF | Go + ozzo |
|---|---|---|
| **Deserialize + validate** | `s.is_valid()` — one step | `ShouldBindJSON` then `Validate()` — two steps |
| **Built-in rules** | Field types (`EmailField`, `CharField`) | `is.Email`, `validation.Length`, `validation.Match` |
| **Custom rules** | `validate_*` function or `validators=[...]` | Struct implementing `validation.Rule` |
| **Nested validation** | Nested serializer | Pointer field + `Validatable` interface |
| **Error format** | `serializer.errors` dict | `validation.Errors` (= `map[string]error`) |
| **RFC 9457 mapping** | Manual or `drf-problems` package | Manual `flattenErrors` → `extensions.errors` |
| **Optional nested** | `required=False` on nested serializer | `*Ptr` field — nil skips validation |

---

## The Single Trap for Django Developers

**You can bind JSON successfully and still send garbage to your database.**

```go
// WRONG — req is deserialized but NOT validated
if err := c.ShouldBindJSON(&req); err != nil { ... }
userService.Create(req) // ← unvalidated data!

// RIGHT — always validate after binding
if err := c.ShouldBindJSON(&req); err != nil { ... }
if err := req.Validate(); err != nil { ... } // ← do not skip this
userService.Create(req) // ← now safe
```

DRF's `is_valid()` made you validate or you couldn't access `validated_data`.
Go has no such gate — the discipline is yours.
