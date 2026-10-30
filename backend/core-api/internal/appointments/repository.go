package appointments

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSlotAlreadyBooked        = errors.New("o horario selecionado ja foi reservado por outro cliente")
	ErrAppointmentNotFound      = errors.New("agendamento nao encontrado")
	ErrUnauthorized             = errors.New("sem permissao para acessar este agendamento")
	ErrInvalidStatus            = errors.New("status do agendamento invalido para esta operacao")
	ErrInvalidSpecialty         = errors.New("o tipo de aula informado nao e oferecido por este profissional")
	ErrInvalidLocation          = errors.New("tipo de local invalido (use: home, studio ou online)")
	ErrProfessionalUnavailable  = errors.New("profissional indisponivel para novos agendamentos")
	ErrUnsupportedPaymentMethod = errors.New("metodo de pagamento nao suportado (use pix)")
	ErrPaymentNotFound          = errors.New("pagamento nao encontrado")
)

type Repository interface {
	CreateAppointmentWithPayment(ctx context.Context, clientID uuid.UUID, clientUserID, profUserID uuid.UUID, amountCents, platformFeePercent int, req *CreateAppointmentRequest) (*Appointment, *Payment, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Appointment, error)
	CreatePixCharge(ctx context.Context, appointmentID uuid.UUID, provider, providerPaymentID, copyPaste, qrImage string, expiresAt time.Time) (*Payment, error)
	MarkPaymentPaid(ctx context.Context, provider, providerPaymentID, rawStatus string, providerFeeCents *int) error
	ExpirePendingPayments(ctx context.Context, grace time.Duration) (int64, error)
	RegisterWebhookEvent(ctx context.Context, provider, eventID, eventType string, payload []byte) (uuid.UUID, bool, error)
	MarkWebhookProcessed(ctx context.Context, eventID uuid.UUID) error
	Cancel(ctx context.Context, id uuid.UUID, cancelledByRole string, reason string) (*Appointment, error)
	ListByClient(ctx context.Context, clientID uuid.UUID) ([]Appointment, error)
	ListByProfessional(ctx context.Context, profID uuid.UUID) ([]Appointment, error)

	GetClientProfile(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
	GetProfessionalProfile(ctx context.Context, profID uuid.UUID) (userID uuid.UUID, priceCents int, err error)
	GetProfessionalStatus(ctx context.Context, profID uuid.UUID) (string, error)
	ProfessionalOffersSpecialty(ctx context.Context, profID uuid.UUID, specialtyID int) (bool, error)
}

type pgRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toLocationType(s *string) *LocationType {
	if s == nil {
		return nil
	}
	lt := LocationType(*s)
	return &lt
}

func (r *pgRepository) CreateAppointmentWithPayment(
	ctx context.Context,
	clientID uuid.UUID,
	clientUserID, profUserID uuid.UUID,
	amountCents, platformFeePercent int,
	req *CreateAppointmentRequest,
) (*Appointment, *Payment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao iniciar transacao: %w", err)
	}
	defer tx.Rollback(ctx)

	// Calculo do split de 10%
	platformFee := (amountCents * platformFeePercent) / 100
	professionalNet := amountCents - platformFee

	reservedUntil := time.Now().Add(10 * time.Minute)

	// 1. Inserir agendamento com status pending_payment
	var locationArg *string
	if req.LocationType != nil {
		v := string(*req.LocationType)
		locationArg = &v
	}

	queryApp := `
		INSERT INTO appointments (professional_id, client_id, starts_at, ends_at, specialty_id, location_type, location_notes, status, reserved_until)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending_payment', $8)
		RETURNING id, professional_id, client_id, starts_at, ends_at, specialty_id, location_type, COALESCE(location_notes, '') AS location_notes, status, reserved_until, created_at, updated_at;
	`
	var app Appointment
	var locationStr *string
	err = tx.QueryRow(ctx, queryApp,
		req.ProfessionalID, clientID, req.StartsAt, req.EndsAt, req.SpecialtyID, locationArg, nullIfEmpty(req.LocationNotes), reservedUntil,
	).Scan(
		&app.ID, &app.ProfessionalID, &app.ClientID, &app.StartsAt, &app.EndsAt,
		&app.SpecialtyID, &locationStr, &app.LocationNotes, &app.Status, &app.ReservedUntil, &app.CreatedAt, &app.UpdatedAt,
	)
	if err == nil {
		app.LocationType = toLocationType(locationStr)
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23P01" { // exclusion_violation
			return nil, nil, ErrSlotAlreadyBooked
		}
		return nil, nil, fmt.Errorf("erro ao criar agendamento: %w", err)
	}

	// 2. Inserir registro financeiro com split
	queryPay := `
		INSERT INTO payments (
			appointment_id, client_id, professional_id,
			amount_cents, platform_fee_cents, professional_net_cents,
			method, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending')
		RETURNING id, appointment_id, client_id, professional_id, amount_cents,
		          platform_fee_cents, professional_net_cents, method, status, created_at;
	`
	var pay Payment
	err = tx.QueryRow(ctx, queryPay,
		app.ID, clientUserID, profUserID,
		amountCents, platformFee, professionalNet,
		req.PaymentMethod,
	).Scan(
		&pay.ID, &pay.AppointmentID, &pay.ClientID, &pay.ProfessionalID,
		&pay.AmountCents, &pay.PlatformFeeCents, &pay.ProfessionalNetCents,
		&pay.Method, &pay.Status, &pay.CreatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao gerar registro de pagamento com split: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("erro ao comitar transacao: %w", err)
	}

	app.PaymentInfo = &pay

	// Enriquece com o nome da especialidade (tipo de aula), se informada.
	if app.SpecialtyID != nil {
		_ = r.pool.QueryRow(ctx, "SELECT name FROM specialties WHERE id = $1;", *app.SpecialtyID).
			Scan(&app.SpecialtyName)
	}

	return &app, &pay, nil
}

func (r *pgRepository) GetByID(ctx context.Context, id uuid.UUID) (*Appointment, error) {
	query := `
		SELECT a.id, a.professional_id, a.client_id, a.starts_at, a.ends_at, a.specialty_id,
		       a.location_type, COALESCE(a.location_notes, '') AS location_notes, a.status,
		       a.reserved_until, a.cancelled_at, a.cancel_reason, a.created_at, a.updated_at,
		       p.display_name, c.name, COALESCE(s.name, ''),
		       pay.id, pay.amount_cents, pay.platform_fee_cents, pay.professional_net_cents,
		       pay.method, pay.status, pay.gateway_transaction_id,
		       COALESCE(pay.provider, ''), pay.provider_payment_id, pay.provider_fee_cents,
		       pay.expires_at, COALESCE(pay.raw_status, ''), pay.paid_at
		FROM appointments a
		JOIN professional_profiles p ON p.id = a.professional_id
		JOIN client_profiles c ON c.id = a.client_id
		LEFT JOIN specialties s ON s.id = a.specialty_id
		LEFT JOIN payments pay ON pay.appointment_id = a.id
		WHERE a.id = $1;
	`
	var app Appointment
	var pay Payment
	var profName, cliName, specName string
	var locationStr *string
	var payID *uuid.UUID

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&app.ID, &app.ProfessionalID, &app.ClientID, &app.StartsAt, &app.EndsAt, &app.SpecialtyID,
		&locationStr, &app.LocationNotes, &app.Status,
		&app.ReservedUntil, &app.CancelledAt, &app.CancelReason, &app.CreatedAt, &app.UpdatedAt,
		&profName, &cliName, &specName,
		&payID, &pay.AmountCents, &pay.PlatformFeeCents, &pay.ProfessionalNetCents,
		&pay.Method, &pay.Status, &pay.GatewayTransactionID,
		&pay.Provider, &pay.ProviderPaymentID, &pay.ProviderFeeCents,
		&pay.ExpiresAt, &pay.RawStatus, &pay.PaidAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAppointmentNotFound
		}
		return nil, err
	}

	app.ProfessionalName = profName
	app.ClientName = cliName
	app.SpecialtyName = specName
	app.LocationType = toLocationType(locationStr)

	if payID != nil {
		pay.ID = *payID
		pay.AppointmentID = app.ID
		app.PaymentInfo = &pay
	}

	return &app, nil
}

func (r *pgRepository) CreatePixCharge(
	ctx context.Context,
	appointmentID uuid.UUID,
	provider, providerPaymentID, copyPaste, qrImage string,
	expiresAt time.Time,
) (*Payment, error) {
	query := `
		UPDATE payments
		SET provider = $2,
		    provider_payment_id = $3,
		    expires_at = $4,
		    raw_status = 'pending',
		    metadata = jsonb_build_object('copy_paste', $5::text, 'qr_code_image', $6::text),
		    updated_at = NOW()
		WHERE appointment_id = $1 AND status = 'pending'
		RETURNING id, appointment_id, client_id, professional_id, amount_cents,
		          platform_fee_cents, professional_net_cents, method, status,
		          gateway_transaction_id, provider, provider_payment_id, provider_fee_cents,
		          expires_at, raw_status, paid_at, created_at;
	`
	var pay Payment
	err := r.pool.QueryRow(ctx, query, appointmentID, provider, providerPaymentID, expiresAt, copyPaste, qrImage).Scan(
		&pay.ID, &pay.AppointmentID, &pay.ClientID, &pay.ProfessionalID,
		&pay.AmountCents, &pay.PlatformFeeCents, &pay.ProfessionalNetCents,
		&pay.Method, &pay.Status, &pay.GatewayTransactionID, &pay.Provider,
		&pay.ProviderPaymentID, &pay.ProviderFeeCents, &pay.ExpiresAt,
		&pay.RawStatus, &pay.PaidAt, &pay.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidStatus
		}
		return nil, err
	}
	return &pay, nil
}

func (r *pgRepository) MarkPaymentPaid(
	ctx context.Context,
	provider, providerPaymentID, rawStatus string,
	providerFeeCents *int,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var paymentID, appointmentID uuid.UUID
	var paymentStatus string
	err = tx.QueryRow(ctx, `
		SELECT id, appointment_id, status::text
		FROM payments
		WHERE provider = $1 AND provider_payment_id = $2
		FOR UPDATE;
	`, provider, providerPaymentID).Scan(&paymentID, &appointmentID, &paymentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPaymentNotFound
		}
		return err
	}

	if paymentStatus == "paid" || paymentStatus == "refunded" {
		return tx.Commit(ctx)
	}

	var appointmentStatus string
	err = tx.QueryRow(ctx, `SELECT status::text FROM appointments WHERE id = $1 FOR UPDATE;`, appointmentID).
		Scan(&appointmentStatus)
	if err != nil {
		return err
	}

	next := "paid"
	switch appointmentStatus {
	case "expired", "cancelled_by_client", "cancelled_by_professional", "cancelled_by_admin":
		next = "refund_pending"
	case "pending_payment":
		if _, err := tx.Exec(ctx, `UPDATE appointments SET status = 'confirmed', updated_at = NOW() WHERE id = $1;`, appointmentID); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE payments
		SET status = $2,
		    raw_status = $3,
		    provider_fee_cents = COALESCE($4, provider_fee_cents),
		    paid_at = COALESCE(paid_at, NOW()),
		    updated_at = NOW()
		WHERE id = $1;
	`, paymentID, next, rawStatus, providerFeeCents); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *pgRepository) ExpirePendingPayments(ctx context.Context, grace time.Duration) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	query := `
		WITH expired AS (
			UPDATE appointments a
			SET status = 'expired', updated_at = NOW()
			WHERE a.status = 'pending_payment'
			  AND a.reserved_until IS NOT NULL
			  AND a.reserved_until < NOW() - ($1::double precision * interval '1 second')
			  AND NOT EXISTS (
			      SELECT 1 FROM payments p
			      WHERE p.appointment_id = a.id
			        AND p.status IN ('paid', 'refund_pending', 'refunded')
			  )
			RETURNING a.id
		)
		UPDATE payments p
		SET status = 'expired', updated_at = NOW()
		FROM expired e
		WHERE p.appointment_id = e.id AND p.status = 'pending';
	`
	tag, err := tx.Exec(ctx, query, grace.Seconds())
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *pgRepository) RegisterWebhookEvent(
	ctx context.Context,
	provider, eventID, eventType string,
	payload []byte,
) (uuid.UUID, bool, error) {
	var id uuid.UUID
	var shouldProcess bool
	err := r.pool.QueryRow(ctx, `
		INSERT INTO payment_webhook_events (provider, provider_event_id, event_type, payload)
		VALUES ($1, $2, $3, $4::jsonb)
		ON CONFLICT (provider, provider_event_id)
		DO UPDATE SET payload = EXCLUDED.payload
		RETURNING id, (processed_at IS NULL);
	`, provider, eventID, eventType, string(payload)).Scan(&id, &shouldProcess)
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, shouldProcess, nil
}

func (r *pgRepository) MarkWebhookProcessed(ctx context.Context, eventID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE payment_webhook_events SET processed_at = NOW()
		WHERE id = $1 AND processed_at IS NULL;
	`, eventID)
	return err
}

func (r *pgRepository) Cancel(ctx context.Context, id uuid.UUID, cancelledByRole string, reason string) (*Appointment, error) {
	status := StatusCancelledByClient
	switch cancelledByRole {
	case "professional":
		status = StatusCancelledByProfessional
	case "admin":
		status = StatusCancelledByAdmin
	}

	query := `
		UPDATE appointments
		SET status = $1, cancelled_at = NOW(), cancel_reason = $2, updated_at = NOW()
		WHERE id = $3 AND status IN ('pending_payment', 'confirmed')
		RETURNING id, professional_id, client_id, starts_at, ends_at, status, cancelled_at, cancel_reason, updated_at;
	`
	var app Appointment
	err := r.pool.QueryRow(ctx, query, status, reason, id).Scan(
		&app.ID, &app.ProfessionalID, &app.ClientID, &app.StartsAt, &app.EndsAt,
		&app.Status, &app.CancelledAt, &app.CancelReason, &app.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidStatus
		}
		return nil, err
	}

	if _, err := r.pool.Exec(ctx,
		`UPDATE payments SET status = 'expired', updated_at = NOW() WHERE appointment_id = $1 AND status = 'pending';`,
		id,
	); err != nil {
		return nil, err
	}

	return &app, nil
}

func (r *pgRepository) ListByClient(ctx context.Context, clientID uuid.UUID) ([]Appointment, error) {
	query := `
		SELECT a.id, a.professional_id, a.client_id, a.starts_at, a.ends_at, a.specialty_id,
		       a.location_type, COALESCE(a.location_notes, '') AS location_notes, a.status,
		       a.cancelled_at, a.cancel_reason, a.created_at, a.updated_at,
		       p.display_name, COALESCE(s.name, ''), pay.amount_cents, pay.status
		FROM appointments a
		JOIN professional_profiles p ON p.id = a.professional_id
		LEFT JOIN specialties s ON s.id = a.specialty_id
		LEFT JOIN payments pay ON pay.appointment_id = a.id
		WHERE a.client_id = $1
		ORDER BY a.starts_at DESC;
	`
	rows, err := r.pool.Query(ctx, query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Appointment
	for rows.Next() {
		var a Appointment
		var profName, specName string
		var locationStr *string
		var payAmount *int
		var payStatus *PaymentStatus

		err := rows.Scan(
			&a.ID, &a.ProfessionalID, &a.ClientID, &a.StartsAt, &a.EndsAt, &a.SpecialtyID,
			&locationStr, &a.LocationNotes, &a.Status,
			&a.CancelledAt, &a.CancelReason, &a.CreatedAt, &a.UpdatedAt,
			&profName, &specName, &payAmount, &payStatus,
		)
		if err == nil {
			a.ProfessionalName = profName
			a.SpecialtyName = specName
			a.LocationType = toLocationType(locationStr)
			if payAmount != nil && payStatus != nil {
				a.PaymentInfo = &Payment{
					AmountCents: *payAmount,
					Status:      *payStatus,
				}
			}
			list = append(list, a)
		}
	}
	return list, nil
}

func (r *pgRepository) ListByProfessional(ctx context.Context, profID uuid.UUID) ([]Appointment, error) {
	query := `
		SELECT a.id, a.professional_id, a.client_id, a.starts_at, a.ends_at, a.specialty_id,
		       a.location_type, COALESCE(a.location_notes, '') AS location_notes, a.status,
		       a.cancelled_at, a.cancel_reason, a.created_at, a.updated_at,
		       c.name, COALESCE(s.name, ''), pay.amount_cents, pay.platform_fee_cents, pay.professional_net_cents, pay.status
		FROM appointments a
		JOIN client_profiles c ON c.id = a.client_id
		LEFT JOIN specialties s ON s.id = a.specialty_id
		LEFT JOIN payments pay ON pay.appointment_id = a.id
		WHERE a.professional_id = $1
		ORDER BY a.starts_at DESC;
	`
	rows, err := r.pool.Query(ctx, query, profID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Appointment
	for rows.Next() {
		var a Appointment
		var cliName, specName string
		var locationStr *string
		var payAmount, feeCents, netCents *int
		var payStatus *PaymentStatus

		err := rows.Scan(
			&a.ID, &a.ProfessionalID, &a.ClientID, &a.StartsAt, &a.EndsAt, &a.SpecialtyID,
			&locationStr, &a.LocationNotes, &a.Status,
			&a.CancelledAt, &a.CancelReason, &a.CreatedAt, &a.UpdatedAt,
			&cliName, &specName, &payAmount, &feeCents, &netCents, &payStatus,
		)
		if err == nil {
			a.ClientName = cliName
			a.SpecialtyName = specName
			a.LocationType = toLocationType(locationStr)
			if payAmount != nil {
				a.PaymentInfo = &Payment{
					AmountCents:          *payAmount,
					PlatformFeeCents:     *feeCents,
					ProfessionalNetCents: *netCents,
					Status:               *payStatus,
				}
			}
			list = append(list, a)
		}
	}
	return list, nil
}

func (r *pgRepository) GetClientProfile(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, "SELECT id FROM client_profiles WHERE user_id = $1;", userID).Scan(&id)
	return id, err
}

func (r *pgRepository) GetProfessionalProfile(ctx context.Context, profID uuid.UUID) (uuid.UUID, int, error) {
	var userID uuid.UUID
	var price int
	err := r.pool.QueryRow(ctx, "SELECT user_id, session_price_cents FROM professional_profiles WHERE id = $1;", profID).Scan(&userID, &price)
	return userID, price, err
}

func (r *pgRepository) GetProfessionalStatus(ctx context.Context, profID uuid.UUID) (string, error) {
	var status string
	err := r.pool.QueryRow(ctx,
		`SELECT u.status::text
		 FROM professional_profiles p
		 JOIN users u ON u.id = p.user_id
		 WHERE p.id = $1;`,
		profID,
	).Scan(&status)
	return status, err
}

func (r *pgRepository) ProfessionalOffersSpecialty(ctx context.Context, profID uuid.UUID, specialtyID int) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM professional_specialties WHERE professional_id = $1 AND specialty_id = $2);",
		profID, specialtyID,
	).Scan(&exists)
	return exists, err
}
