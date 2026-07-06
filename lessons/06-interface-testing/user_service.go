package main

import (
	"context"
	"fmt"
	"strings"
)

// UserService holds business logic. It depends on the UserRepo interface,
// so tests can inject fakeUserRepo and never touch a real database.
type UserService struct {
	repo UserRepo
}

func NewUserService(repo UserRepo) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetUser(ctx context.Context, id int64) (*User, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid user id: %d", id)
	}
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("GetUser: %w", err)
	}
	return u, nil
}

// CreateUser has enough branching logic to be worth table-driven testing:
// empty fields, malformed email, and duplicate email all fail differently.
func (s *UserService) CreateUser(ctx context.Context, email, name string) (*User, error) {
	email = strings.TrimSpace(email)
	name = strings.TrimSpace(name)

	if email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if !strings.Contains(email, "@") {
		return nil, fmt.Errorf("invalid email: %s", email)
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	u := &User{Email: email, Name: name}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("CreateUser: %w", err)
	}
	return u, nil
}
