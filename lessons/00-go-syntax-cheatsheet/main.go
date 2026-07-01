// Lesson 00 — run every section and read the printed output next to README.md.
package main

import (
	"errors"
	"fmt"
)

// ---------- structs, methods, embedding ----------

type Animal struct {
	Name string
}

func (a Animal) Speak() string { // value receiver: gets a copy
	return a.Name + " makes a sound"
}

type Dog struct {
	Animal // embedding == Django-ish "inheritance" but it's composition
	Breed  string
}

func (d Dog) Speak() string { // "overrides" Animal.Speak — no virtual dispatch, just method resolution
	return d.Name + " barks"
}

// ---------- interfaces ----------

type Speaker interface {
	Speak() string
}

// ---------- pointer vs value receiver ----------

type Counter struct{ n int }

func (c *Counter) Inc() { c.n++ } // pointer receiver: mutates the original

// ---------- generics (Go 1.18+) ----------

func Map[T, U any](in []T, f func(T) U) []U {
	out := make([]U, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

// ---------- custom error + errors.Is/As ----------

type NotFoundError struct{ ID int }

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("id %d not found", e.ID)
}

func findUser(id int) error {
	if id != 1 {
		return fmt.Errorf("findUser: %w", &NotFoundError{ID: id}) // %w wraps, keeps the chain
	}
	return nil
}

func main() {
	fmt.Println("== structs & embedding ==")
	d := Dog{Animal: Animal{Name: "Rex"}, Breed: "Lab"}
	fmt.Println(d.Speak())     // Dog's own method wins
	fmt.Println(d.Animal.Name) // promoted field, also just d.Name works

	fmt.Println("\n== interfaces ==")
	var s Speaker = d // Dog satisfies Speaker implicitly — no "implements" keyword
	fmt.Println(s.Speak())

	fmt.Println("\n== pointer receiver mutation ==")
	c := Counter{}
	c.Inc()
	c.Inc()
	fmt.Println("count:", c.n) // 2 — Go auto-takes &c because Inc has a pointer receiver

	fmt.Println("\n== slices vs arrays ==")
	arr := [3]int{1, 2, 3}    // fixed size, part of the type
	sl := []int{1, 2, 3}      // dynamic, backed by an array
	sl = append(sl, 4)        // may or may not reallocate — see README "slice gotchas"
	fmt.Println(arr, sl, len(sl), cap(sl))

	fmt.Println("\n== maps ==")
	m := map[string]int{"a": 1}
	v, ok := m["missing"] // zero value + ok, never panics on missing key
	fmt.Println(v, ok)

	fmt.Println("\n== generics ==")
	doubled := Map([]int{1, 2, 3}, func(n int) int { return n * 2 })
	fmt.Println(doubled)

	fmt.Println("\n== error wrapping ==")
	err := findUser(42)
	var nf *NotFoundError
	if errors.As(err, &nf) { // unwraps the chain looking for *NotFoundError
		fmt.Println("caught NotFoundError, id =", nf.ID)
	}
	fmt.Println("full error:", err)

	fmt.Println("\n== defer, panic, recover ==")
	safeDivide(10, 0)

	fmt.Println("\n== goroutines + channels ==")
	ch := make(chan int)
	go func() { ch <- 21 * 2 }() // launch, send result on channel
	fmt.Println("from goroutine:", <-ch)

	fmt.Println("\n== closures capture by reference ==")
	counters := makeCounters(3)
	for _, next := range counters {
		fmt.Println(next(), next()) // each closure has its OWN n
	}
}

func safeDivide(a, b int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered:", r)
		}
	}()
	fmt.Println(a / b) // divide by zero panics; recover() catches it above
}

func makeCounters(n int) []func() int {
	fns := make([]func() int, n)
	for i := range fns {
		count := 0
		fns[i] = func() int { // closes over its own `count`, not shared
			count++
			return count
		}
	}
	return fns
}
