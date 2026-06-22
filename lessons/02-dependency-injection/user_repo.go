package main

import (
	"context"
	"fmt"
)

type User struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// UserRepo is the interface your service layer depends on.
// This is the key DI move: depend on an interface, not a concrete type.
// It lets you swap in a fakeUserRepo in tests without touching handler or service code.
type UserRepo interface {
	FindByID(ctx context.Context, id int64) (*User, error)
}

// pgxUserRepo is the real implementation — used in production with a live DB.
// In a real Coordeck setup, the field would be *pgxpool.Pool.
// We use a marker string here so this lesson compiles without a DB connection.
type pgxUserRepo struct {
	dsn string // swap for *pgxpool.Pool in production
}

func NewUserRepo(dsn string) UserRepo {
	return &pgxUserRepo{dsn: dsn}
}

func (r *pgxUserRepo) FindByID(_ context.Context, id int64) (*User, error) {
	// Real implementation:
	//   var u User
	//   err := r.db.QueryRow(ctx,
	//       `SELECT id, email, name FROM users WHERE id = $1`, id,
	//   ).Scan(&u.ID, &u.Email, &u.Name)
	return nil, fmt.Errorf("pgxUserRepo: no real DB in this lesson (id=%d)", id)
}

// fakeUserRepo is an in-memory repo used in main() so the lesson runs without a DB.
// This is exactly the pattern seniors use in unit tests.
type fakeUserRepo struct {
	data map[int64]*User
}

func NewFakeUserRepo() UserRepo {
	return &fakeUserRepo{
		data: map[int64]*User{
			1: {ID: 1, Email: "amir@coordeck.com", Name: "Amir"},
			2: {ID: 2, Email: "alice@example.com", Name: "Alice"},
		},
	}
}

func (r *fakeUserRepo) FindByID(_ context.Context, id int64) (*User, error) {
	u, ok := r.data[id]
	if !ok {
		return nil, fmt.Errorf("user %d not found", id)
	}
	return u, nil
}
