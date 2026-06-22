package main

import (
	"context"
	"fmt"
)

// UserService holds business logic. It depends on a UserRepo interface, not a concrete type.
// In Django you'd import models.User directly and call User.objects.get(pk=id).
// Here the service is decoupled — it doesn't know whether the repo hits Postgres or a map.
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
