package service

import (
	"context"
	"fmt"

	"lesson09/module-layout/internal/domain"
	"lesson09/module-layout/internal/repository"
)

type BookingService interface {
	CreateBooking(ctx context.Context, title string) (domain.Booking, error)
	GetBooking(ctx context.Context, id string) (domain.Booking, error)
}

type bookingService struct {
	repo repository.BookingRepository
}

func NewBookingService(repo repository.BookingRepository) BookingService {
	return &bookingService{repo: repo}
}

func (s *bookingService) CreateBooking(ctx context.Context, title string) (domain.Booking, error) {
	if title == "" {
		return domain.Booking{}, fmt.Errorf("title is required")
	}
	return s.repo.Create(ctx, title)
}

func (s *bookingService) GetBooking(ctx context.Context, id string) (domain.Booking, error) {
	return s.repo.Get(ctx, id)
}
