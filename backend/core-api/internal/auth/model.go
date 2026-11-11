package auth

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleClient       Role = "client"
	RoleProfessional Role = "professional"
	RoleAdmin        Role = "admin"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     Role   `json:"role"` // "client" ou "professional"

	// Dados do perfil (preenchidos de acordo com o papel)
	Name              string `json:"name"`                          // Nome exibido ou do cliente
	City              string `json:"city"`                          // Cidade
	State             string `json:"state"`                         // UF (ex: MG, SP)
	Phone             string `json:"phone,omitempty"`               // Telefone do cliente
	Bio               string `json:"bio,omitempty"`                 // Bio do profissional
	SessionPriceCents int    `json:"session_price_cents,omitempty"` // Valor da sessao em centavos
	Branch            string `json:"branch,omitempty"`              // Ramo do profissional: physical_education (padrao) ou nutrition
	CREFNumber        string `json:"cref_number,omitempty"`         // Registro CREF (obrigatorio quando ramo = physical_education)
	CRNNumber         string `json:"crn_number,omitempty"`          // Registro CRN (obrigatorio quando ramo = nutrition)
	SpecialtyIDs      []int  `json:"specialty_ids,omitempty"`       // Especialidades do profissional
	NeighborhoodIDs   []int  `json:"neighborhood_ids,omitempty"`    // Bairros de atuacao do profissional
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token     string      `json:"token"`
	User      User        `json:"user"`
	Profile   interface{} `json:"profile,omitempty"`
	ExpiresIn int64       `json:"expires_in"` // segundos
}

type UserProfileResponse struct {
	User    User        `json:"user"`
	Profile interface{} `json:"profile,omitempty"`
}
