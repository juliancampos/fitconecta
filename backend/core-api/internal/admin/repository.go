package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ListAppointments(ctx context.Context, f AppointmentFilters) (Page[AdminAppointment], error)
	GetAppointment(ctx context.Context, id uuid.UUID) (*AdminAppointment, error)
	CancelAppointment(ctx context.Context, adminID, id uuid.UUID, reason, ip, userAgent string) (*AdminAppointment, error)

	ListPayments(ctx context.Context, f PaymentFilters) (Page[AdminPayment], error)
	SummaryPayments(ctx context.Context, f PaymentFilters) (*PaymentSummary, error)

	ListUsers(ctx context.Context, f UserFilters) (Page[AdminUser], error)
	GetUser(ctx context.Context, id uuid.UUID) (*AdminUser, error)
	UpdateUserStatus(ctx context.Context, adminID, targetID uuid.UUID, status, reason, ip, userAgent string) (*UpdateUserStatusResponse, error)
	SetProfessionalCREFVerified(ctx context.Context, adminID, userID uuid.UUID, number string, verifiedAt *time.Time, reason, ip, userAgent string) (bool, error)
}

type pgRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

const appointmentSelect = `
	SELECT a.id,
	       cu.id, c.name, cu.email,
	       pu.id, p.display_name, pu.email,
	       a.starts_at, a.ends_at, a.status::text,
	       s.name, a.location_type::text, COALESCE(a.location_notes, ''),
	       pay.id, pay.amount_cents, pay.status::text,
	       a.cancelled_at, a.cancel_reason, a.created_at
	FROM appointments a
	JOIN client_profiles c ON c.id = a.client_id
	JOIN users cu ON cu.id = c.user_id
	JOIN professional_profiles p ON p.id = a.professional_id
	JOIN users pu ON pu.id = p.user_id
	LEFT JOIN specialties s ON s.id = a.specialty_id
	LEFT JOIN payments pay ON pay.appointment_id = a.id
`

const paymentSelect = `
	SELECT pay.id, pay.appointment_id,
	       cu.id, c.name, cu.email,
	       pu.id, p.display_name, pu.email,
	       pay.amount_cents, pay.platform_fee_cents, pay.professional_net_cents,
	       pay.method::text, pay.status::text, pay.gateway_transaction_id,
	       pay.paid_at, pay.created_at
	FROM payments pay
	JOIN appointments a ON a.id = pay.appointment_id
	JOIN client_profiles c ON c.id = a.client_id
	JOIN users cu ON cu.id = c.user_id
	JOIN professional_profiles p ON p.id = a.professional_id
	JOIN users pu ON pu.id = p.user_id
`

const userSelect = `
	SELECT u.id, u.email, u.role::text, u.status::text, u.status_reason, u.status_changed_at,
	       COALESCE(cp.name, pp.display_name, u.email) AS display_name,
	       COALESCE(cp.phone, ''), COALESCE(cp.city, ''), COALESCE(cp.state, ''),
	       pp.id, pp.cref_number, pp.cref_verified_at, cp.id, u.created_at, u.updated_at
	FROM users u
	LEFT JOIN client_profiles cp ON cp.user_id = u.id
	LEFT JOIN professional_profiles pp ON pp.user_id = u.id
`

type scanner interface {
	Scan(dest ...any) error
}

func scanAppointment(row scanner) (*AdminAppointment, error) {
	var a AdminAppointment
	var specialtyName, locationType, paymentStatus *string
	var paymentID *uuid.UUID
	var amountCents *int
	if err := row.Scan(
		&a.ID,
		&a.Client.ID, &a.Client.Name, &a.Client.Email,
		&a.Professional.ID, &a.Professional.Name, &a.Professional.Email,
		&a.StartsAt, &a.EndsAt, &a.Status,
		&specialtyName, &locationType, &a.LocationNotes,
		&paymentID, &amountCents, &paymentStatus,
		&a.CancelledAt, &a.CancelReason, &a.CreatedAt,
	); err != nil {
		return nil, err
	}
	if specialtyName != nil {
		a.SpecialtyName = *specialtyName
	}
	if locationType != nil {
		a.LocationType = *locationType
	}
	a.PaymentID = paymentID
	a.AmountCents = amountCents
	if paymentStatus != nil {
		a.PaymentStatus = *paymentStatus
	}
	return &a, nil
}

func scanPayment(row scanner) (*AdminPayment, error) {
	var p AdminPayment
	if err := row.Scan(
		&p.ID, &p.AppointmentID,
		&p.Client.ID, &p.Client.Name, &p.Client.Email,
		&p.Professional.ID, &p.Professional.Name, &p.Professional.Email,
		&p.AmountCents, &p.PlatformFeeCents, &p.ProfessionalNetCents,
		&p.Method, &p.Status, &p.GatewayTransactionID,
		&p.PaidAt, &p.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &p, nil
}

func scanUser(row scanner) (*AdminUser, error) {
	var u AdminUser
	if err := row.Scan(
		&u.ID, &u.Email, &u.Role, &u.Status, &u.StatusReason, &u.StatusChangedAt,
		&u.DisplayName, &u.Phone, &u.City, &u.State,
		&u.ProfessionalID, &u.CREFNumber, &u.CREFVerifiedAt, &u.ClientID, &u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &u, nil
}

type whereBuilder struct {
	conditions []string
	args       []any
}

func (b *whereBuilder) add(format string, value any) int {
	b.args = append(b.args, value)
	n := len(b.args)
	b.conditions = append(b.conditions, fmt.Sprintf(format, n))
	return n
}

func (b *whereBuilder) addRaw(condition string) {
	b.conditions = append(b.conditions, condition)
}

func (b *whereBuilder) sql() string {
	if len(b.conditions) == 0 {
		return "TRUE"
	}
	return strings.Join(b.conditions, " AND ")
}

func addAppointmentFilters(b *whereBuilder, f AppointmentFilters) {
	if f.ClientID != nil {
		b.add("cu.id = $%d", *f.ClientID)
	}
	if f.ProfessionalID != nil {
		b.add("pu.id = $%d", *f.ProfessionalID)
	}
	if f.ClientQuery != "" {
		like := "%" + f.ClientQuery + "%"
		n := len(b.args) + 1
		b.args = append(b.args, like)
		b.conditions = append(b.conditions, fmt.Sprintf("(c.name ILIKE $%d OR cu.email::text ILIKE $%d)", n, n))
	}
	if f.ProfessionalQuery != "" {
		like := "%" + f.ProfessionalQuery + "%"
		n := len(b.args) + 1
		b.args = append(b.args, like)
		b.conditions = append(b.conditions, fmt.Sprintf("(p.display_name ILIKE $%d OR pu.email::text ILIKE $%d)", n, n))
	}
	if f.Query != "" {
		like := "%" + f.Query + "%"
		n := len(b.args) + 1
		b.args = append(b.args, like)
		b.conditions = append(b.conditions, fmt.Sprintf(
			"(c.name ILIKE $%d OR cu.email::text ILIKE $%d OR p.display_name ILIKE $%d OR pu.email::text ILIKE $%d)",
			n, n, n, n,
		))
	}
	if f.Status != "" {
		b.add("a.status::text = $%d", f.Status)
	}
	if f.DateFrom != nil {
		b.add("a.starts_at >= $%d", *f.DateFrom)
	}
	if f.DateTo != nil {
		b.add("a.starts_at < $%d", *f.DateTo)
	}
}

func addPaymentFilters(b *whereBuilder, f PaymentFilters) {
	if f.ClientID != nil {
		b.add("cu.id = $%d", *f.ClientID)
	}
	if f.ProfessionalID != nil {
		b.add("pu.id = $%d", *f.ProfessionalID)
	}
	if f.ClientQuery != "" {
		like := "%" + f.ClientQuery + "%"
		n := len(b.args) + 1
		b.args = append(b.args, like)
		b.conditions = append(b.conditions, fmt.Sprintf("(c.name ILIKE $%d OR cu.email::text ILIKE $%d)", n, n))
	}
	if f.ProfessionalQuery != "" {
		like := "%" + f.ProfessionalQuery + "%"
		n := len(b.args) + 1
		b.args = append(b.args, like)
		b.conditions = append(b.conditions, fmt.Sprintf("(p.display_name ILIKE $%d OR pu.email::text ILIKE $%d)", n, n))
	}
	if f.Query != "" {
		like := "%" + f.Query + "%"
		n := len(b.args) + 1
		b.args = append(b.args, like)
		b.conditions = append(b.conditions, fmt.Sprintf(
			"(c.name ILIKE $%d OR cu.email::text ILIKE $%d OR p.display_name ILIKE $%d OR pu.email::text ILIKE $%d)",
			n, n, n, n,
		))
	}
	if f.Status != "" {
		b.add("pay.status::text = $%d", f.Status)
	}
	if f.Method != "" {
		b.add("pay.method::text = $%d", f.Method)
	}
	dateColumn := "pay.created_at"
	if f.DateField == "paid_at" {
		dateColumn = "COALESCE(pay.paid_at, pay.created_at)"
	}
	if f.DateFrom != nil {
		b.add(dateColumn+" >= $%d", *f.DateFrom)
	}
	if f.DateTo != nil {
		b.add(dateColumn+" < $%d", *f.DateTo)
	}
}

func appointmentOrder(sort string) string {
	switch sort {
	case "starts_at_asc":
		return "a.starts_at ASC"
	case "created_at_desc":
		return "a.created_at DESC"
	default:
		return "a.starts_at DESC"
	}
}

func (r *pgRepository) ListAppointments(ctx context.Context, f AppointmentFilters) (Page[AdminAppointment], error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 20
	}

	b := &whereBuilder{}
	addAppointmentFilters(b, f)
	where := b.sql()

	var total int64
	countQuery := `SELECT COUNT(*) FROM appointments a
		JOIN client_profiles c ON c.id = a.client_id
		JOIN users cu ON cu.id = c.user_id
		JOIN professional_profiles p ON p.id = a.professional_id
		JOIN users pu ON pu.id = p.user_id
		LEFT JOIN payments pay ON pay.appointment_id = a.id
		WHERE ` + where
	if err := r.pool.QueryRow(ctx, countQuery, b.args...).Scan(&total); err != nil {
		return Page[AdminAppointment]{}, fmt.Errorf("erro ao contar agendamentos: %w", err)
	}

	args := append(append([]any{}, b.args...), f.PageSize, (f.Page-1)*f.PageSize)
	query := appointmentSelect + ` WHERE ` + where + ` ORDER BY ` + appointmentOrder(f.Sort) +
		fmt.Sprintf(` LIMIT $%d OFFSET $%d`, len(b.args)+1, len(b.args)+2)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return Page[AdminAppointment]{}, fmt.Errorf("erro ao listar agendamentos: %w", err)
	}
	defer rows.Close()

	items := make([]AdminAppointment, 0)
	for rows.Next() {
		item, err := scanAppointment(rows)
		if err != nil {
			return Page[AdminAppointment]{}, fmt.Errorf("erro ao mapear agendamento: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return Page[AdminAppointment]{}, err
	}

	return Page[AdminAppointment]{Items: items, Page: f.Page, PageSize: f.PageSize, Total: total}, nil
}

func (r *pgRepository) GetAppointment(ctx context.Context, id uuid.UUID) (*AdminAppointment, error) {
	item, err := scanAppointment(r.pool.QueryRow(ctx, appointmentSelect+` WHERE a.id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("erro ao obter agendamento: %w", err)
	}
	return item, nil
}

func (r *pgRepository) CancelAppointment(ctx context.Context, adminID, id uuid.UUID, reason, ip, userAgent string) (*AdminAppointment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var previousStatus string
	err = tx.QueryRow(ctx, `SELECT status::text FROM appointments WHERE id = $1 FOR UPDATE`, id).Scan(&previousStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if previousStatus != "pending_payment" && previousStatus != "confirmed" {
		return nil, ErrInvalidStatus
	}

	_, err = tx.Exec(ctx, `
		UPDATE appointments
		SET status = 'cancelled_by_admin', cancelled_at = NOW(), cancel_reason = $1, updated_at = NOW()
		WHERE id = $2`, reason, id)
	if err != nil {
		return nil, fmt.Errorf("erro ao cancelar agendamento: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE payments
		SET status = 'refund_pending', updated_at = NOW()
		WHERE appointment_id = $1 AND status = 'paid'`, id)
	if err != nil {
		return nil, fmt.Errorf("erro ao marcar reembolso pendente: %w", err)
	}

	if err := insertAudit(ctx, tx, adminID, "appointment_cancel", "appointment", id, map[string]any{
		"reason":          reason,
		"previous_status": previousStatus,
	}, ip, userAgent); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.GetAppointment(ctx, id)
}

func (r *pgRepository) ListPayments(ctx context.Context, f PaymentFilters) (Page[AdminPayment], error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 20
	}

	b := &whereBuilder{}
	addPaymentFilters(b, f)
	where := b.sql()

	var total int64
	countQuery := `SELECT COUNT(*) FROM payments pay
		JOIN appointments a ON a.id = pay.appointment_id
		JOIN client_profiles c ON c.id = a.client_id
		JOIN users cu ON cu.id = c.user_id
		JOIN professional_profiles p ON p.id = a.professional_id
		JOIN users pu ON pu.id = p.user_id
		WHERE ` + where
	if err := r.pool.QueryRow(ctx, countQuery, b.args...).Scan(&total); err != nil {
		return Page[AdminPayment]{}, fmt.Errorf("erro ao contar pagamentos: %w", err)
	}

	args := append(append([]any{}, b.args...), f.PageSize, (f.Page-1)*f.PageSize)
	query := paymentSelect + ` WHERE ` + where + fmt.Sprintf(` ORDER BY pay.created_at DESC LIMIT $%d OFFSET $%d`, len(b.args)+1, len(b.args)+2)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return Page[AdminPayment]{}, fmt.Errorf("erro ao listar pagamentos: %w", err)
	}
	defer rows.Close()

	items := make([]AdminPayment, 0)
	for rows.Next() {
		item, err := scanPayment(rows)
		if err != nil {
			return Page[AdminPayment]{}, fmt.Errorf("erro ao mapear pagamento: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return Page[AdminPayment]{}, err
	}

	return Page[AdminPayment]{Items: items, Page: f.Page, PageSize: f.PageSize, Total: total}, nil
}

func (r *pgRepository) SummaryPayments(ctx context.Context, f PaymentFilters) (*PaymentSummary, error) {
	b := &whereBuilder{}
	addPaymentFilters(b, f)
	where := b.sql()

	summary := &PaymentSummary{ByStatus: map[string]int64{}}
	query := `SELECT COUNT(*), COALESCE(SUM(pay.amount_cents), 0),
	                 COALESCE(SUM(pay.platform_fee_cents), 0), COALESCE(SUM(pay.professional_net_cents), 0)
	          FROM payments pay
	          JOIN appointments a ON a.id = pay.appointment_id
	          JOIN client_profiles c ON c.id = a.client_id
	          JOIN users cu ON cu.id = c.user_id
	          JOIN professional_profiles p ON p.id = a.professional_id
	          JOIN users pu ON pu.id = p.user_id
	          WHERE ` + where
	if err := r.pool.QueryRow(ctx, query, b.args...).Scan(
		&summary.Count, &summary.AmountCents, &summary.PlatformFeeCents, &summary.ProfessionalNetCents,
	); err != nil {
		return nil, fmt.Errorf("erro ao resumir pagamentos: %w", err)
	}

	statusQuery := `SELECT pay.status::text, COUNT(*)
	                FROM payments pay
	                JOIN appointments a ON a.id = pay.appointment_id
	                JOIN client_profiles c ON c.id = a.client_id
	                JOIN users cu ON cu.id = c.user_id
	                JOIN professional_profiles p ON p.id = a.professional_id
	                JOIN users pu ON pu.id = p.user_id
	                WHERE ` + where + ` GROUP BY pay.status`
	rows, err := r.pool.Query(ctx, statusQuery, b.args...)
	if err != nil {
		return nil, fmt.Errorf("erro ao agrupar pagamentos: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		summary.ByStatus[status] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return summary, nil
}

func (r *pgRepository) ListUsers(ctx context.Context, f UserFilters) (Page[AdminUser], error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 20
	}

	b := &whereBuilder{}
	b.addRaw("u.role IN ('client', 'professional')")
	if f.Role != "" {
		b.add("u.role::text = $%d", f.Role)
	}
	if f.Status != "" {
		b.add("u.status::text = $%d", f.Status)
	}
	if f.Query != "" {
		like := "%" + f.Query + "%"
		n := len(b.args) + 1
		b.args = append(b.args, like)
		b.conditions = append(b.conditions, fmt.Sprintf(
			"(u.email::text ILIKE $%d OR COALESCE(cp.name, pp.display_name, '') ILIKE $%d)",
			n, n,
		))
	}
	where := b.sql()

	var total int64
	countQuery := `SELECT COUNT(*) FROM users u
		LEFT JOIN client_profiles cp ON cp.user_id = u.id
		LEFT JOIN professional_profiles pp ON pp.user_id = u.id
		WHERE ` + where
	if err := r.pool.QueryRow(ctx, countQuery, b.args...).Scan(&total); err != nil {
		return Page[AdminUser]{}, fmt.Errorf("erro ao contar usuarios: %w", err)
	}

	args := append(append([]any{}, b.args...), f.PageSize, (f.Page-1)*f.PageSize)
	query := userSelect + ` WHERE ` + where + fmt.Sprintf(` ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d`, len(b.args)+1, len(b.args)+2)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return Page[AdminUser]{}, fmt.Errorf("erro ao listar usuarios: %w", err)
	}
	defer rows.Close()

	items := make([]AdminUser, 0)
	for rows.Next() {
		item, err := scanUser(rows)
		if err != nil {
			return Page[AdminUser]{}, fmt.Errorf("erro ao mapear usuario: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return Page[AdminUser]{}, err
	}

	return Page[AdminUser]{Items: items, Page: f.Page, PageSize: f.PageSize, Total: total}, nil
}

// SetProfessionalCREFVerified grava (ou limpa) a data da verificacao apenas se o
// CREF ainda for o numero consultado, e registra a acao na auditoria.
// Devolve false quando o numero mudou durante a consulta.
func (r *pgRepository) SetProfessionalCREFVerified(ctx context.Context, adminID, userID uuid.UUID, number string, verifiedAt *time.Time, reason, ip, userAgent string) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("falha ao iniciar transacao: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE professional_profiles
		SET cref_verified_at = $1
		WHERE user_id = $2 AND btrim(cref_number) = btrim($3);
	`, verifiedAt, userID, number)
	if err != nil {
		return false, fmt.Errorf("erro ao gravar verificacao do CREF: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if err := insertAudit(ctx, tx, adminID, "user_cref_revalidated", "user", userID, map[string]any{
		"reason":      reason,
		"cref_number": number,
	}, ip, userAgent); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("erro ao confirmar verificacao do CREF: %w", err)
	}
	return true, nil
}

func (r *pgRepository) GetUser(ctx context.Context, id uuid.UUID) (*AdminUser, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, userSelect+` WHERE u.id = $1 AND u.role IN ('client', 'professional')`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("erro ao obter usuario: %w", err)
	}
	count, err := r.futureAppointments(ctx, id)
	if err != nil {
		return nil, err
	}
	u.FutureAppointments = count
	return u, nil
}

func (r *pgRepository) UpdateUserStatus(ctx context.Context, adminID, targetID uuid.UUID, status, reason, ip, userAgent string) (*UpdateUserStatusResponse, error) {
	if status != "active" && status != "blocked" && status != "inactive" {
		return nil, ErrInvalidStatus
	}
	if status != "active" && strings.TrimSpace(reason) == "" {
		return nil, ErrReasonRequired
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var oldStatus string
	err = tx.QueryRow(ctx, `
		SELECT status::text FROM users
		WHERE id = $1 AND role IN ('client', 'professional')
		FOR UPDATE`, targetID).Scan(&oldStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE users
		SET status = $1::user_status,
		    status_reason = $2,
		    status_changed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $3`, status, nullIfEmpty(reason), targetID)
	if err != nil {
		return nil, fmt.Errorf("erro ao atualizar status do usuario: %w", err)
	}

	action := "user_status_changed"
	switch status {
	case "active":
		action = "user_reactivate"
	case "blocked":
		action = "user_block"
	case "inactive":
		action = "user_inactivate"
	}
	if err := insertAudit(ctx, tx, adminID, action, "user", targetID, map[string]any{
		"before": oldStatus,
		"after":  status,
		"reason": reason,
	}, ip, userAgent); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	updated, err := r.GetUser(ctx, targetID)
	if err != nil {
		return nil, err
	}
	return &UpdateUserStatusResponse{User: *updated, FutureAppointments: updated.FutureAppointments}, nil
}

func (r *pgRepository) futureAppointments(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM appointments a
		JOIN client_profiles cp ON cp.id = a.client_id
		JOIN professional_profiles pp ON pp.id = a.professional_id
		WHERE (cp.user_id = $1 OR pp.user_id = $1)
		  AND a.starts_at > NOW()
		  AND a.status IN ('pending_payment', 'confirmed')`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("erro ao contar agendamentos futuros: %w", err)
	}
	return count, nil
}

func insertAudit(ctx context.Context, tx pgx.Tx, adminID uuid.UUID, action, targetType string, targetID uuid.UUID, metadata map[string]any, ip, userAgent string) error {
	payload, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("erro ao serializar auditoria: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO admin_audit_log (admin_user_id, action, target_type, target_id, metadata, ip, user_agent)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7)`,
		adminID, action, targetType, targetID, string(payload), nullIfEmpty(ip), nullIfEmpty(userAgent))
	if err != nil {
		return fmt.Errorf("erro ao registrar auditoria: %w", err)
	}
	return nil
}

func nullIfEmpty(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}
