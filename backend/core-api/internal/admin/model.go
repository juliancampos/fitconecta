package admin

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound          = errors.New("registro nao encontrado")
	ErrInvalidStatus     = errors.New("status invalido para esta operacao")
	ErrInvalidTarget     = errors.New("registro nao pode ser alterado pelo painel")
	ErrReasonRequired    = errors.New("informe o motivo da alteracao")
	ErrInvalidAdminLogin = errors.New("credenciais administrativas invalidas")
	ErrInvalidFilter     = errors.New("filtro invalido")
)

type Party struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

type AdminAppointment struct {
	ID            uuid.UUID  `json:"id"`
	Client        Party      `json:"client"`
	Professional  Party      `json:"professional"`
	StartsAt      time.Time  `json:"starts_at"`
	EndsAt        time.Time  `json:"ends_at"`
	Status        string     `json:"status"`
	SpecialtyName string     `json:"specialty_name,omitempty"`
	LocationType  string     `json:"location_type,omitempty"`
	LocationNotes string     `json:"location_notes,omitempty"`
	PaymentID     *uuid.UUID `json:"payment_id,omitempty"`
	AmountCents   *int       `json:"amount_cents,omitempty"`
	PaymentStatus string     `json:"payment_status,omitempty"`
	CancelledAt   *time.Time `json:"cancelled_at,omitempty"`
	CancelReason  *string    `json:"cancel_reason,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type AdminPayment struct {
	ID                   uuid.UUID  `json:"id"`
	AppointmentID        uuid.UUID  `json:"appointment_id"`
	Client               Party      `json:"client"`
	Professional         Party      `json:"professional"`
	AmountCents          int        `json:"amount_cents"`
	PlatformFeeCents     int        `json:"platform_fee_cents"`
	ProfessionalNetCents int        `json:"professional_net_cents"`
	Method               string     `json:"method"`
	Status               string     `json:"status"`
	GatewayTransactionID *string    `json:"gateway_transaction_id,omitempty"`
	PaidAt               *time.Time `json:"paid_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
}

type AdminUser struct {
	ID                 uuid.UUID  `json:"id"`
	Email              string     `json:"email"`
	Role               string     `json:"role"`
	Status             string     `json:"status"`
	StatusReason       *string    `json:"status_reason,omitempty"`
	StatusChangedAt    *time.Time `json:"status_changed_at,omitempty"`
	DisplayName        string     `json:"display_name"`
	Phone              string     `json:"phone,omitempty"`
	City               string     `json:"city,omitempty"`
	State              string     `json:"state,omitempty"`
	ProfessionalID     *uuid.UUID `json:"professional_id,omitempty"`
	ClientID           *uuid.UUID `json:"client_id,omitempty"`
	CREFNumber         *string    `json:"cref_number,omitempty"`
	CREFVerifiedAt     *time.Time `json:"cref_verified_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	FutureAppointments int64      `json:"future_appointments"`
}

// CREFCheck é o resultado de uma nova consulta ao portal do CREF6.
// Reason: ok | no_cref | not_found | inactive | unavailable.
type CREFCheck struct {
	Verified bool   `json:"verified"`
	Reason   string `json:"reason"`
	Message  string `json:"message"`
}

type PaymentSummary struct {
	Count                int64            `json:"count"`
	AmountCents          int64            `json:"amount_cents"`
	PlatformFeeCents     int64            `json:"platform_fee_cents"`
	ProfessionalNetCents int64            `json:"professional_net_cents"`
	ByStatus             map[string]int64 `json:"by_status"`
}

type Page[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

type UpdateUserStatusResponse struct {
	User               AdminUser `json:"user"`
	FutureAppointments int64     `json:"future_appointments"`
}

type AppointmentFilters struct {
	ClientID          *uuid.UUID
	ProfessionalID    *uuid.UUID
	ClientQuery       string
	ProfessionalQuery string
	Query             string
	Status            string
	DateFrom          *time.Time
	DateTo            *time.Time
	Sort              string
	Page              int
	PageSize          int
}

type PaymentFilters struct {
	ClientID          *uuid.UUID
	ProfessionalID    *uuid.UUID
	ClientQuery       string
	ProfessionalQuery string
	Query             string
	Status            string
	Method            string
	DateField         string
	DateFrom          *time.Time
	DateTo            *time.Time
	Page              int
	PageSize          int
}

type UserFilters struct {
	Query    string
	Role     string
	Status   string
	Page     int
	PageSize int
}
