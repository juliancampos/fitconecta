package appointments

import (
	"time"

	"github.com/google/uuid"
)

type AppointmentStatus string

const (
	StatusPendingPayment          AppointmentStatus = "pending_payment"
	StatusConfirmed               AppointmentStatus = "confirmed"
	StatusCancelledByClient       AppointmentStatus = "cancelled_by_client"
	StatusCancelledByProfessional AppointmentStatus = "cancelled_by_professional"
	StatusCancelledByAdmin        AppointmentStatus = "cancelled_by_admin"
	StatusCompleted               AppointmentStatus = "completed"
	StatusNoShow                  AppointmentStatus = "no_show"
	StatusExpired                 AppointmentStatus = "expired"
)

type PaymentMethod string

const (
	MethodPix        PaymentMethod = "pix"
	MethodCreditCard PaymentMethod = "credit_card"
)

type LocationType string

const (
	LocationHome   LocationType = "home"
	LocationStudio LocationType = "studio"
	LocationOnline LocationType = "online"
)

func (l LocationType) Valid() bool {
	return l == LocationHome || l == LocationStudio || l == LocationOnline
}

type PaymentStatus string

const (
	PaymentPending       PaymentStatus = "pending"
	PaymentPaid          PaymentStatus = "paid"
	PaymentFailed        PaymentStatus = "failed"
	PaymentRefundPending PaymentStatus = "refund_pending"
	PaymentRefunded      PaymentStatus = "refunded"
	PaymentExpired       PaymentStatus = "expired"
)

type Appointment struct {
	ID             uuid.UUID         `json:"id"`
	ProfessionalID uuid.UUID         `json:"professional_id"`
	ClientID       uuid.UUID         `json:"client_id"`
	StartsAt       time.Time         `json:"starts_at"`
	EndsAt         time.Time         `json:"ends_at"`
	SpecialtyID    *int              `json:"specialty_id,omitempty"`
	LocationType   *LocationType     `json:"location_type,omitempty"`
	LocationNotes  string            `json:"location_notes,omitempty"`
	Status         AppointmentStatus `json:"status"`
	ReservedUntil  *time.Time        `json:"reserved_until,omitempty"`
	CancelledAt    *time.Time        `json:"cancelled_at,omitempty"`
	CancelReason   *string           `json:"cancel_reason,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`

	// Dados agregados para visualizacao
	ProfessionalName string   `json:"professional_name,omitempty"`
	ClientName       string   `json:"client_name,omitempty"`
	SpecialtyName    string   `json:"specialty_name,omitempty"`
	PaymentInfo      *Payment `json:"payment,omitempty"`
}

type Payment struct {
	ID                   uuid.UUID     `json:"id"`
	AppointmentID        uuid.UUID     `json:"appointment_id"`
	ClientID             uuid.UUID     `json:"client_id"`
	ProfessionalID       uuid.UUID     `json:"professional_id"`
	AmountCents          int           `json:"amount_cents"`
	PlatformFeeCents     int           `json:"platform_fee_cents"`
	ProfessionalNetCents int           `json:"professional_net_cents"`
	Method               PaymentMethod `json:"method"`
	Status               PaymentStatus `json:"status"`
	GatewayTransactionID *string       `json:"gateway_transaction_id,omitempty"`
	Provider             string        `json:"provider,omitempty"`
	ProviderPaymentID    *string       `json:"provider_payment_id,omitempty"`
	ProviderFeeCents     *int          `json:"provider_fee_cents,omitempty"`
	ExpiresAt            *time.Time    `json:"expires_at,omitempty"`
	RawStatus            string        `json:"raw_status,omitempty"`
	PaidAt               *time.Time    `json:"paid_at,omitempty"`
	CreatedAt            time.Time     `json:"created_at"`
}

type PixChargeResponse struct {
	PaymentID         uuid.UUID     `json:"payment_id"`
	AppointmentID     uuid.UUID     `json:"appointment_id"`
	Method            PaymentMethod `json:"method"`
	Status            PaymentStatus `json:"status"`
	AmountCents       int           `json:"amount_cents"`
	ExpiresAt         time.Time     `json:"expires_at"`
	QRCodeImage       string        `json:"qr_code_image,omitempty"`
	CopyPaste         string        `json:"copy_paste,omitempty"`
	ProviderPaymentID string        `json:"provider_payment_id,omitempty"`
}

type CreateAppointmentRequest struct {
	ProfessionalID uuid.UUID     `json:"professional_id"`
	StartsAt       time.Time     `json:"starts_at"`
	EndsAt         time.Time     `json:"ends_at"`
	SpecialtyID    *int          `json:"specialty_id,omitempty"`   // tipo de aula (especialidade do profissional)
	LocationType   *LocationType `json:"location_type,omitempty"`  // tipo de local: home | studio | online
	LocationNotes  string        `json:"location_notes,omitempty"` // descricao livre
	PaymentMethod  PaymentMethod `json:"payment_method"`           // "pix" ou "credit_card"
}

type CancelAppointmentRequest struct {
	Reason string `json:"reason,omitempty"`
}
