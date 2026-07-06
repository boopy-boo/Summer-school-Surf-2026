package repository

import (
	"context"

	"pottery-api/internal/domain"
)

type ClientStore interface {
	GetByPhone(ctx context.Context, phone string) (*domain.Client, error)
	GetByID(ctx context.Context, id string) (*domain.Client, error)
	Create(ctx context.Context, name, phone string) (*domain.Client, error)
	Update(ctx context.Context, id, name string) error
	Delete(ctx context.Context, id string) error
}

type SlotReader interface {
	ListSlots(ctx context.Context, filter domain.SlotFilter) ([]domain.SlotListItem, int, error)
	GetSlotByID(ctx context.Context, id string) (*domain.Slot, error)
	ListMasters(ctx context.Context) ([]domain.Master, error)
}

type BookingStore interface {
	ListBookings(ctx context.Context, clientID string, limit, offset int) ([]domain.Booking, int, error)
	GetBookingByID(ctx context.Context, clientID, id string) (*domain.Booking, error)
	CreateBooking(ctx context.Context, clientID string, req domain.BookingCreateRequest) (*domain.Booking, error)
	CancelBooking(ctx context.Context, clientID, id string) error
}

type OTPStore interface {
	Set(ctx context.Context, phone, code string, ttl int) error
	Get(ctx context.Context, phone string) (string, error)
	IncrAttempts(ctx context.Context, phone string, ttl int) (int, error)
	IncrSendCount(ctx context.Context, phone string, window int) (int, error)
	IncrFailCount(ctx context.Context, phone string, window int) (int, error)
	Delete(ctx context.Context, phone string) error
}