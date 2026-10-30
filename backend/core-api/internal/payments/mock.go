package payments

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	qrcode "github.com/skip2/go-qrcode"
)

// MockProvider simula um PSP Pix para desenvolvimento local/testes. Ele nao
// liquida dinheiro real e so deve ser habilitado por PIX_PROVIDER=mock.
type MockProvider struct {
	mu      sync.Mutex
	secret  string
	now     func() time.Time
	charges map[string]*Charge
}

func NewMockProvider(webhookSecret string) *MockProvider {
	return &MockProvider{
		secret:  webhookSecret,
		now:     time.Now,
		charges: make(map[string]*Charge),
	}
}

func (m *MockProvider) Name() string { return "mock" }

func (m *MockProvider) CreatePix(_ context.Context, in ChargeInput) (*Charge, error) {
	providerID := "mock_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	copyPaste := buildPixPayload(in.AmountCents, providerID)
	image, err := qrcode.Encode(copyPaste, qrcode.Medium, 384)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar QR Code Pix: %w", err)
	}

	expiresAt := in.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = m.now().Add(10 * time.Minute)
	}

	charge := &Charge{
		ProviderPaymentID: providerID,
		Status:            StatusPending,
		AmountCents:       in.AmountCents,
		QRCodeCopyPaste:   copyPaste,
		QRCodeImage:       "data:image/png;base64," + base64.StdEncoding.EncodeToString(image),
		ExpiresAt:         expiresAt,
	}

	m.mu.Lock()
	m.charges[providerID] = charge
	m.mu.Unlock()

	return charge, nil
}

func (m *MockProvider) GetPix(_ context.Context, providerPaymentID string) (*Charge, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	charge, ok := m.charges[providerPaymentID]
	if !ok {
		return nil, ErrChargeNotFound
	}
	copy := *charge
	return &copy, nil
}

func (m *MockProvider) ParseWebhook(r *http.Request) (*WebhookEvent, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: corpo invalido", ErrInvalidWebhook)
	}

	signature := strings.TrimSpace(r.Header.Get("X-Mock-Signature"))
	if signature == "" || !m.validSignature(body, signature) {
		return nil, fmt.Errorf("%w: assinatura invalida", ErrInvalidWebhook)
	}

	var payload struct {
		EventID           string `json:"event_id"`
		Type              string `json:"type"`
		ProviderPaymentID string `json:"provider_payment_id"`
		Status            string `json:"status"`
		AmountCents       int    `json:"amount_cents"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("%w: payload invalido", ErrInvalidWebhook)
	}
	if payload.EventID == "" || payload.ProviderPaymentID == "" || payload.Status == "" {
		return nil, fmt.Errorf("%w: campos obrigatorios ausentes", ErrInvalidWebhook)
	}

	if payload.Status == StatusPaid {
		m.mu.Lock()
		if charge, ok := m.charges[payload.ProviderPaymentID]; ok {
			charge.Status = StatusPaid
		}
		m.mu.Unlock()
	}

	return &WebhookEvent{
		ProviderEventID:   payload.EventID,
		Type:              payload.Type,
		ProviderPaymentID: payload.ProviderPaymentID,
		Status:            payload.Status,
		AmountCents:       payload.AmountCents,
		Payload:           body,
	}, nil
}

func (m *MockProvider) Refund(_ context.Context, providerPaymentID string, _ int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	charge, ok := m.charges[providerPaymentID]
	if !ok {
		return ErrChargeNotFound
	}
	charge.Status = "refunded"
	return nil
}

func (m *MockProvider) validSignature(body []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(m.secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(strings.ToLower(signature)))
}
