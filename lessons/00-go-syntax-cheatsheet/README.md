# Lesson 00 — Go Syntax Cheat Sheet

This is not a lesson with a single takeaway — it's the **reference you come back to**
when you've forgotten a piece of syntax. Everything demoed here compiles and runs
in `main.go`. Run it once, then use this file to scan for what you forgot.

```bash
cd lessons/00-go-syntax-cheatsheet
go run .
```

---

## 1. Terminology: Django/Python → Go

| Python / Django | Go |
|---|---|
| `pip` / `requirements.txt` | `go mod` / `go.mod` + `go.sum` |
| `venv` | none needed — modules are per-project already |
| `class` | `struct` + methods (no classes, no inheritance) |
| inheritance | **composition** via struct embedding |
| `self` | explicit receiver name (`func (u *User) ...`), you choose the name |
| duck typing | interfaces — but checked at **compile time** |
| `None` | `nil` (only valid for pointers, slices, maps, chans, funcs, interfaces) |
| `try/except` | no exceptions — functions **return** an `error` value |
| `raise` | `panic()` — reserved for truly unrecoverable situations, not control flow |
| list | slice (`[]T`) — dynamic, like a Python list |
| tuple | fixed-size array (`[N]T`) — rare in practice |
| dict | `map[K]V` |
| `f"{x}"` / `.format()` | `fmt.Sprintf("%v", x)` |
| decorator | no equivalent — use middleware (Gin) or wrap functions manually |
| `async def` / `await` | goroutines (`go f()`) + channels, no async/await syntax |
| `*args, **kwargs` | variadic params (`args ...T`); no kwargs — use a struct or functional options |
| type hints (optional) | static types (mandatory, checked by compiler) |
| `__init__` | a `NewThing(...) *Thing` constructor function (convention, not a keyword) |
| property/getter | none — just export the field (`Name` not `name`) or write a plain method |

---

## 2. Packages, Files, Visibility

```go
package main // every file starts with a package declaration

import (
    "fmt"        // stdlib
    "os"

    "github.com/gin-gonic/gin" // third-party — module path, not a package "name"
)
```

- **Capitalized identifier → exported** (public). `func GetUser` is visible outside the package.
- **lowercase identifier → unexported** (package-private). `func getUser` is invisible outside.
- There's no `__init__.py`. A directory **is** a package; every file in it shares the same package name.
- No `__all__`, no `if __name__ == "__main__"` — the `main` package's `func main()` is the entry point.

---

## 3. Variables & Constants

```go
var x int              // zero value: 0
var y = 5              // type inferred: int
z := 5                 // short form, only inside functions, type inferred
var name string        // zero value: ""
var ok bool             // zero value: false
var p *int              // zero value: nil

const Pi = 3.14159      // compile-time constant
const (
    StatusActive   = "active"   // grouped constants
    StatusInactive = "inactive"
)

// iota — Go's auto-incrementing enum helper
type Status int
const (
    Pending Status = iota // 0
    Approved               // 1
    Rejected               // 2
)
```

**Zero values matter** — Go never leaves a variable "undefined." Every type has a
default: `0`, `""`, `false`, `nil`. There's no `NameError`.

---

## 4. Basic Types & Conversion

| Type | Notes |
|---|---|
| `int`, `int8/16/32/64`, `uint*` | `int` is 64-bit on modern platforms, but treat it as platform-dependent |
| `float32`, `float64` | use `float64` unless you have a specific reason not to |
| `string` | immutable, UTF-8 bytes under the hood |
| `bool` | `true` / `false` |
| `byte` | alias for `uint8` |
| `rune` | alias for `int32`, represents a single Unicode code point |

```go
i := 42
f := float64(i)       // explicit conversion required — NO implicit int→float
s := fmt.Sprintf("%d", i) // int → string via formatting, NOT string(i) (that gives a rune!)
n, err := strconv.Atoi("42") // string → int, always returns (value, error)
```

There is **no implicit type coercion** anywhere in Go. `1 + 1.0` is a compile error
unless both sides are literals.

---

## 5. Structs & Methods

```go
type User struct {
    ID    int64
    Name  string
    email string // unexported — package-private, not visible via JSON either
}

func NewUser(name string) *User { // constructor convention
    return &User{Name: name}
}

// Value receiver — gets a COPY, can't mutate the original
func (u User) Greet() string { return "Hi " + u.Name }

// Pointer receiver — operates on the ORIGINAL, can mutate
func (u *User) Rename(name string) { u.Name = name }
```

**Rule of thumb**: if any method on the type needs a pointer receiver, make them
*all* pointer receivers, for consistency. Go auto-takes the address (`(&u).Rename(...)`)
when you call a pointer-receiver method on an addressable value.

### Struct tags (used by `encoding/json`, `ozzo-validation`, etc.)

```go
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required"`
    Email string `json:"email"`
}
```

---

## 6. Embedding (composition, not inheritance)

```go
type Animal struct{ Name string }
func (a Animal) Speak() string { return a.Name + " makes a sound" }

type Dog struct {
    Animal      // embedded — no field name, "promotes" Animal's fields/methods
    Breed string
}

d := Dog{Animal: Animal{Name: "Rex"}, Breed: "Lab"}
d.Name        // promoted field — same as d.Animal.Name
d.Speak()     // promoted method — same as d.Animal.Speak()
```

If `Dog` defines its own `Speak()`, it shadows the embedded one — there's no
`super().speak()` call, but you can still reach it explicitly via `d.Animal.Speak()`.

---

## 7. Interfaces (implicit — no `implements`)

```go
type Speaker interface {
    Speak() string
}

// Dog satisfies Speaker automatically because it HAS a Speak() string method.
// No declaration needed anywhere.
var s Speaker = Dog{Animal: Animal{Name: "Rex"}}
```

- Interfaces are satisfied **structurally**, like Python duck typing, but verified
  at compile time.
- Small interfaces are idiomatic: `io.Reader` is one method. Define interfaces at
  the **consumer**, not the implementer (a repo package shouldn't define its own interface —
  the service package that *uses* the repo does).
- **The nil-interface trap**: an interface holding a `nil` pointer is NOT itself `nil`.
  ```go
  var p *MyError = nil
  var err error = p
  err == nil // false! err has a type (*MyError) even though the pointer is nil
  ```

---

## 8. Pointers

```go
x := 5
p := &x        // p is *int, holds the address of x
*p = 10        // dereference to read/write through the pointer
fmt.Println(x) // 10

func modify(n *int) { *n = 99 }
modify(&x)
```

- Go has pointers but **no pointer arithmetic** — you can't do `p + 1`.
- Passing a pointer avoids copying; passing a value copies. Structs are copied by
  value by default (unlike Python objects, which are always references).
- `new(T)` allocates a zeroed `T` and returns `*T` — rarely used directly; prefer
  `&T{}` literal syntax.

---

## 9. Slices, Arrays, Maps

```go
arr := [3]int{1, 2, 3}     // array: fixed length is part of the TYPE ([3]int != [4]int)
sl := []int{1, 2, 3}       // slice: dynamic, backed by an array, has len + cap
sl2 := make([]int, 0, 10)  // len=0, cap=10 — pre-allocate to avoid reallocations

sl = append(sl, 4)         // may reallocate if cap is exceeded — reassign the result!
sub := sl[1:3]             // slicing: shares the SAME backing array — mutating sub mutates sl

m := map[string]int{"a": 1}
v, ok := m["missing"]      // zero value + ok bool, never panics on a missing key
delete(m, "a")

var nilMap map[string]int  // nil map: reading is safe (zero value), WRITING panics
var nilSlice []int         // nil slice: len/append both safe, == nil is true
```

**The #1 slice gotcha**: slicing shares memory. `sub := sl[1:3]; sub[0] = 99` mutates
`sl` too. If you need an independent copy, use `copy()`:
```go
dst := make([]int, len(sl))
copy(dst, sl)
```

---

## 10. Control Flow

```go
// if — no parens, braces mandatory
if x > 0 {
    // ...
} else if x < 0 {
    // ...
} else {
    // ...
}

// if with a scoped init statement — extremely common with errors
if err := doThing(); err != nil {
    return err
}

// for — the ONLY loop keyword in Go (no while, no do-while)
for i := 0; i < 10; i++ { }   // classic
for x > 0 { x-- }              // while-style
for { break }                  // infinite loop

// range — like Python's for-in / enumerate
for i, v := range slice {}     // index + value
for k, v := range someMap {}   // key + value (iteration order is RANDOM, unlike dict in py3.7+)
for i := range slice {}        // index only
for range ch {}                 // just the values, no index

// switch — no fallthrough by default (opposite of C/JS!)
switch status {
case "active":
    // ...
case "inactive", "banned": // comma = OR
    // ...
default:
    // ...
}

// switch with no condition = cleaner if/else chain
switch {
case x > 100:
case x > 10:
default:
}

// type switch — checking an interface's concrete type
switch v := someInterface.(type) {
case int:
case string:
default:
}
```

---

## 11. Functions

```go
func add(a, b int) int { return a + b }

func divmod(a, b int) (int, int) { return a / b, a % b } // multiple return values
q, r := divmod(10, 3)

func divide(a, b int) (result int, err error) { // named returns
    if b == 0 {
        err = errors.New("division by zero")
        return // "naked" return — returns current values of result, err
    }
    result = a / b
    return
}

func sum(nums ...int) int { // variadic — like Python's *args
    total := 0
    for _, n := range nums {
        total += n
    }
    return total
}
sum(1, 2, 3)
sum(mySlice...) // spread a slice into variadic args

// functions are values — assign, pass, return them
add := func(a, b int) int { return a + b }
func makeAdder(x int) func(int) int {
    return func(y int) int { return x + y } // closure — captures x
}
```

**No default arguments, no kwargs.** For "optional config," Go uses either a
config struct or the **functional options pattern**:
```go
type Option func(*Server)
func WithPort(p int) Option { return func(s *Server) { s.port = p } }
NewServer(WithPort(8080))
```

---

## 12. Error Handling (no exceptions)

```go
// error is just an interface: type error interface { Error() string }

func doThing() error {
    if somethingWrong {
        return errors.New("something wrong")
    }
    return nil // nil error == success, ALWAYS check this
}

// wrapping preserves the chain — like Python's `raise X from Y`
func outer() error {
    if err := doThing(); err != nil {
        return fmt.Errorf("outer failed: %w", err) // %w wraps, %v would NOT
    }
    return nil
}

// custom error types
type NotFoundError struct{ ID int }
func (e *NotFoundError) Error() string { return fmt.Sprintf("id %d not found", e.ID) }

// unwrapping the chain
var nf *NotFoundError
if errors.As(err, &nf) { /* err OR anything it wraps is *NotFoundError */ }
if errors.Is(err, sql.ErrNoRows) { /* sentinel error comparison */ }
```

**The single biggest habit to build**: after almost every function call that
returns an error, the next line is `if err != nil { ... }`. This isn't boilerplate
to Go developers — it's the whole error-handling model. There's no `try/except`
to catch it three layers up by accident.

---

## 13. defer, panic, recover

```go
func readFile() {
    f, _ := os.Open("x.txt")
    defer f.Close() // runs when readFile RETURNS, no matter how (return, panic)
    // ... use f
}

// defer runs in LIFO order, args are evaluated when defer is CALLED, not when it runs
for i := 0; i < 3; i++ {
    defer fmt.Println(i) // prints 2, 1, 0
}

// panic/recover — NOT try/except. Reserve for programmer errors, not expected failures.
func safeDivide(a, b int) {
    defer func() {
        if r := recover(); r != nil {
            fmt.Println("recovered:", r)
        }
    }()
    fmt.Println(a / b) // divide by zero panics
}
```

Use `error` returns for anything expected (not found, invalid input, DB down).
Use `panic` only for "this should be impossible" (nil pointer you control, programmer
bug). Gin's own middleware already recovers panics for you in HTTP handlers.

---

## 14. Goroutines & Channels

```go
go doSomething() // launches a goroutine — fire and forget, NOT a thread pool task, real (green) thread

ch := make(chan int)       // unbuffered — send blocks until someone receives
ch2 := make(chan int, 10)  // buffered — send blocks only when full

ch <- 5      // send
v := <-ch    // receive (blocks until a value arrives)
v, ok := <-ch // ok is false if channel is closed AND drained

close(ch)    // signals "no more values" — only the SENDER should close

// select — like a switch for channel operations
select {
case v := <-ch1:
    fmt.Println(v)
case ch2 <- 5:
    fmt.Println("sent")
default:
    fmt.Println("no channel ready")
}

// the standard concurrency safety primitives
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    // work
}()
wg.Wait() // blocks until all Done() calls happen

var mu sync.Mutex
mu.Lock()
// critical section
mu.Unlock()
```

There's no GIL. Goroutines genuinely run in parallel across CPU cores. Shared
mutable state **will** race unless protected — `go run -race ./...` catches this.

---

## 15. Generics (Go 1.18+)

```go
func Map[T, U any](in []T, f func(T) U) []U {
    out := make([]U, len(in))
    for i, v := range in {
        out[i] = f(v)
    }
    return out
}
doubled := Map([]int{1, 2, 3}, func(n int) int { return n * 2 })

// constraints narrow `any` to types supporting specific operations
type Number interface {
    int | int64 | float64
}
func Sum[T Number](nums []T) T {
    var total T
    for _, n := range nums {
        total += n
    }
    return total
}
```

Used sparingly in most Go codebases — prefer concrete types unless you're writing
genuinely reusable container/algorithm code.

---

## 16. Formatting Verbs (`fmt`)

| Verb | Meaning |
|---|---|
| `%v` | default format for any value |
| `%+v` | default format, but includes struct field names |
| `%#v` | Go-syntax representation (useful for debugging) |
| `%T` | the type of the value |
| `%d` | integer |
| `%s` | string |
| `%f` | float (`%.2f` for 2 decimal places) |
| `%t` | bool |
| `%p` | pointer address |
| `%q` | quoted string |
| `%w` | **only** valid inside `fmt.Errorf`, wraps an error |

```go
fmt.Printf("%+v\n", user)        // {ID:1 Name:Amir email:}
fmt.Println(user)                 // same as %v, adds newline
s := fmt.Sprintf("id=%d", user.ID) // build a string instead of printing
```

---

## 17. Testing (stdlib `testing`, no pytest)

```go
// user_test.go — MUST end in _test.go, lives next to the code it tests
package main

import "testing"

func TestAdd(t *testing.T) {
    got := add(2, 3)
    if got != 5 {
        t.Errorf("add(2,3) = %d; want 5", got) // Errorf: mark failed, keep running
    }
}

// table-driven tests — the idiomatic Go pattern for multiple cases
func TestAddCases(t *testing.T) {
    cases := []struct {
        a, b, want int
    }{
        {2, 3, 5},
        {0, 0, 0},
        {-1, 1, 0},
    }
    for _, tc := range cases {
        got := add(tc.a, tc.b)
        if got != tc.want {
            t.Errorf("add(%d,%d) = %d; want %d", tc.a, tc.b, got, tc.want)
        }
    }
}
```

```bash
go test ./...          # run all tests
go test -v ./...       # verbose
go test -run TestAdd   # run one test by name
go test -race ./...    # catch data races
```

---

## 18. Gotchas That Bite Django/Python Developers Specifically

| Gotcha | What happens | Fix |
|---|---|---|
| Loop variable capture (pre-Go 1.22) | closures in a `for` loop all saw the *same* final `i` | Go 1.22+ fixed this — each iteration gets its own `i`. On older Go, shadow it: `i := i` |
| Nil interface ≠ nil value | `var err error = (*MyErr)(nil); err != nil` is `true` | Return a plain untyped `nil`, not a typed nil pointer |
| Struct copies by value | passing a struct to a function copies it; mutations don't propagate | pass `*Struct` if you need mutation to be visible |
| Slice aliasing | sub-slices share the backing array | `copy()` when you need independence |
| `map` iteration order | random every run | sort keys explicitly if you need determinism |
| No method overloading | can't have two `Foo(int)` / `Foo(string)` | different function names, or variadic/`any` params |
| Unused imports/variables | **compile error**, not a warning | remove it — Go won't build otherwise |
| `==` on structs | works only if all fields are comparable (no slices/maps/funcs inside) | use `reflect.DeepEqual` or write an `Equals` method |

---

## 19. Useful Stdlib Packages (the ones you'll actually reach for)

| Package | Django/Python equivalent | Use for |
|---|---|---|
| `fmt` | `print` / f-strings | formatting, printing |
| `errors` | exceptions | wrapping/inspecting errors |
| `context` | request-scoped state, but explicit | cancellation, deadlines, passing `request_id` |
| `encoding/json` | `json` module | (de)serialization |
| `net/http` | Django's request/response cycle, lower level | HTTP servers/clients (Gin sits on top) |
| `strconv` | `int()`, `str()` | string ↔ number conversion |
| `strings` | `str` methods | string manipulation |
| `time` | `datetime` | dates, durations, timers |
| `log/slog` | `logging` module | structured logging (Go 1.21+) |
| `sync` | `threading` | mutexes, wait groups |
| `os` | `os` | env vars, files, args |
| `testing` | `pytest` / `unittest` | tests, benchmarks |

---

## What's Next

This cheat sheet is meant to be re-read, not memorized in one pass. When a lesson
uses syntax you don't recognize, come back here first — if it's still not covered,
that's a sign we should add a proper lesson for it. See `agents.md` for the running
lesson index and the next-topics queue.
