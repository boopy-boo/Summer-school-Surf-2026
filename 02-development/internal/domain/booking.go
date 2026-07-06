package domain

import "time"

type BookingStatus string

const (
	BookingActive            BookingStatus = "active"
	BookingCancelled         BookingStatus = "cancelled"
	BookingLateCancel        BookingStatus = "late_cancel"
	BookingWorkshopCancelled BookingStatus = "workshop_cancelled"
)

type Booking struct {
	ID                 string        `json:"id"`
	SlotID             string        `json:"slot_id"`
	ClientID           string        `json:"client_id"`
	SeatsCount         int           `json:"seats_count"`
	RentalCount        int           `json:"rental_count"`
	PriceTotal         float64       `json:"price_total"`
	Status             BookingStatus `json:"status"`
	CancellationReason string        `json:"cancellation_reason,omitempty"`
	CreatedAt          time.Time     `json:"created_at"`
	CancelledAt        *time.Time    `json:"cancelled_at,omitempty"`
}

type SeatRequest struct {
	Rental bool `json:"rental"`
}

type BookingCreateRequest struct {
	SlotID string        `json:"slot_id" validate:"required,uuid"`
	Seats  []SeatRequest `json:"seats" validate:"required,min=1,max=3,dive"`
}

type BookingListResponse struct {
	Items []Booking `json:"items"`
	Total int       `json:"total"`
}

// MaxSeats returns the maximum number of seats allowed for a booking.
// Formula: min(freeSeats, capacityCap, 3) — FR-13
func MaxSeats(freeSeats, capacityCap int) int {
	const bookingLimit = 3
	max := freeSeats
	if capacityCap < max {
		max = capacityCap
	}
	if bookingLimit < max {
		max = bookingLimit
	}
	return max
}

// CanCancel checks if a booking can be cancelled without late penalty. FR-17
func (b Booking) CanCancel(now, slotStart time.Time) bool {
	if b.Status != BookingActive {
		return false
	}
	return now.Before(slotStart.Add(-2 * time.Hour))
}

// CountRental counts rental instruments in seat requests.
func CountRental(seats []SeatRequest) int {
	count := 0
	for _, s := range seats {
		if s.Rental {
			count++
		}
	}
	return count
}