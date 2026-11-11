package professionals

import (
	"time"

	"github.com/google/uuid"
)

// Ramos do profissional: decidem qual registro de conselho ele informa.
const (
	BranchPhysicalEducation = "physical_education"
	BranchNutrition         = "nutrition"
)

type Specialty struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Neighborhood struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	City  string `json:"city"`
	State string `json:"state"`
	Slug  string `json:"slug,omitempty"`
}

type Objective struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Audience struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	IsSpecific bool   `json:"is_specific"`
}

type ServiceLocation struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ProfessionalProfile struct {
	ID                 uuid.UUID           `json:"id"`
	UserID             uuid.UUID           `json:"user_id"`
	DisplayName        string              `json:"display_name"`
	Bio                *string             `json:"bio,omitempty"`
	City               string              `json:"city"`
	State              string              `json:"state"`
	Timezone           string              `json:"timezone"`
	SessionPriceCents  int                 `json:"session_price_cents"`
	CREFNumber         *string             `json:"cref_number,omitempty"`
	CREFVerifiedAt     *time.Time          `json:"cref_verified_at,omitempty"`
	Branch             string              `json:"branch"`
	CRNNumber          *string             `json:"crn_number,omitempty"`
	CRNVerifiedAt      *time.Time          `json:"crn_verified_at,omitempty"`
	AvatarURL          *string             `json:"avatar_url,omitempty"`
	RatingAvg          float64             `json:"rating"`
	ReviewsCount       int                 `json:"reviews"`
	ApprovedAt         *time.Time          `json:"approved_at,omitempty"`
	Specialties        []Specialty         `json:"specialties,omitempty"`
	Neighborhoods      []Neighborhood      `json:"neighborhoods,omitempty"`
	Objectives         []Objective         `json:"objectives,omitempty"`
	Audiences          []Audience          `json:"audiences,omitempty"`
	AudienceOther      *string             `json:"audience_other"`
	AudienceExperience *string             `json:"audience_experience"`
	Locations          []ServiceLocation   `json:"locations,omitempty"`
	Photos             []ProfessionalPhoto `json:"photos,omitempty"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
}

// CatalogResponse agrupa os catalogos usados no cadastro do profissional.
type CatalogResponse struct {
	Services   []Specialty       `json:"services"`
	Objectives []Objective       `json:"objectives"`
	Audiences  []Audience        `json:"audiences"`
	Locations  []ServiceLocation `json:"locations"`
}

type SetAudiencesRequest struct {
	AudienceIDs        []int  `json:"audience_ids"`
	AudienceOther      string `json:"audience_other"`
	AudienceExperience string `json:"audience_experience"`
}

// ProfessionalPhoto representa uma imagem da galeria do profissional.
type ProfessionalPhoto struct {
	ID             uuid.UUID `json:"id"`
	ProfessionalID uuid.UUID `json:"professional_id"`
	URL            string    `json:"url"`
	Position       int       `json:"position"`
	CreatedAt      time.Time `json:"created_at"`
}

type UpdateAvatarRequest struct {
	AvatarURL string `json:"avatar_url"`
}

type AddPhotoRequest struct {
	URL string `json:"url"`
}

type FilterParams struct {
	City           string `json:"city"`
	State          string `json:"state"`
	SpecialtyID    int    `json:"specialty_id"`
	NeighborhoodID int    `json:"neighborhood_id"`
	Limit          int    `json:"limit"`
	Offset         int    `json:"offset"`
}

type UpdateProfileRequest struct {
	DisplayName       string `json:"display_name"`
	Bio               string `json:"bio"`
	City              string `json:"city"`
	State             string `json:"state"`
	SessionPriceCents int    `json:"session_price_cents"`
	CREFNumber        string `json:"cref_number"`
	// branch nao entra aqui: o ramo e imutavel apos o cadastro (ADR 0014).
	CRNNumber string `json:"crn_number"`
}
