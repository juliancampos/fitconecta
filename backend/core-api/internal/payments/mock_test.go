package payments

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

	"github.com/google/uuid"
)

func TestMockProviderPixAndWebhook(t *testing.T) {
	provider := NewMockProvider("segredo")
	charge, err := provider.CreatePix(context.Background(), ChargeInput{
		PaymentID:     uuid.New(),
		AppointmentID: uuid.New(),
		AmountCents:   9900,
		ExpiresAt:     time.Now().Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("erro ao criar cobranca: %v", err)
	}
	if charge.ProviderPaymentID == "" || charge.QRCodeCopyPaste == "" || charge.QRCodeImage == "" {
		t.Fatal("cobranca mock incompleta")
	}

	body, _ := json.Marshal(map[string]any{
		"event_id":            "evt_1",
		"type":                "payment.updated",
		"provider_payment_id": charge.ProviderPaymentID,
		"status":              StatusPaid,
		"amount_cents":        charge.AmountCents,
	})
	mac := hmac.New(sha256.New, []byte("segredo"))
	mac.Write(body)
	req := httptest.NewRequest("POST", "/payments/webhook/mock", bytes.NewReader(body))
	req.Header.Set("X-Mock-Signature", hex.EncodeToString(mac.Sum(nil)))

	event, err := provider.ParseWebhook(req)
	if err != nil {
		t.Fatalf("webhook deveria ser valido: %v", err)
	}
	if event.ProviderEventID != "evt_1" || event.Status != StatusPaid {
		t.Fatalf("evento inesperado: %+v", event)
	}

	updated, err := provider.GetPix(context.Background(), charge.ProviderPaymentID)
	if err != nil {
		t.Fatalf("cobranca deveria existir: %v", err)
	}
	if updated.Status != StatusPaid {
		t.Fatalf("status esperado paid, obteve %s", updated.Status)
	}
}

func TestMockProviderRejectsInvalidSignature(t *testing.T) {
	provider := NewMockProvider("segredo")
	req := httptest.NewRequest("POST", "/payments/webhook/mock", bytes.NewReader([]byte(`{"event_id":"x"}`)))
	req.Header.Set("X-Mock-Signature", "invalida")
	if _, err := provider.ParseWebhook(req); err == nil {
		t.Fatal("esperava erro de assinatura")
	}
}
