package availability

import (
	"time"

	"github.com/google/uuid"
)

type AvailabilityRule struct {
	ID             uuid.UUID `json:"id"`
	ProfessionalID uuid.UUID `json:"professional_id"`
	Weekday        int       `json:"weekday"` // 0=Domingo, 6=Sabado
	StartTime      string    `json:"start_time"` // formato "HH:MM:SS"
	EndTime        string    `json:"end_time"`   // formato "HH:MM:SS"
	SlotMinutes    int       `json:"slot_minutes"`
	ValidFrom      *string   `json:"valid_from,omitempty"`
	ValidUntil     *string   `json:"valid_until,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateRuleRequest struct {
	Weekday     int    `json:"weekday"`
	StartTime   string `json:"start_time"` // "08:00"
	EndTime     string `json:"end_time"`   // "18:00"
	SlotMinutes int    `json:"slot_minutes"` // ex: 60
}

type AvailabilityException struct {
	ID             uuid.UUID `json:"id"`
	ProfessionalID uuid.UUID `json:"professional_id"`
	StartsAt       time.Time `json:"starts_at"`
	EndsAt         time.Time `json:"ends_at"`
	Reason         *string   `json:"reason,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateExceptionRequest struct {
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
	Reason   string    `json:"reason"`
}

type Slot struct {
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	Available bool      `json:"available"`
}
