package main

import (
	"context"
	"fmt"
)

// UserRepo is the seam. The service depends on this interface, never on a
// concrete DB type. That single decision is what makes everything below
// testable without spinning up Postgres.
type UserRepo interface {
	FindByID(ctx context.Context, id int64) (*User, error)
	Create(ctx context.Context, u *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
}

// pgxUserRepo is the real implementation. In Coordeck this would hold a
// *pgxpool.Pool and run real SQL — see lesson 04. We stub the queries here
// so this lesson compiles and runs without a live database.
type pgxUserRepo struct {
	dsn string
}

func NewUserRepo(dsn string) UserRepo {
	return &pgxUserRepo{dsn: dsn}
}

func (r *pgxUserRepo) FindByID(_ context.Context, id int64) (*User, error) {
	return nil, fmt.Errorf("pgxUserRepo: no real DB in this lesson (id=%d)", id)
}

func (r *pgxUserRepo) Create(_ context.Context, u *User) error {
	return fmt.Errorf("pgxUserRepo: no real DB in this lesson (email=%s)", u.Email)
}

func (r *pgxUserRepo) FindByEmail(_ context.Context, email string) (*User, error) {
	return nil, fmt.Errorf("pgxUserRepo: no real DB in this lesson (email=%s)", email)
}

// fakeUserRepo is an in-memory stand-in for UserRepo. This is the
// "fakeRepo pattern" seniors reach for instead of a mocking library:
// a small, real implementation backed by a map, with the same semantics
// as the production repo (duplicate emails fail, missing IDs fail, etc).
//
// It lives in a non-_test.go file here so main.go can also use it to run
// the demo without a DB — in a real project you'd move this into
// user_repo_test.go (or a testutil package) since only tests should
// depend on it.
type fakeUserRepo struct {
	byID    map[int64]*User
	byEmail map[string]*User
	nextID  int64
}

func NewFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byID:    make(map[int64]*User),
		byEmail: make(map[string]*User),
		nextID:  1,
	}
}

func (r *fakeUserRepo) FindByID(_ context.Context, id int64) (*User, error) {
	u, ok := r.byID[id]
	if !ok {
		return nil, fmt.Errorf("user %d not found", id)
	}
	return u, nil
}

func (r *fakeUserRepo) FindByEmail(_ context.Context, email string) (*User, error) {
	u, ok := r.byEmail[email]
	if !ok {
		return nil, fmt.Errorf("user with email %s not found", email)
	}
	return u, nil
}

func (r *fakeUserRepo) Create(_ context.Context, u *User) error {
	if _, exists := r.byEmail[u.Email]; exists {
		return fmt.Errorf("email %s already registered", u.Email)
	}
	u.ID = r.nextID
	r.nextID++
	r.byID[u.ID] = u
	r.byEmail[u.Email] = u
	return nil
}
