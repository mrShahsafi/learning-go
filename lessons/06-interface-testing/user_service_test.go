package main

import (
	"context"
	"testing"
)

// This is the Go equivalent of a Django TestCase method using a fixture
// or an in-memory-friendly ORM state. No mocking library, no monkeypatch —
// just a real (if fake) implementation of the interface.

func TestUserService_GetUser(t *testing.T) {
	repo := NewFakeUserRepo()
	ctx := context.Background()
	seeded, err := (&UserService{repo: repo}).CreateUser(ctx, "amir@coordeck.com", "Amir")
	if err != nil {
		t.Fatalf("seed CreateUser failed: %v", err)
	}
	svc := NewUserService(repo)

	// Table-driven test: one struct per case, one loop, one t.Run per row.
	// This is the idiomatic replacement for pytest.mark.parametrize.
	tests := []struct {
		name    string
		id      int64
		wantErr bool
	}{
		{name: "existing user", id: seeded.ID, wantErr: false},
		{name: "zero id is invalid", id: 0, wantErr: true},
		{name: "negative id is invalid", id: -5, wantErr: true},
		{name: "unknown id not found", id: 999, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.GetUser(ctx, tt.id)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("GetUser(%d) = nil error, want error", tt.id)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetUser(%d) unexpected error: %v", tt.id, err)
			}
			if got.Email != seeded.Email {
				t.Errorf("GetUser(%d).Email = %q, want %q", tt.id, got.Email, seeded.Email)
			}
		})
	}
}

func TestUserService_CreateUser(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		userN   string
		wantErr bool
	}{
		{name: "valid input", email: "alice@example.com", userN: "Alice", wantErr: false},
		{name: "empty email", email: "", userN: "Alice", wantErr: true},
		{name: "email missing @", email: "not-an-email", userN: "Alice", wantErr: true},
		{name: "empty name", email: "bob@example.com", userN: "", wantErr: true},
		{name: "whitespace-only name", email: "carol@example.com", userN: "   ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fresh fakeUserRepo per subtest — no shared state leaking between
			// cases. This is the Go analogue of Django's per-test transaction
			// rollback, done by construction instead of framework magic.
			svc := NewUserService(NewFakeUserRepo())

			got, err := svc.CreateUser(context.Background(), tt.email, tt.userN)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("CreateUser(%q, %q) = nil error, want error", tt.email, tt.userN)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateUser(%q, %q) unexpected error: %v", tt.email, tt.userN, err)
			}
			if got.ID == 0 {
				t.Errorf("CreateUser did not assign an ID")
			}
		})
	}
}

func TestUserService_CreateUser_DuplicateEmail(t *testing.T) {
	svc := NewUserService(NewFakeUserRepo())
	ctx := context.Background()

	if _, err := svc.CreateUser(ctx, "dupe@example.com", "First"); err != nil {
		t.Fatalf("first CreateUser failed: %v", err)
	}

	_, err := svc.CreateUser(ctx, "dupe@example.com", "Second")
	if err == nil {
		t.Fatal("expected duplicate email to fail, got nil error")
	}
}
