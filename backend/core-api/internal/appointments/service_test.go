package appointments

import (
	"context"
	"errors"
	"testing"
	"time"

	"fitconecta/core-api/internal/platform/middleware"
	"github.com/google/uuid"
)

type mockRepo struct {
	createCalled     bool
	specialtyOffered bool
}

func (m *mockRepo) CreateAppointmentWithPayment(ctx context.Context, clientID uuid.UUID, clientUserID, profUserID uuid.UUID, amountCents, platformFeePercent int, req *CreateAppointmentRequest) (*Appointment, *Payment, error) {
	m.createCalled = true
	fee := (amountCents * platformFeePercent) / 100
	net := amountCents - fee

	return &Appointment{
			ID:             uuid.New(),
			ProfessionalID: req.ProfessionalID,
			ClientID:       clientID,
			StartsAt:       req.StartsAt,
			EndsAt:         req.EndsAt,
			Status:         StatusPendingPayment,
		}, &Payment{
			ID:                   uuid.New(),
			AmountCents:          amountCents,
			PlatformFeeCents:     fee,
			ProfessionalNetCents: net,
			Method:               req.PaymentMethod,
			Status:               PaymentPending,
		}, nil
}

func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (*Appointment, error) {
	return nil, nil
}
func (m *mockRepo) CreatePixCharge(ctx context.Context, appointmentID uuid.UUID, provider, providerPaymentID, copyPaste, qrImage string, expiresAt time.Time) (*Payment, error) {
	return nil, nil
}
func (m *mockRepo) MarkPaymentPaid(ctx context.Context, provider, providerPaymentID, rawStatus string, providerFeeCents *int) error {
	return nil
}
func (m *mockRepo) ExpirePendingPayments(ctx context.Context, grace time.Duration) (int64, error) {
	return 0, nil
}
func (m *mockRepo) RegisterWebhookEvent(ctx context.Context, provider, eventID, eventType string, payload []byte) (uuid.UUID, bool, error) {
	return uuid.Nil, false, nil
}
func (m *mockRepo) MarkWebhookProcessed(ctx context.Context, eventID uuid.UUID) error {
	return nil
}
func (m *mockRepo) Cancel(ctx context.Context, id uuid.UUID, cancelledByRole string, reason string) (*Appointment, error) {
	return nil, nil
}
func (m *mockRepo) ListByClient(ctx context.Context, clientID uuid.UUID) ([]Appointment, error) {
	return nil, nil
}
func (m *mockRepo) ListByProfessional(ctx context.Context, profID uuid.UUID) ([]Appointment, error) {
	return nil, nil
}
func (m *mockRepo) GetClientProfile(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockRepo) GetProfessionalProfile(ctx context.Context, profID uuid.UUID) (uuid.UUID, int, error) {
	return uuid.New(), 15000, nil // R$ 150,00
}

func (m *mockRepo) GetProfessionalStatus(ctx context.Context, profID uuid.UUID) (string, error) {
	return "active", nil
}

func (m *mockRepo) ProfessionalOffersSpecialty(ctx context.Context, profID uuid.UUID, specialtyID int) (bool, error) {
	return m.specialtyOffered, nil
}

func TestSplitCalculation(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, 10) // 10% de split

	futureStart := time.Now().Add(24 * time.Hour)
	futureEnd := futureStart.Add(1 * time.Hour)

	req := &CreateAppointmentRequest{
		ProfessionalID: uuid.New(),
		StartsAt:       futureStart,
		EndsAt:         futureEnd,
		PaymentMethod:  MethodPix,
	}

	clientUser := middleware.AuthUser{
		ID:    uuid.New(),
		Email: "client@test.com",
		Role:  "client",
	}

	app, pay, err := svc.Create(context.Background(), clientUser, req)
	if err != nil {
		t.Fatalf("esperava sucesso ao criar agendamento, obteve erro: %v", err)
	}

	if app.Status != StatusPendingPayment {
		t.Errorf("status esperado %s, obteve %s", StatusPendingPayment, app.Status)
	}

	// Valor total: 15000 centavos (R$ 150,00)
	// 10% da plataforma: 1500 centavos
	// 90% do profissional: 13500 centavos
	expectedFee := 1500
	expectedNet := 13500

	if pay.PlatformFeeCents != expectedFee {
		t.Errorf("taxa da plataforma esperada: %d, obteve: %d", expectedFee, pay.PlatformFeeCents)
	}

	if pay.ProfessionalNetCents != expectedNet {
		t.Errorf("valor liquido do profissional esperado: %d, obteve: %d", expectedNet, pay.ProfessionalNetCents)
	}

	if pay.PlatformFeeCents+pay.ProfessionalNetCents != pay.AmountCents {
		t.Errorf("soma do split (%d + %d) diverge do total (%d)",
			pay.PlatformFeeCents, pay.ProfessionalNetCents, pay.AmountCents)
	}
}

func TestAppointmentValidation(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, 10)

	clientUser := middleware.AuthUser{
		ID:    uuid.New(),
		Email: "client@test.com",
		Role:  "client",
	}

	// 1. Horario inicial posterior ao final
	t.Run("start after end", func(t *testing.T) {
		req := &CreateAppointmentRequest{
			ProfessionalID: uuid.New(),
			StartsAt:       time.Now().Add(2 * time.Hour),
			EndsAt:         time.Now().Add(1 * time.Hour),
			PaymentMethod:  MethodPix,
		}
		_, _, err := svc.Create(context.Background(), clientUser, req)
		if err == nil {
			t.Error("esperava erro quando starts_at > ends_at, mas obteve sucesso")
		}
	})

	// 2. Horario no passado
	t.Run("past time", func(t *testing.T) {
		req := &CreateAppointmentRequest{
			ProfessionalID: uuid.New(),
			StartsAt:       time.Now().Add(-2 * time.Hour),
			EndsAt:         time.Now().Add(-1 * time.Hour),
			PaymentMethod:  MethodPix,
		}
		_, _, err := svc.Create(context.Background(), clientUser, req)
		if err == nil {
			t.Error("esperava erro ao agendar no passado, mas obteve sucesso")
		}
	})

	// 3. Usuario nao e cliente
	t.Run("non client role", func(t *testing.T) {
		profUser := middleware.AuthUser{
			ID:    uuid.New(),
			Email: "prof@test.com",
			Role:  "professional",
		}
		req := &CreateAppointmentRequest{
			ProfessionalID: uuid.New(),
			StartsAt:       time.Now().Add(1 * time.Hour),
			EndsAt:         time.Now().Add(2 * time.Hour),
			PaymentMethod:  MethodPix,
		}
		_, _, err := svc.Create(context.Background(), profUser, req)
		if err == nil {
			t.Error("esperava erro quando papel nao e cliente, mas obteve sucesso")
		}
	})

	// 4. Tipo de aula nao oferecido pelo profissional
	t.Run("specialty not offered", func(t *testing.T) {
		repo := &mockRepo{specialtyOffered: false}
		svc := NewService(repo, 10)
		specialtyID := 99
		req := &CreateAppointmentRequest{
			ProfessionalID: uuid.New(),
			StartsAt:       time.Now().Add(1 * time.Hour),
			EndsAt:         time.Now().Add(2 * time.Hour),
			SpecialtyID:    &specialtyID,
			PaymentMethod:  MethodPix,
		}
		_, _, err := svc.Create(context.Background(), clientUser, req)
		if !errors.Is(err, ErrInvalidSpecialty) {
			t.Errorf("esperava ErrInvalidSpecialty, obteve: %v", err)
		}
	})

	// 5. Tipo de aula oferecido pelo profissional
	t.Run("specialty offered", func(t *testing.T) {
		repo := &mockRepo{specialtyOffered: true}
		svc := NewService(repo, 10)
		specialtyID := 1
		req := &CreateAppointmentRequest{
			ProfessionalID: uuid.New(),
			StartsAt:       time.Now().Add(1 * time.Hour),
			EndsAt:         time.Now().Add(2 * time.Hour),
			SpecialtyID:    &specialtyID,
			PaymentMethod:  MethodPix,
		}
		_, _, err := svc.Create(context.Background(), clientUser, req)
		if err != nil {
			t.Errorf("esperava sucesso com especialidade valida, obteve: %v", err)
		}
	})

	// 6. Tipo de local invalido
	t.Run("invalid location", func(t *testing.T) {
		repo := &mockRepo{}
		svc := NewService(repo, 10)
		bad := LocationType("gym")
		req := &CreateAppointmentRequest{
			ProfessionalID: uuid.New(),
			StartsAt:       time.Now().Add(1 * time.Hour),
			EndsAt:         time.Now().Add(2 * time.Hour),
			LocationType:   &bad,
			PaymentMethod:  MethodPix,
		}
		_, _, err := svc.Create(context.Background(), clientUser, req)
		if !errors.Is(err, ErrInvalidLocation) {
			t.Errorf("esperava ErrInvalidLocation, obteve: %v", err)
		}
	})

	// 7. Tipo de local valido
	t.Run("valid location", func(t *testing.T) {
		repo := &mockRepo{}
		svc := NewService(repo, 10)
		home := LocationHome
		req := &CreateAppointmentRequest{
			ProfessionalID: uuid.New(),
			StartsAt:       time.Now().Add(1 * time.Hour),
			EndsAt:         time.Now().Add(2 * time.Hour),
			LocationType:   &home,
			LocationNotes:  "Rua das Flores, 100",
			PaymentMethod:  MethodPix,
		}
		_, _, err := svc.Create(context.Background(), clientUser, req)
		if err != nil {
			t.Errorf("esperava sucesso com local valido, obteve: %v", err)
		}
	})
}
