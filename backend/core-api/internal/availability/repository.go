package availability

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrProfessionalUnavailable = errors.New("profissional indisponivel")

type BookedInterval struct {
	StartsAt time.Time
	EndsAt   time.Time
}

type Repository interface {
	CreateRule(ctx context.Context, profID uuid.UUID, req *CreateRuleRequest) (*AvailabilityRule, error)
	ListRules(ctx context.Context, profID uuid.UUID) ([]AvailabilityRule, error)
	DeleteRule(ctx context.Context, profID, ruleID uuid.UUID) error

	CreateException(ctx context.Context, profID uuid.UUID, req *CreateExceptionRequest) (*AvailabilityException, error)
	ListExceptions(ctx context.Context, profID uuid.UUID, from, to time.Time) ([]AvailabilityException, error)
	DeleteException(ctx context.Context, profID, excID uuid.UUID) error

	GetBookedIntervals(ctx context.Context, profID uuid.UUID, from, to time.Time) ([]BookedInterval, error)
	GetProfessionalTimezone(ctx context.Context, profID uuid.UUID) (string, error)
}

type pgRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

func (r *pgRepository) CreateRule(ctx context.Context, profID uuid.UUID, req *CreateRuleRequest) (*AvailabilityRule, error) {
	query := `
		INSERT INTO availability_rules (professional_id, weekday, start_time, end_time, slot_minutes)
		VALUES ($1, $2, $3::time, $4::time, $5)
		RETURNING id, professional_id, weekday, start_time::text, end_time::text, slot_minutes, created_at;
	`
	var rule AvailabilityRule
	err := r.pool.QueryRow(ctx, query,
		profID, req.Weekday, req.StartTime, req.EndTime, req.SlotMinutes,
	).Scan(
		&rule.ID, &rule.ProfessionalID, &rule.Weekday, &rule.StartTime, &rule.EndTime, &rule.SlotMinutes, &rule.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao inserir regra de disponibilidade: %w", err)
	}
	return &rule, nil
}

func (r *pgRepository) ListRules(ctx context.Context, profID uuid.UUID) ([]AvailabilityRule, error) {
	query := `
		SELECT id, professional_id, weekday, start_time::text, end_time::text, slot_minutes, created_at
		FROM availability_rules
		WHERE professional_id = $1
		ORDER BY weekday ASC, start_time ASC;
	`
	rows, err := r.pool.Query(ctx, query, profID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []AvailabilityRule
	for rows.Next() {
		var rule AvailabilityRule
		err := rows.Scan(
			&rule.ID, &rule.ProfessionalID, &rule.Weekday, &rule.StartTime, &rule.EndTime, &rule.SlotMinutes, &rule.CreatedAt,
		)
		if err == nil {
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

func (r *pgRepository) DeleteRule(ctx context.Context, profID, ruleID uuid.UUID) error {
	query := `DELETE FROM availability_rules WHERE id = $1 AND professional_id = $2;`
	_, err := r.pool.Exec(ctx, query, ruleID, profID)
	return err
}

func (r *pgRepository) CreateException(ctx context.Context, profID uuid.UUID, req *CreateExceptionRequest) (*AvailabilityException, error) {
	query := `
		INSERT INTO availability_exceptions (professional_id, starts_at, ends_at, reason)
		VALUES ($1, $2, $3, $4)
		RETURNING id, professional_id, starts_at, ends_at, reason, created_at;
	`
	var exc AvailabilityException
	err := r.pool.QueryRow(ctx, query, profID, req.StartsAt, req.EndsAt, req.Reason).Scan(
		&exc.ID, &exc.ProfessionalID, &exc.StartsAt, &exc.EndsAt, &exc.Reason, &exc.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar excecao de agenda: %w", err)
	}
	return &exc, nil
}

func (r *pgRepository) ListExceptions(ctx context.Context, profID uuid.UUID, from, to time.Time) ([]AvailabilityException, error) {
	query := `
		SELECT id, professional_id, starts_at, ends_at, reason, created_at
		FROM availability_exceptions
		WHERE professional_id = $1
		  AND ends_at >= $2 AND starts_at <= $3
		ORDER BY starts_at ASC;
	`
	rows, err := r.pool.Query(ctx, query, profID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []AvailabilityException
	for rows.Next() {
		var exc AvailabilityException
		err := rows.Scan(&exc.ID, &exc.ProfessionalID, &exc.StartsAt, &exc.EndsAt, &exc.Reason, &exc.CreatedAt)
		if err == nil {
			list = append(list, exc)
		}
	}
	return list, nil
}

func (r *pgRepository) DeleteException(ctx context.Context, profID, excID uuid.UUID) error {
	query := `DELETE FROM availability_exceptions WHERE id = $1 AND professional_id = $2;`
	_, err := r.pool.Exec(ctx, query, excID, profID)
	return err
}

func (r *pgRepository) GetBookedIntervals(ctx context.Context, profID uuid.UUID, from, to time.Time) ([]BookedInterval, error) {
	query := `
		SELECT starts_at, ends_at
		FROM appointments
		WHERE professional_id = $1
		  AND status IN ('pending_payment', 'confirmed')
		  AND ends_at > $2 AND starts_at < $3
		ORDER BY starts_at ASC;
	`
	rows, err := r.pool.Query(ctx, query, profID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var intervals []BookedInterval
	for rows.Next() {
		var b BookedInterval
		if err := rows.Scan(&b.StartsAt, &b.EndsAt); err == nil {
			intervals = append(intervals, b)
		}
	}
	return intervals, nil
}

func (r *pgRepository) GetProfessionalTimezone(ctx context.Context, profID uuid.UUID) (string, error) {
	query := `
		SELECT p.timezone
		FROM professional_profiles p
		JOIN users u ON u.id = p.user_id
		WHERE p.id = $1 AND u.status = 'active';`
	var tz string
	err := r.pool.QueryRow(ctx, query, profID).Scan(&tz)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrProfessionalUnavailable
		}
		return "", err
	}
	return tz, nil
}
