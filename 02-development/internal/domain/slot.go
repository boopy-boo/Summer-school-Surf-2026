package domain

import "time"

type SlotStatus string

const (
	SlotActive    SlotStatus = "active"
	SlotFilled    SlotStatus = "filled"
	SlotCancelled SlotStatus = "cancelled"
)

type ProgramType string

const (
	ProgramHandbuilding ProgramType = "handbuilding"
	ProgramWheel        ProgramType = "wheel"
)

type Program struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	Type           ProgramType `json:"type"`
	CapacityCap    int         `json:"capacity_cap"`
	DurationMinutes int        `json:"duration_minutes"`
}

type ProgramBrief struct {
	ID   string      `json:"id"`
	Name string      `json:"name"`
	Type ProgramType `json:"type"`
}

type Master struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Coordinates struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Slot struct {
	ID                string       `json:"id"`
	Program           Program      `json:"program"`
	Master            Master       `json:"master"`
	StartAt           time.Time    `json:"start_at"`
	TotalSeats        int          `json:"total_seats"`
	FreeSeats         int          `json:"free_seats"`
	FreeRentalKits    int          `json:"free_rental_kits"`
	Price             float64      `json:"price"`
	RentalPrice       float64      `json:"rental_price"`
	WorkshopAddress   string       `json:"workshop_address"`
	WorkshopCoordinates *Coordinates `json:"workshop_coordinates,omitempty"`
	Status            SlotStatus   `json:"status"`
}

type SlotListItem struct {
	ID         string       `json:"id"`
	StartAt    time.Time    `json:"start_at"`
	Program    ProgramBrief `json:"program"`
	Master     Master       `json:"master"`
	TotalSeats int          `json:"total_seats"`
	FreeSeats  int          `json:"free_seats"`
	Price      float64      `json:"price"`
	Status     SlotStatus   `json:"status"`
}

type SlotFilter struct {
	DateFrom      *time.Time
	DateTo        *time.Time
	ProgramTypes  []string
	MasterIDs     []string
	OnlyAvailable bool
	Limit         int
	Offset        int
}

type SlotListResponse struct {
	Items []SlotListItem `json:"items"`
	Total int            `json:"total"`
}