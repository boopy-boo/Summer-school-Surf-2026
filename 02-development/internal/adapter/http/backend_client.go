package http

import (
	"context"
	"time"

	"pottery-api/internal/domain"
)

type BackendClient struct {
	baseURL string
	timeout time.Duration
}

func NewBackendClient(baseURL string, timeout time.Duration) *BackendClient {
	return &BackendClient{
		baseURL: baseURL,
		timeout: timeout,
	}
}

// ClientStore

func (c *BackendClient) GetByPhone(ctx context.Context, phone string) (*domain.Client, error) {
	return nil, domain.ErrNotFound
}

func (c *BackendClient) Create(ctx context.Context, name, phone string) (*domain.Client, error) {
	return &domain.Client{ID: "mock", Name: name, Phone: phone}, nil
}

func (c *BackendClient) GetByID(ctx context.Context, id string) (*domain.Client, error) {
	return &domain.Client{ID: id, Name: "Mock", Phone: "+70000000000"}, nil
}

func (c *BackendClient) Update(ctx context.Context, id, name string) error {
	return nil
}

func (c *BackendClient) Delete(ctx context.Context, id string) error {
	return nil
}

// SlotReader

func (c *BackendClient) List(ctx context.Context, filter domain.SlotFilter) ([]domain.SlotListItem, int, error) {
	return []domain.SlotListItem{}, 0, nil
}

func (c *BackendClient) GetSlotByID(ctx context.Context, id string) (*domain.Slot, error) {
	return nil, domain.ErrNotFound
}

func (c *BackendClient) ListMasters(ctx context.Context) ([]domain.Master, error) {
	return []domain.Master{}, nil
}

// BookingStore

func (c *BackendClient) ListBookings(ctx context.Context, clientID string, limit, offset int) ([]domain.Booking, int, error) {
	return []domain.Booking{}, 0, nil
}

func (c *BackendClient) GetBookingByID(ctx context.Context, clientID, id string) (*domain.Booking, error) {
	return nil, domain.ErrNotFound
}

func (c *BackendClient) CreateBooking(ctx context.Context, clientID string, req domain.BookingCreateRequest) (*domain.Booking, error) {
	return nil, domain.ErrConflict
}

func (c *BackendClient) CancelBooking(ctx context.Context, clientID, id string) error {
	return nil
}