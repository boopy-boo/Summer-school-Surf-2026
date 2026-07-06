package service

import (
	"context"

	"pottery-api/internal/domain"
	"pottery-api/internal/repository"
)

type BookingService struct {
	bookingRepo repository.BookingStore
	slotReader  repository.SlotReader
}

func NewBookingService(repo repository.BookingStore, slots repository.SlotReader) *BookingService {
	return &BookingService{
		bookingRepo: repo,
		slotReader:  slots,
	}
}

func (s *BookingService) Create(ctx context.Context, clientID string, req domain.BookingCreateRequest) (*domain.Booking, error) {
	slot, err := s.slotReader.GetSlotByID(ctx, req.SlotID)
	if err != nil {
		return nil, err
	}

	if slot.Status == domain.SlotCancelled {
		return nil, domain.ErrGone
	}

	maxSeats := domain.MaxSeats(slot.FreeSeats, slot.Program.CapacityCap)
	if len(req.Seats) > maxSeats {
		return nil, domain.ErrConflict
	}

	rentalCount := domain.CountRental(req.Seats)
	if rentalCount > slot.FreeRentalKits {
		return nil, domain.ErrConflict
	}

	return s.bookingRepo.CreateBooking(ctx, clientID, req)
}

func (s *BookingService) List(ctx context.Context, clientID string, limit, offset int) ([]domain.Booking, int, error) {
	return s.bookingRepo.ListBookings(ctx, clientID, limit, offset)
}

func (s *BookingService) GetByID(ctx context.Context, clientID, id string) (*domain.Booking, error) {
	return s.bookingRepo.GetBookingByID(ctx, clientID, id)
}

func (s *BookingService) Cancel(ctx context.Context, clientID, bookingID string) error {
	booking, err := s.bookingRepo.GetBookingByID(ctx, clientID, bookingID)
	if err != nil {
		return err
	}
	if booking.Status != domain.BookingActive {
		return domain.ErrBadRequest
	}
	return s.bookingRepo.CancelBooking(ctx, clientID, bookingID)
}