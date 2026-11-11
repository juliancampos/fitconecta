package professionals

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProfessionalNotFound = errors.New("perfil do profissional nao encontrado")
)

type Repository interface {
	List(ctx context.Context, f FilterParams) ([]ProfessionalProfile, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ProfessionalProfile, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*ProfessionalProfile, error)
	Update(ctx context.Context, userID uuid.UUID, req *UpdateProfileRequest, crefVerifiedAt, crnVerifiedAt *time.Time) (*ProfessionalProfile, error)
	SetCREFVerified(ctx context.Context, userID uuid.UUID, number string, verifiedAt *time.Time) error
	SetCRNVerified(ctx context.Context, userID uuid.UUID, number string, verifiedAt *time.Time) error
	UpdateAvatar(ctx context.Context, userID uuid.UUID, avatarURL string) (*ProfessionalProfile, error)
	ListSpecialties(ctx context.Context) ([]Specialty, error)
	SetSpecialties(ctx context.Context, profID uuid.UUID, specialtyIDs []int) error

	ListObjectives(ctx context.Context) ([]Objective, error)
	GetObjectives(ctx context.Context, profID uuid.UUID) ([]Objective, error)
	SetObjectives(ctx context.Context, profID uuid.UUID, objectiveIDs []int) error

	ListAudiences(ctx context.Context) ([]Audience, error)
	GetAudiences(ctx context.Context, profID uuid.UUID) ([]Audience, error)
	SetAudiences(ctx context.Context, profID uuid.UUID, req *SetAudiencesRequest) error

	ListServiceLocations(ctx context.Context) ([]ServiceLocation, error)
	GetLocations(ctx context.Context, profID uuid.UUID) ([]ServiceLocation, error)
	SetLocations(ctx context.Context, profID uuid.UUID, locationIDs []int) error

	ListNeighborhoods(ctx context.Context, city, state string) ([]Neighborhood, error)
	GetNeighborhoods(ctx context.Context, profID uuid.UUID) ([]Neighborhood, error)
	SetNeighborhoods(ctx context.Context, profID uuid.UUID, neighborhoodIDs []int) error

	ListPhotos(ctx context.Context, profID uuid.UUID) ([]ProfessionalPhoto, error)
	AddPhoto(ctx context.Context, profID uuid.UUID, url string) (*ProfessionalPhoto, error)
	DeletePhoto(ctx context.Context, profID uuid.UUID, photoID uuid.UUID) error
}

type pgRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

const profileColumns = `id, user_id, display_name, bio, city, state, timezone,
	session_price_cents, cref_number, cref_verified_at, branch, crn_number, crn_verified_at, avatar_url, rating_avg, reviews_count, approved_at, created_at, updated_at,
	audience_other, audience_experience`

func scanProfile(row pgx.Row) (*ProfessionalProfile, error) {
	var p ProfessionalProfile
	err := row.Scan(
		&p.ID, &p.UserID, &p.DisplayName, &p.Bio, &p.City, &p.State, &p.Timezone,
		&p.SessionPriceCents, &p.CREFNumber, &p.CREFVerifiedAt, &p.Branch, &p.CRNNumber, &p.CRNVerifiedAt, &p.AvatarURL, &p.RatingAvg, &p.ReviewsCount, &p.ApprovedAt, &p.CreatedAt, &p.UpdatedAt,
		&p.AudienceOther, &p.AudienceExperience,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *pgRepository) List(ctx context.Context, f FilterParams) ([]ProfessionalProfile, error) {
	if f.Limit <= 0 || f.Limit > 50 {
		f.Limit = 20
	}

	query := `
		SELECT DISTINCT p.id, p.user_id, p.display_name, p.bio, p.city, p.state, p.timezone,
		       p.session_price_cents, p.cref_number, p.cref_verified_at, p.branch, p.crn_number, p.crn_verified_at, p.avatar_url, p.rating_avg, p.reviews_count, p.approved_at, p.created_at, p.updated_at
		FROM professional_profiles p
		JOIN users u ON u.id = p.user_id AND u.status = 'active'
		LEFT JOIN professional_specialties ps ON ps.professional_id = p.id
		LEFT JOIN professional_neighborhoods pn ON pn.professional_id = p.id
		WHERE ($1 = '' OR LOWER(p.city) = LOWER($1))
		  AND ($2 = '' OR UPPER(p.state) = UPPER($2))
		  AND ($3 = 0 OR ps.specialty_id = $3)
		  AND ($4 = 0 OR pn.neighborhood_id = $4)
		  -- Perfil incompleto (cadastro minimo, ADR 0016) nao aparece na vitrine.
		  AND p.city <> '' AND p.state <> ''
		ORDER BY p.display_name ASC
		LIMIT $5 OFFSET $6;
	`

	rows, err := r.pool.Query(ctx, query, f.City, f.State, f.SpecialtyID, f.NeighborhoodID, f.Limit, f.Offset)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar profissionais: %w", err)
	}
	defer rows.Close()

	var results []ProfessionalProfile
	for rows.Next() {
		var p ProfessionalProfile
		err := rows.Scan(
			&p.ID, &p.UserID, &p.DisplayName, &p.Bio, &p.City, &p.State, &p.Timezone,
			&p.SessionPriceCents, &p.CREFNumber, &p.CREFVerifiedAt, &p.Branch, &p.CRNNumber, &p.CRNVerifiedAt, &p.AvatarURL, &p.RatingAvg, &p.ReviewsCount, &p.ApprovedAt, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao mapear profissional: %w", err)
		}
		results = append(results, p)
	}

	// Carregar especialidades e bairros para cada profissional retornado
	for i := range results {
		specs, err := r.getSpecialtiesByProfID(ctx, results[i].ID)
		if err == nil {
			results[i].Specialties = specs
		}
		neigh, err := r.GetNeighborhoods(ctx, results[i].ID)
		if err == nil {
			results[i].Neighborhoods = neigh
		}
	}

	return results, nil
}

func (r *pgRepository) GetByID(ctx context.Context, id uuid.UUID) (*ProfessionalProfile, error) {
	query := `SELECT p.id, p.user_id, p.display_name, p.bio, p.city, p.state, p.timezone,
		p.session_price_cents, p.cref_number, p.cref_verified_at, p.branch, p.crn_number, p.crn_verified_at, p.avatar_url, p.rating_avg, p.reviews_count, p.approved_at, p.created_at, p.updated_at,
		p.audience_other, p.audience_experience
		FROM professional_profiles p
		JOIN users u ON u.id = p.user_id AND u.status = 'active'
		WHERE p.id = $1;`

	p, err := scanProfile(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProfessionalNotFound
		}
		return nil, fmt.Errorf("erro ao obter perfil do profissional: %w", err)
	}

	specs, _ := r.getSpecialtiesByProfID(ctx, p.ID)
	p.Specialties = specs
	neigh, _ := r.GetNeighborhoods(ctx, p.ID)
	p.Neighborhoods = neigh
	photos, _ := r.ListPhotos(ctx, p.ID)
	p.Photos = photos
	p.Objectives, _ = r.GetObjectives(ctx, p.ID)
	p.Audiences, _ = r.GetAudiences(ctx, p.ID)
	p.Locations, _ = r.GetLocations(ctx, p.ID)

	return p, nil
}

func (r *pgRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*ProfessionalProfile, error) {
	query := `SELECT ` + profileColumns + ` FROM professional_profiles WHERE user_id = $1;`

	p, err := scanProfile(r.pool.QueryRow(ctx, query, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProfessionalNotFound
		}
		return nil, err
	}

	specs, _ := r.getSpecialtiesByProfID(ctx, p.ID)
	p.Specialties = specs
	neigh, _ := r.GetNeighborhoods(ctx, p.ID)
	p.Neighborhoods = neigh
	photos, _ := r.ListPhotos(ctx, p.ID)
	p.Photos = photos
	p.Objectives, _ = r.GetObjectives(ctx, p.ID)
	p.Audiences, _ = r.GetAudiences(ctx, p.ID)
	p.Locations, _ = r.GetLocations(ctx, p.ID)

	return p, nil
}

func (r *pgRepository) Update(ctx context.Context, userID uuid.UUID, req *UpdateProfileRequest, crefVerifiedAt, crnVerifiedAt *time.Time) (*ProfessionalProfile, error) {
	query := `
		UPDATE professional_profiles
		SET display_name = $1, bio = $2, city = $3, state = $4, session_price_cents = $5,
		    cref_number = $6, crn_number = $8,
		    cref_verified_at = CASE WHEN cref_number IS DISTINCT FROM $6 THEN $7 ELSE cref_verified_at END,
		    crn_verified_at = CASE WHEN crn_number IS DISTINCT FROM $8 THEN $9 ELSE crn_verified_at END,
		    updated_at = NOW()
		WHERE user_id = $10
		RETURNING ` + profileColumns + `;
	`
	p, err := scanProfile(r.pool.QueryRow(ctx, query,
		req.DisplayName, req.Bio, req.City, req.State,
		req.SessionPriceCents, req.CREFNumber, crefVerifiedAt,
		nullIfEmpty(req.CRNNumber), crnVerifiedAt, userID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProfessionalNotFound
		}
		return nil, fmt.Errorf("erro ao atualizar perfil do profissional: %w", err)
	}

	specs, _ := r.getSpecialtiesByProfID(ctx, p.ID)
	p.Specialties = specs
	neigh, _ := r.GetNeighborhoods(ctx, p.ID)
	p.Neighborhoods = neigh
	photos, _ := r.ListPhotos(ctx, p.ID)
	p.Photos = photos
	p.Objectives, _ = r.GetObjectives(ctx, p.ID)
	p.Audiences, _ = r.GetAudiences(ctx, p.ID)
	p.Locations, _ = r.GetLocations(ctx, p.ID)

	return p, nil
}

// SetCREFVerified grava a data da validacao apenas se o CREF ainda for o mesmo
// que foi consultado (evita marcar um numero editado no meio do caminho).
func (r *pgRepository) SetCREFVerified(ctx context.Context, userID uuid.UUID, number string, verifiedAt *time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE professional_profiles
		SET cref_verified_at = $1
		WHERE user_id = $2 AND cref_number = $3;
	`, verifiedAt, userID, number)
	if err != nil {
		return fmt.Errorf("erro ao gravar verificacao do CREF: %w", err)
	}
	return nil
}

// SetCRNVerified grava a data da validacao apenas se o CRN ainda for o mesmo
// que foi consultado (evita marcar um numero editado no meio do caminho).
func (r *pgRepository) SetCRNVerified(ctx context.Context, userID uuid.UUID, number string, verifiedAt *time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE professional_profiles
		SET crn_verified_at = $1
		WHERE user_id = $2 AND crn_number = $3;
	`, verifiedAt, userID, number)
	if err != nil {
		return fmt.Errorf("erro ao gravar verificacao do CRN: %w", err)
	}
	return nil
}

func (r *pgRepository) UpdateAvatar(ctx context.Context, userID uuid.UUID, avatarURL string) (*ProfessionalProfile, error) {
	query := `
		UPDATE professional_profiles
		SET avatar_url = $1, updated_at = NOW()
		WHERE user_id = $2
		RETURNING ` + profileColumns + `;
	`
	p, err := scanProfile(r.pool.QueryRow(ctx, query, nullIfEmpty(avatarURL), userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProfessionalNotFound
		}
		return nil, fmt.Errorf("erro ao atualizar avatar do profissional: %w", err)
	}
	return p, nil
}

func (r *pgRepository) ListSpecialties(ctx context.Context) ([]Specialty, error) {
	query := `SELECT id, name FROM specialties ORDER BY name ASC;`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Specialty
	for rows.Next() {
		var s Specialty
		if err := rows.Scan(&s.ID, &s.Name); err == nil {
			list = append(list, s)
		}
	}
	return list, nil
}

func (r *pgRepository) SetSpecialties(ctx context.Context, profID uuid.UUID, specialtyIDs []int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Remove associacoes antigas
	_, err = tx.Exec(ctx, "DELETE FROM professional_specialties WHERE professional_id = $1;", profID)
	if err != nil {
		return err
	}

	// Insere novas
	for _, specID := range specialtyIDs {
		_, err = tx.Exec(ctx,
			"INSERT INTO professional_specialties (professional_id, specialty_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;",
			profID, specID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *pgRepository) ListObjectives(ctx context.Context) ([]Objective, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, name FROM objectives ORDER BY name ASC;")
	if err != nil {
		return nil, fmt.Errorf("erro ao listar objetivos: %w", err)
	}
	defer rows.Close()

	var list []Objective
	for rows.Next() {
		var o Objective
		if err := rows.Scan(&o.ID, &o.Name); err == nil {
			list = append(list, o)
		}
	}
	return list, nil
}

func (r *pgRepository) GetObjectives(ctx context.Context, profID uuid.UUID) ([]Objective, error) {
	query := `
		SELECT o.id, o.name
		FROM objectives o
		INNER JOIN professional_objectives po ON po.objective_id = o.id
		WHERE po.professional_id = $1
		ORDER BY o.name ASC;
	`
	rows, err := r.pool.Query(ctx, query, profID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Objective
	for rows.Next() {
		var o Objective
		if err := rows.Scan(&o.ID, &o.Name); err == nil {
			list = append(list, o)
		}
	}
	return list, nil
}

func (r *pgRepository) SetObjectives(ctx context.Context, profID uuid.UUID, objectiveIDs []int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM professional_objectives WHERE professional_id = $1;", profID)
	if err != nil {
		return err
	}

	for _, id := range objectiveIDs {
		_, err = tx.Exec(ctx,
			"INSERT INTO professional_objectives (professional_id, objective_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;",
			profID, id)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *pgRepository) ListAudiences(ctx context.Context) ([]Audience, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, name, is_specific FROM audiences ORDER BY name ASC;")
	if err != nil {
		return nil, fmt.Errorf("erro ao listar publicos: %w", err)
	}
	defer rows.Close()

	var list []Audience
	for rows.Next() {
		var a Audience
		if err := rows.Scan(&a.ID, &a.Name, &a.IsSpecific); err == nil {
			list = append(list, a)
		}
	}
	return list, nil
}

func (r *pgRepository) GetAudiences(ctx context.Context, profID uuid.UUID) ([]Audience, error) {
	query := `
		SELECT a.id, a.name, a.is_specific
		FROM audiences a
		INNER JOIN professional_audiences pa ON pa.audience_id = a.id
		WHERE pa.professional_id = $1
		ORDER BY a.name ASC;
	`
	rows, err := r.pool.Query(ctx, query, profID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Audience
	for rows.Next() {
		var a Audience
		if err := rows.Scan(&a.ID, &a.Name, &a.IsSpecific); err == nil {
			list = append(list, a)
		}
	}
	return list, nil
}

func (r *pgRepository) SetAudiences(ctx context.Context, profID uuid.UUID, req *SetAudiencesRequest) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE professional_profiles
		SET audience_other = $1, audience_experience = $2, updated_at = NOW()
		WHERE id = $3;`,
		nullIfEmpty(req.AudienceOther), nullIfEmpty(req.AudienceExperience), profID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "DELETE FROM professional_audiences WHERE professional_id = $1;", profID)
	if err != nil {
		return err
	}

	for _, id := range req.AudienceIDs {
		_, err = tx.Exec(ctx,
			"INSERT INTO professional_audiences (professional_id, audience_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;",
			profID, id)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *pgRepository) ListServiceLocations(ctx context.Context) ([]ServiceLocation, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, name FROM service_locations ORDER BY name ASC;")
	if err != nil {
		return nil, fmt.Errorf("erro ao listar locais de atendimento: %w", err)
	}
	defer rows.Close()

	var list []ServiceLocation
	for rows.Next() {
		var l ServiceLocation
		if err := rows.Scan(&l.ID, &l.Name); err == nil {
			list = append(list, l)
		}
	}
	return list, nil
}

func (r *pgRepository) GetLocations(ctx context.Context, profID uuid.UUID) ([]ServiceLocation, error) {
	query := `
		SELECT l.id, l.name
		FROM service_locations l
		INNER JOIN professional_locations pl ON pl.location_id = l.id
		WHERE pl.professional_id = $1
		ORDER BY l.name ASC;
	`
	rows, err := r.pool.Query(ctx, query, profID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ServiceLocation
	for rows.Next() {
		var l ServiceLocation
		if err := rows.Scan(&l.ID, &l.Name); err == nil {
			list = append(list, l)
		}
	}
	return list, nil
}

func (r *pgRepository) SetLocations(ctx context.Context, profID uuid.UUID, locationIDs []int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM professional_locations WHERE professional_id = $1;", profID)
	if err != nil {
		return err
	}

	for _, id := range locationIDs {
		_, err = tx.Exec(ctx,
			"INSERT INTO professional_locations (professional_id, location_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;",
			profID, id)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *pgRepository) ListPhotos(ctx context.Context, profID uuid.UUID) ([]ProfessionalPhoto, error) {
	query := `
		SELECT id, professional_id, url, position, created_at
		FROM professional_photos
		WHERE professional_id = $1
		ORDER BY position ASC, created_at ASC;
	`
	rows, err := r.pool.Query(ctx, query, profID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ProfessionalPhoto
	for rows.Next() {
		var p ProfessionalPhoto
		if err := rows.Scan(&p.ID, &p.ProfessionalID, &p.URL, &p.Position, &p.CreatedAt); err == nil {
			list = append(list, p)
		}
	}
	return list, nil
}

func (r *pgRepository) AddPhoto(ctx context.Context, profID uuid.UUID, url string) (*ProfessionalPhoto, error) {
	query := `
		INSERT INTO professional_photos (professional_id, url, position)
		VALUES ($1, $2, COALESCE((SELECT MAX(position) + 1 FROM professional_photos WHERE professional_id = $1), 0))
		RETURNING id, professional_id, url, position, created_at;
	`
	var p ProfessionalPhoto
	err := r.pool.QueryRow(ctx, query, profID, url).Scan(
		&p.ID, &p.ProfessionalID, &p.URL, &p.Position, &p.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao adicionar foto: %w", err)
	}
	return &p, nil
}

func (r *pgRepository) DeletePhoto(ctx context.Context, profID uuid.UUID, photoID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM professional_photos WHERE id = $1 AND professional_id = $2;",
		photoID, profID)
	if err != nil {
		return fmt.Errorf("erro ao remover foto: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrProfessionalNotFound
	}
	return nil
}

func (r *pgRepository) getSpecialtiesByProfID(ctx context.Context, profID uuid.UUID) ([]Specialty, error) {
	query := `
		SELECT s.id, s.name
		FROM specialties s
		INNER JOIN professional_specialties ps ON ps.specialty_id = s.id
		WHERE ps.professional_id = $1
		ORDER BY s.name ASC;
	`
	rows, err := r.pool.Query(ctx, query, profID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Specialty
	for rows.Next() {
		var s Specialty
		if err := rows.Scan(&s.ID, &s.Name); err == nil {
			list = append(list, s)
		}
	}
	return list, nil
}

func (r *pgRepository) GetNeighborhoods(ctx context.Context, profID uuid.UUID) ([]Neighborhood, error) {
	query := `
		SELECT n.id, n.name, n.city, n.state, n.slug
		FROM neighborhoods n
		INNER JOIN professional_neighborhoods pn ON pn.neighborhood_id = n.id
		WHERE pn.professional_id = $1
		ORDER BY n.name ASC;
	`
	rows, err := r.pool.Query(ctx, query, profID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Neighborhood
	for rows.Next() {
		var n Neighborhood
		if err := rows.Scan(&n.ID, &n.Name, &n.City, &n.State, &n.Slug); err == nil {
			list = append(list, n)
		}
	}
	return list, nil
}

func (r *pgRepository) ListNeighborhoods(ctx context.Context, city, state string) ([]Neighborhood, error) {
	query := `
		SELECT id, name, city, state, slug
		FROM neighborhoods
		WHERE is_active = TRUE
		  AND ($1 = '' OR LOWER(city) = LOWER($1))
		  AND ($2 = '' OR UPPER(state) = UPPER($2))
		ORDER BY city ASC, name ASC;
	`
	rows, err := r.pool.Query(ctx, query, city, state)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Neighborhood
	for rows.Next() {
		var n Neighborhood
		if err := rows.Scan(&n.ID, &n.Name, &n.City, &n.State, &n.Slug); err == nil {
			list = append(list, n)
		}
	}
	return list, nil
}

func (r *pgRepository) SetNeighborhoods(ctx context.Context, profID uuid.UUID, neighborhoodIDs []int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM professional_neighborhoods WHERE professional_id = $1;", profID)
	if err != nil {
		return err
	}

	for _, id := range neighborhoodIDs {
		_, err = tx.Exec(ctx,
			"INSERT INTO professional_neighborhoods (professional_id, neighborhood_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;",
			profID, id)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func nullIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
