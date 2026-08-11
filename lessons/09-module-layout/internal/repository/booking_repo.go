package repository

import (
	"context"
	"fmt"
	"sync"

	"lesson09/module-layout/internal/domain"
)

// BookingRepository is defined here, next to the implementation — service
// will import this interface, never the other way around.
type BookingRepository interface {
	Create(ctx context.Context, title string) (domain.Booking, error)
	Get(ctx context.Context, id string) (domain.Booking, error)
}

type memoryBookingRepo struct {
	mu   sync.Mutex
	seq  int
	byID map[string]domain.Booking
}

func NewMemoryBookingRepo() BookingRepository {
	return &memoryBookingRepo{byID: make(map[string]domain.Booking)}
}

func (r *memoryBookingRepo) Create(_ context.Context, title string) (domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	b := domain.Booking{ID: fmt.Sprintf("bk_%d", r.seq), Title: title}
	r.byID[b.ID] = b
	return b, nil
}

func (r *memoryBookingRepo) Get(_ context.Context, id string) (domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.byID[id]
	if !ok {
		return domain.Booking{}, fmt.Errorf("booking %q not found", id)
	}
	return b, nil
}
