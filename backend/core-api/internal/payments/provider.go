package payments

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
)

var (
	ErrProviderUnavailable = errors.New("provedor de pagamento indisponivel")
	ErrChargeNotFound      = errors.New("cobranca nao encontrada")
	ErrInvalidWebhook      = errors.New("webhook invalido")
)

const (
	StatusPending = "pending"
	StatusPaid    = "paid"
	StatusExpired = "expired"
	StatusFailed  = "failed"
)

type ChargeInput struct {
	PaymentID     uuid.UUID
	AppointmentID uuid.UUID
	AmountCents   int
	Description   string
	PayerEmail    string
	ExpiresAt     time.Time
}

type Charge struct {
	ProviderPaymentID string
	Status            string
	AmountCents       int
	QRCodeCopyPaste   string
	QRCodeImage       string
	ExpiresAt         time.Time
}

type WebhookEvent struct {
	ProviderEventID   string
	Type              string
	ProviderPaymentID string
	Status            string
	AmountCents       int
	Payload           []byte
}

type Provider interface {
	Name() string
	CreatePix(ctx context.Context, in ChargeInput) (*Charge, error)
	GetPix(ctx context.Context, providerPaymentID string) (*Charge, error)
	ParseWebhook(r *http.Request) (*WebhookEvent, error)
	Refund(ctx context.Context, providerPaymentID string, amountCents int) error
}
