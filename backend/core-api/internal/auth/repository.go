package auth

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
	ErrEmailAlreadyExists = errors.New("o e-mail informado ja esta cadastrado")
	ErrUserNotFound       = errors.New("usuario nao encontrado")
)

type Repository interface {
	CreateUserWithProfile(ctx context.Context, u *User, req *RegisterRequest) (*UserProfileResponse, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetProfileByUserID(ctx context.Context, userID uuid.UUID, role Role) (*UserProfileResponse, error)
	SetCREFVerified(ctx context.Context, userID uuid.UUID, number string) error
	SetCRNVerified(ctx context.Context, userID uuid.UUID, number string) error
}

type pgRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

func (r *pgRepository) CreateUserWithProfile(ctx context.Context, u *User, req *RegisterRequest) (*UserProfileResponse, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao iniciar transacao: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Inserir usuario na tabela users
	queryUser := `
		INSERT INTO users (email, password_hash, role, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING id, email, role, status, created_at, updated_at;
	`
	var createdUser User
	err = tx.QueryRow(ctx, queryUser, u.Email, u.PasswordHash, u.Role).Scan(
		&createdUser.ID,
		&createdUser.Email,
		&createdUser.Role,
		&createdUser.Status,
		&createdUser.CreatedAt,
		&createdUser.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique violation
			return nil, ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("erro ao inserir usuario: %w", err)
	}

	var profileData interface{}

	// 2. Inserir perfil correspondente
	if req.Role == RoleProfessional {
		sessionPrice := req.SessionPriceCents
		if sessionPrice <= 0 {
			sessionPrice = 10000 // R$ 100,00 padrao caso nao informado
		}

		queryProf := `
			INSERT INTO professional_profiles (user_id, display_name, bio, city, state, session_price_cents, branch, cref_number, crn_number)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id, display_name, bio, city, state, session_price_cents, branch, cref_number, cref_verified_at, crn_number, crn_verified_at, approved_at;
		`
		type profProfile struct {
			ID                uuid.UUID  `json:"id"`
			DisplayName       string     `json:"display_name"`
			Bio               *string    `json:"bio"`
			City              string     `json:"city"`
			State             string     `json:"state"`
			SessionPriceCents int        `json:"session_price_cents"`
			Branch            string     `json:"branch"`
			CREFNumber        *string    `json:"cref_number"`
			CREFVerifiedAt    *time.Time `json:"cref_verified_at"`
			CRNNumber         *string    `json:"crn_number"`
			CRNVerifiedAt     *time.Time `json:"crn_verified_at"`
			ApprovedAt        *string    `json:"approved_at"`
		}
		var p profProfile
		err = tx.QueryRow(ctx, queryProf,
			createdUser.ID,
			req.Name,
			req.Bio,
			req.City,
			req.State,
			sessionPrice,
			req.Branch,
			nullIfEmpty(req.CREFNumber),
			nullIfEmpty(req.CRNNumber),
		).Scan(&p.ID, &p.DisplayName, &p.Bio, &p.City, &p.State, &p.SessionPriceCents, &p.Branch, &p.CREFNumber, &p.CREFVerifiedAt, &p.CRNNumber, &p.CRNVerifiedAt, &p.ApprovedAt)
		if err != nil {
			return nil, fmt.Errorf("erro ao criar perfil do profissional autônomo: %w", err)
		}

		// Associar as especialidades selecionadas no cadastro
		for _, specID := range req.SpecialtyIDs {
			_, err = tx.Exec(ctx,
				"INSERT INTO professional_specialties (professional_id, specialty_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;",
				p.ID, specID)
			if err != nil {
				return nil, fmt.Errorf("erro ao associar especialidade do profissional: %w", err)
			}
		}

		// Associar os bairros de atuacao selecionados no cadastro
		for _, neighID := range req.NeighborhoodIDs {
			_, err = tx.Exec(ctx,
				"INSERT INTO professional_neighborhoods (professional_id, neighborhood_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;",
				p.ID, neighID)
			if err != nil {
				return nil, fmt.Errorf("erro ao associar bairro do profissional: %w", err)
			}
		}

		profileData = p
	} else {
		queryClient := `
			INSERT INTO client_profiles (user_id, name, phone, city, state)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, name, phone, city, state;
		`
		type cliProfile struct {
			ID    uuid.UUID `json:"id"`
			Name  string    `json:"name"`
			Phone *string   `json:"phone"`
			City  *string   `json:"city"`
			State *string   `json:"state"`
		}
		var c cliProfile
		err = tx.QueryRow(ctx, queryClient,
			createdUser.ID,
			req.Name,
			req.Phone,
			req.City,
			req.State,
		).Scan(&c.ID, &c.Name, &c.Phone, &c.City, &c.State)
		if err != nil {
			return nil, fmt.Errorf("erro ao criar perfil do cliente: %w", err)
		}
		profileData = c
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("erro ao comitar transacao: %w", err)
	}

	return &UserProfileResponse{
		User:    createdUser,
		Profile: profileData,
	}, nil
}

func (r *pgRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, role, status, created_at, updated_at
		FROM users
		WHERE email = $1;
	`
	var u User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.Status,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("erro ao buscar usuario por email: %w", err)
	}
	return &u, nil
}

func (r *pgRepository) GetProfileByUserID(ctx context.Context, userID uuid.UUID, role Role) (*UserProfileResponse, error) {
	queryUser := `
		SELECT id, email, role, status, created_at, updated_at
		FROM users
		WHERE id = $1;
	`
	var u User
	err := r.pool.QueryRow(ctx, queryUser, userID).Scan(
		&u.ID,
		&u.Email,
		&u.Role,
		&u.Status,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	var profile interface{}
	if u.Role == RoleProfessional {
		queryProf := `
			SELECT id, display_name, bio, city, state, session_price_cents, branch, cref_number, cref_verified_at, crn_number, crn_verified_at, approved_at
			FROM professional_profiles
			WHERE user_id = $1;
		`
		var p struct {
			ID                uuid.UUID  `json:"id"`
			DisplayName       string     `json:"display_name"`
			Bio               *string    `json:"bio"`
			City              string     `json:"city"`
			State             string     `json:"state"`
			SessionPriceCents int        `json:"session_price_cents"`
			Branch            string     `json:"branch"`
			CREFNumber        *string    `json:"cref_number"`
			CREFVerifiedAt    *time.Time `json:"cref_verified_at"`
			CRNNumber         *string    `json:"crn_number"`
			CRNVerifiedAt     *time.Time `json:"crn_verified_at"`
			ApprovedAt        *string    `json:"approved_at"`
		}
		err = r.pool.QueryRow(ctx, queryProf, userID).Scan(
			&p.ID, &p.DisplayName, &p.Bio, &p.City, &p.State, &p.SessionPriceCents, &p.Branch, &p.CREFNumber, &p.CREFVerifiedAt, &p.CRNNumber, &p.CRNVerifiedAt, &p.ApprovedAt,
		)
		if err == nil {
			profile = p
		}
	} else {
		queryCli := `
			SELECT id, name, phone, city, state
			FROM client_profiles
			WHERE user_id = $1;
		`
		var c struct {
			ID    uuid.UUID `json:"id"`
			Name  string    `json:"name"`
			Phone *string   `json:"phone"`
			City  *string   `json:"city"`
			State *string   `json:"state"`
		}
		err = r.pool.QueryRow(ctx, queryCli, userID).Scan(
			&c.ID, &c.Name, &c.Phone, &c.City, &c.State,
		)
		if err == nil {
			profile = c
		}
	}

	return &UserProfileResponse{
		User:    u,
		Profile: profile,
	}, nil
}

// SetCREFVerified grava a data da validacao apenas se o CREF ainda for o mesmo
// que foi consultado em segundo plano.
func (r *pgRepository) SetCREFVerified(ctx context.Context, userID uuid.UUID, number string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE professional_profiles
		SET cref_verified_at = NOW()
		WHERE user_id = $1 AND cref_number = $2;
	`, userID, number)
	if err != nil {
		return fmt.Errorf("erro ao gravar verificacao do CREF: %w", err)
	}
	return nil
}

// SetCRNVerified grava a data da validacao apenas se o CRN ainda for o mesmo
// que foi consultado em segundo plano.
func (r *pgRepository) SetCRNVerified(ctx context.Context, userID uuid.UUID, number string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE professional_profiles
		SET crn_verified_at = NOW()
		WHERE user_id = $1 AND crn_number = $2;
	`, userID, number)
	if err != nil {
		return fmt.Errorf("erro ao gravar verificacao do CRN: %w", err)
	}
	return nil
}

// nullIfEmpty grava NULL nas colunas de registro de conselho nao informadas
// (ex.: nutricionista nao tem CREF).
func nullIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
