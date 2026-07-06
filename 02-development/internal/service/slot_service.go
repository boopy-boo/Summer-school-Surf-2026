package service

import (
	"context"

	"pottery-api/internal/domain"
	"pottery-api/internal/repository"
)

type SlotService struct {
	reader repository.SlotReader
}

func NewSlotService(reader repository.SlotReader) *SlotService {
	return &SlotService{reader: reader}
}

func (s *SlotService) ListSlots(ctx context.Context, filter domain.SlotFilter) ([]domain.SlotListItem, int, error) {
	if filter.Limit == 0 {
		filter.Limit = 20
	}
	if filter.DateFrom == nil && filter.DateTo == nil {
		// Default: now to now+7 days handled by backend or caller
	}
	return s.reader.ListSlots(ctx, filter)
}

func (s *SlotService) GetSlot(ctx context.Context, id string) (*domain.Slot, error) {
	return s.reader.GetSlotByID(ctx, id)
}

func (s *SlotService) ListMasters(ctx context.Context) ([]domain.Master, error) {
	return s.reader.ListMasters(ctx)
}