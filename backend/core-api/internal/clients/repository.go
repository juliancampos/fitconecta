package clients

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrProfileNotFound indica usuario sem linha em client_profiles (ex.: admin).
var ErrProfileNotFound = errors.New("perfil nao encontrado")

type Repository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*ClientPersonalProfile, error)
	Update(ctx context.Context, userID uuid.UUID, data *ClientPersonalProfile) (*ClientPersonalProfile, error)
}

type pgRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

// Colunas explicitas: client_profiles tambem e lida por auth e admin, e
// SELECT * vazaria dado de saude para fora deste endpoint (ADR 0012 §6).
const personalColumns = `birth_date, sex, height_cm, weight_kg, activity_level, objective, cpf`

func scanPersonal(row pgx.Row) (*ClientPersonalProfile, error) {
	var p ClientPersonalProfile
	err := row.Scan(&p.BirthDate, &p.Sex, &p.HeightCM, &p.WeightKG, &p.ActivityLevel, &p.Objective, &p.CPF)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *pgRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*ClientPersonalProfile, error) {
	query := `SELECT ` + personalColumns + ` FROM client_profiles WHERE user_id = $1;`

	p, err := scanPersonal(r.pool.QueryRow(ctx, query, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("erro ao obter dados pessoais do cliente: %w", err)
	}
	return p, nil
}

func (r *pgRepository) Update(ctx context.Context, userID uuid.UUID, data *ClientPersonalProfile) (*ClientPersonalProfile, error) {
	query := `
		UPDATE client_profiles
		SET birth_date = $1, sex = $2, height_cm = $3, weight_kg = $4, activity_level = $5, objective = $6, cpf = $7, updated_at = NOW()
		WHERE user_id = $8
		RETURNING ` + personalColumns + `;`

	p, err := scanPersonal(r.pool.QueryRow(ctx, query,
		data.BirthDate, data.Sex, data.HeightCM, data.WeightKG, data.ActivityLevel, data.Objective, data.CPF, userID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("erro ao salvar dados pessoais do cliente: %w", err)
	}
	return p, nil
}
