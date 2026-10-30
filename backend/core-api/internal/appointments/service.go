package appointments

import (
	"context"
	"errors"
	"fmt"
	"time"

	"fitconecta/core-api/internal/payments"
	"fitconecta/core-api/internal/platform/middleware"
	"github.com/google/uuid"
)

type Service interface {
	Create(ctx context.Context, clientUser middleware.AuthUser, req *CreateAppointmentRequest) (*Appointment, *Payment, error)
	GetByID(ctx context.Context, user middleware.AuthUser, id uuid.UUID) (*Appointment, error)
	CreatePixCharge(ctx context.Context, user middleware.AuthUser, appointmentID uuid.UUID) (*PixChargeResponse, error)
	HandleWebhook(ctx context.Context, event *payments.WebhookEvent) error
	ExpirePending(ctx context.Context) (int64, error)
	Cancel(ctx context.Context, user middleware.AuthUser, id uuid.UUID, reason string) (*Appointment, error)
	ListMyAppointments(ctx context.Context, user middleware.AuthUser) ([]Appointment, error)
}

type service struct {
	repo               Repository
	platformFeePercent int
	provider           payments.Provider
	pixExpiration      time.Duration
	pixGrace           time.Duration
}

func NewService(repo Repository, platformFeePercent int) Service {
	return NewServiceWithPayments(repo, platformFeePercent, nil, 10*time.Minute, 2*time.Minute)
}

func NewServiceWithPayments(
	repo Repository,
	platformFeePercent int,
	provider payments.Provider,
	pixExpiration, pixGrace time.Duration,
) Service {
	if platformFeePercent <= 0 {
		platformFeePercent = 10
	}
	if pixExpiration <= 0 {
		pixExpiration = 10 * time.Minute
	}
	if pixGrace < 0 {
		pixGrace = 0
	}
	return &service{
		repo:               repo,
		platformFeePercent: platformFeePercent,
		provider:           provider,
		pixExpiration:      pixExpiration,
		pixGrace:           pixGrace,
	}
}

func (s *service) Create(ctx context.Context, clientUser middleware.AuthUser, req *CreateAppointmentRequest) (*Appointment, *Payment, error) {
	if clientUser.Role != "client" {
		return nil, nil, fmt.Errorf("%w: apenas clientes podem agendar sessoes", ErrUnauthorized)
	}
	if clientUser.Status == "blocked" || clientUser.Status == "inactive" {
		return nil, nil, fmt.Errorf("%w: sua conta nao pode criar novos agendamentos", ErrUnauthorized)
	}

	if !req.StartsAt.Before(req.EndsAt) {
		return nil, nil, errors.New("horario inicial deve ser anterior ao horario final")
	}

	if req.StartsAt.Before(time.Now()) {
		return nil, nil, errors.New("nao e possivel agendar em horarios passados")
	}

	if req.PaymentMethod == "" {
		req.PaymentMethod = MethodPix
	}
	if req.PaymentMethod != MethodPix {
		return nil, nil, ErrUnsupportedPaymentMethod
	}

	// Validar o tipo de local quando informado
	if req.LocationType != nil && !req.LocationType.Valid() {
		return nil, nil, ErrInvalidLocation
	}

	// Buscar perfil de cliente
	clientProfileID, err := s.repo.GetClientProfile(ctx, clientUser.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("perfil de cliente nao encontrado: %w", err)
	}

	// Buscar dados do profissional (preco e user_id para split)
	profUserID, priceCents, err := s.repo.GetProfessionalProfile(ctx, req.ProfessionalID)
	if err != nil {
		return nil, nil, fmt.Errorf("profissional nao encontrado: %w", err)
	}

	profStatus, err := s.repo.GetProfessionalStatus(ctx, req.ProfessionalID)
	if err != nil {
		return nil, nil, fmt.Errorf("nao foi possivel validar o profissional: %w", err)
	}
	if profStatus != "active" {
		return nil, nil, ErrProfessionalUnavailable
	}

	// Validar o tipo de aula (especialidade) quando informado
	if req.SpecialtyID != nil {
		offered, err := s.repo.ProfessionalOffersSpecialty(ctx, req.ProfessionalID, *req.SpecialtyID)
		if err != nil {
			return nil, nil, fmt.Errorf("erro ao validar tipo de aula: %w", err)
		}
		if !offered {
			return nil, nil, ErrInvalidSpecialty
		}
	}

	return s.repo.CreateAppointmentWithPayment(
		ctx,
		clientProfileID,
		clientUser.ID,
		profUserID,
		priceCents,
		s.platformFeePercent,
		req,
	)
}

func (s *service) GetByID(ctx context.Context, user middleware.AuthUser, id uuid.UUID) (*Appointment, error) {
	app, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validacao estrita contra IDOR
	if user.Role != "admin" {
		if user.Role == "client" {
			clientProfileID, err := s.repo.GetClientProfile(ctx, user.ID)
			if err != nil || app.ClientID != clientProfileID {
				return nil, ErrUnauthorized
			}
		} else if user.Role == "professional" {
			profUserID, _, err := s.repo.GetProfessionalProfile(ctx, app.ProfessionalID)
			if err != nil || profUserID != user.ID {
				return nil, ErrUnauthorized
			}
		}
	}

	return app, nil
}

func (s *service) CreatePixCharge(
	ctx context.Context,
	user middleware.AuthUser,
	appointmentID uuid.UUID,
) (*PixChargeResponse, error) {
	if user.Role != "client" {
		return nil, fmt.Errorf("%w: apenas clientes podem pagar agendamentos", ErrUnauthorized)
	}
	if s.provider == nil {
		return nil, payments.ErrProviderUnavailable
	}

	app, err := s.GetByID(ctx, user, appointmentID)
	if err != nil {
		return nil, err
	}
	if app.Status != StatusPendingPayment {
		return nil, ErrInvalidStatus
	}
	if app.PaymentInfo == nil {
		return nil, ErrPaymentNotFound
	}
	pay := app.PaymentInfo
	if pay.Method != MethodPix {
		return nil, ErrUnsupportedPaymentMethod
	}
	if pay.Status != PaymentPending {
		return nil, ErrInvalidStatus
	}

	if pay.ProviderPaymentID != nil && pay.ExpiresAt != nil && pay.ExpiresAt.After(time.Now()) {
		existing, err := s.provider.GetPix(ctx, *pay.ProviderPaymentID)
		switch {
		case err == nil && existing.Status == payments.StatusPending:
			return pixResponse(app.ID, pay, existing), nil
		case err != nil && !errors.Is(err, payments.ErrChargeNotFound):
			return nil, fmt.Errorf("%w: %v", payments.ErrProviderUnavailable, err)
		}
	}

	now := time.Now()
	expiresAt := now.Add(s.pixExpiration)
	if app.ReservedUntil != nil && app.ReservedUntil.After(now) && app.ReservedUntil.Before(expiresAt) {
		expiresAt = *app.ReservedUntil
	}

	charge, err := s.provider.CreatePix(ctx, payments.ChargeInput{
		PaymentID:     pay.ID,
		AppointmentID: app.ID,
		AmountCents:   pay.AmountCents,
		Description:   "Sessao FitConecta",
		ExpiresAt:     expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", payments.ErrProviderUnavailable, err)
	}

	updated, err := s.repo.CreatePixCharge(
		ctx, app.ID, s.provider.Name(), charge.ProviderPaymentID,
		charge.QRCodeCopyPaste, charge.QRCodeImage, charge.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	return pixResponse(app.ID, updated, charge), nil
}

func pixResponse(appointmentID uuid.UUID, pay *Payment, charge *payments.Charge) *PixChargeResponse {
	expiresAt := charge.ExpiresAt
	if pay.ExpiresAt != nil {
		expiresAt = *pay.ExpiresAt
	}
	return &PixChargeResponse{
		PaymentID:         pay.ID,
		AppointmentID:     appointmentID,
		Method:            pay.Method,
		Status:            pay.Status,
		AmountCents:       pay.AmountCents,
		ExpiresAt:         expiresAt,
		QRCodeImage:       charge.QRCodeImage,
		CopyPaste:         charge.QRCodeCopyPaste,
		ProviderPaymentID: charge.ProviderPaymentID,
	}
}

func (s *service) HandleWebhook(ctx context.Context, event *payments.WebhookEvent) error {
	if s.provider == nil {
		return payments.ErrProviderUnavailable
	}

	eventID, shouldProcess, err := s.repo.RegisterWebhookEvent(
		ctx, s.provider.Name(), event.ProviderEventID, event.Type, event.Payload,
	)
	if err != nil {
		return err
	}
	if !shouldProcess {
		return nil
	}

	if event.Status == payments.StatusPaid {
		charge, err := s.provider.GetPix(ctx, event.ProviderPaymentID)
		if err != nil {
			return fmt.Errorf("%w: %v", payments.ErrInvalidWebhook, err)
		}
		if charge.Status != payments.StatusPaid {
			return fmt.Errorf("%w: cobranca ainda nao liquidada", payments.ErrInvalidWebhook)
		}
		if event.AmountCents > 0 && charge.AmountCents > 0 && event.AmountCents != charge.AmountCents {
			return fmt.Errorf("%w: valor divergente", payments.ErrInvalidWebhook)
		}
		if err := s.repo.MarkPaymentPaid(
			ctx, s.provider.Name(), event.ProviderPaymentID, charge.Status, nil,
		); err != nil {
			return err
		}
	}

	return s.repo.MarkWebhookProcessed(ctx, eventID)
}

func (s *service) ExpirePending(ctx context.Context) (int64, error) {
	return s.repo.ExpirePendingPayments(ctx, s.pixGrace)
}

func (s *service) Cancel(ctx context.Context, user middleware.AuthUser, id uuid.UUID, reason string) (*Appointment, error) {
	app, err := s.GetByID(ctx, user, id)
	if err != nil {
		return nil, err
	}

	if app.Status != StatusPendingPayment && app.Status != StatusConfirmed {
		return nil, fmt.Errorf("%w: agendamento ja se encontra cancelado ou concluido", ErrInvalidStatus)
	}

	return s.repo.Cancel(ctx, id, user.Role, reason)
}

func (s *service) ListMyAppointments(ctx context.Context, user middleware.AuthUser) ([]Appointment, error) {
	if user.Role == "client" {
		clientProfileID, err := s.repo.GetClientProfile(ctx, user.ID)
		if err != nil {
			return nil, fmt.Errorf("perfil de cliente nao encontrado: %w", err)
		}
		return s.repo.ListByClient(ctx, clientProfileID)
	}

	// Profissional
	var profID uuid.UUID
	err := s.repo.(*pgRepository).pool.QueryRow(ctx, "SELECT id FROM professional_profiles WHERE user_id = $1;", user.ID).Scan(&profID)
	if err != nil {
		return nil, fmt.Errorf("perfil de profissional nao encontrado: %w", err)
	}

	return s.repo.ListByProfessional(ctx, profID)
}
