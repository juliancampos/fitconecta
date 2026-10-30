package appointments

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"fitconecta/core-api/internal/payments"
	"fitconecta/core-api/internal/platform/middleware"
	"github.com/google/uuid"
)

type pixRepo struct {
	mockRepo
	app            *Appointment
	clientID       uuid.UUID
	markedPaid     bool
	processedHook  bool
	registerShould bool
}

func (r *pixRepo) GetByID(ctx context.Context, id uuid.UUID) (*Appointment, error) {
	return r.app, nil
}

func (r *pixRepo) GetClientProfile(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	return r.clientID, nil
}

func (r *pixRepo) CreatePixCharge(ctx context.Context, appointmentID uuid.UUID, provider, providerPaymentID, copyPaste, qrImage string, expiresAt time.Time) (*Payment, error) {
	pay := r.app.PaymentInfo
	pay.Provider = provider
	pay.ProviderPaymentID = &providerPaymentID
	pay.ExpiresAt = &expiresAt
	pay.RawStatus = "pending"
	return pay, nil
}

func (r *pixRepo) MarkPaymentPaid(ctx context.Context, provider, providerPaymentID, rawStatus string, providerFeeCents *int) error {
	r.markedPaid = true
	return nil
}

func (r *pixRepo) RegisterWebhookEvent(ctx context.Context, provider, eventID, eventType string, payload []byte) (uuid.UUID, bool, error) {
	return uuid.New(), r.registerShould, nil
}

func (r *pixRepo) MarkWebhookProcessed(ctx context.Context, eventID uuid.UUID) error {
	r.processedHook = true
	return nil
}

func newPixTestRepo() (*pixRepo, uuid.UUID) {
	appID := uuid.New()
	clientID := uuid.New()
	reserved := time.Now().Add(10 * time.Minute)
	app := &Appointment{
		ID:            appID,
		ClientID:      clientID,
		Status:        StatusPendingPayment,
		ReservedUntil: &reserved,
		PaymentInfo: &Payment{
			ID:            uuid.New(),
			AppointmentID: appID,
			AmountCents:   15000,
			Method:        MethodPix,
			Status:        PaymentPending,
		},
	}
	return &pixRepo{app: app, clientID: clientID, registerShould: true}, clientID
}

func TestCreatePixCharge(t *testing.T) {
	repo, clientID := newPixTestRepo()
	provider := payments.NewMockProvider("segredo")
	svc := NewServiceWithPayments(repo, 10, provider, 10*time.Minute, 2*time.Minute)

	user := middleware.AuthUser{ID: uuid.New(), Role: "client"}
	charge, err := svc.CreatePixCharge(context.Background(), user, repo.app.ID)
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if charge.AppointmentID != repo.app.ID || charge.AmountCents != 15000 {
		t.Fatalf("cobranca inesperada: %+v", charge)
	}
	if charge.CopyPaste == "" || charge.QRCodeImage == "" || charge.ProviderPaymentID == "" {
		t.Fatalf("cobranca Pix incompleta: %+v", charge)
	}
	if repo.app.PaymentInfo.Provider != "mock" {
		t.Fatalf("provider esperado mock, obteve %q", repo.app.PaymentInfo.Provider)
	}

	professional := middleware.AuthUser{ID: clientID, Role: "professional"}
	if _, err := svc.CreatePixCharge(context.Background(), professional, repo.app.ID); err == nil {
		t.Fatal("profissional nao deveria gerar cobranca")
	}
}

func TestHandleWebhookConfirmsPayment(t *testing.T) {
	repo, _ := newPixTestRepo()
	provider := payments.NewMockProvider("segredo")
	svc := NewServiceWithPayments(repo, 10, provider, 10*time.Minute, 2*time.Minute)

	created, err := provider.CreatePix(context.Background(), payments.ChargeInput{
		PaymentID:     repo.app.PaymentInfo.ID,
		AppointmentID: repo.app.ID,
		AmountCents:   repo.app.PaymentInfo.AmountCents,
		ExpiresAt:     repo.app.ReservedUntil.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("erro ao preparar cobranca: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"event_id":            "evt_service_1",
		"type":                "payment.updated",
		"provider_payment_id": created.ProviderPaymentID,
		"status":              payments.StatusPaid,
		"amount_cents":        created.AmountCents,
	})
	mac := hmac.New(sha256.New, []byte("segredo"))
	mac.Write(body)
	req := httptest.NewRequest("POST", "/payments/webhook/mock", bytes.NewReader(body))
	req.Header.Set("X-Mock-Signature", hex.EncodeToString(mac.Sum(nil)))

	event, err := provider.ParseWebhook(req)
	if err != nil {
		t.Fatalf("webhook invalido: %v", err)
	}
	if err := svc.HandleWebhook(context.Background(), event); err != nil {
		t.Fatalf("esperava processamento do webhook, obteve: %v", err)
	}
	if !repo.markedPaid {
		t.Fatal("pagamento deveria ter sido marcado como pago")
	}
	if !repo.processedHook {
		t.Fatal("evento deveria ter sido marcado como processado")
	}
}

func TestHandleWebhookIgnoresDuplicate(t *testing.T) {
	repo, _ := newPixTestRepo()
	repo.registerShould = false
	provider := payments.NewMockProvider("segredo")
	svc := NewServiceWithPayments(repo, 10, provider, 10*time.Minute, 2*time.Minute)

	event := &payments.WebhookEvent{
		ProviderEventID:   "evt_duplicado",
		Type:              "payment.updated",
		ProviderPaymentID: "mock_x",
		Status:            payments.StatusPaid,
		Payload:           []byte("{}"),
	}
	if err := svc.HandleWebhook(context.Background(), event); err != nil {
		t.Fatalf("evento duplicado deveria ser ignorado sem erro: %v", err)
	}
	if repo.markedPaid {
		t.Fatal("evento duplicado nao deveria alterar o pagamento")
	}
}
