package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"fitconecta/core-api/internal/cref"
	"fitconecta/core-api/internal/crn"
	"fitconecta/core-api/internal/platform/middleware"
	"fitconecta/core-api/internal/professionals"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("credenciais invalidas")
	ErrInvalidInput       = errors.New("dados de entrada invalidos")
)

type Service interface {
	Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error)
	GetMe(ctx context.Context, userID uuid.UUID, role Role) (*UserProfileResponse, error)
}

type service struct {
	repo               Repository
	jwtSecret          string
	jwtExpirationHours int
	crefValidator      cref.Validator
	crnValidator       crn.Validator
}

func NewService(repo Repository, jwtSecret string, jwtExpirationHours int, crefValidator cref.Validator, crnValidator crn.Validator) Service {
	return &service{
		repo:               repo,
		jwtSecret:          jwtSecret,
		jwtExpirationHours: jwtExpirationHours,
		crefValidator:      crefValidator,
		crnValidator:       crnValidator,
	}
}

func (s *service) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	// Validacoes basicas
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, fmt.Errorf("%w: e-mail invalido", ErrInvalidInput)
	}
	if len(req.Password) < 8 {
		return nil, fmt.Errorf("%w: a senha deve conter pelo menos 8 caracteres", ErrInvalidInput)
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("%w: o nome e obrigatorio", ErrInvalidInput)
	}
	if req.Role != RoleClient && req.Role != RoleProfessional {
		return nil, fmt.Errorf("%w: papel de usuario deve ser 'client' ou 'professional'", ErrInvalidInput)
	}
	// O ramo decide qual registro do conselho o profissional informa; sem
	// branch assume EF (retrocompatibilidade). A consulta ao conselho e
	// apenas informativa: falha na validacao nao impede o cadastro (ADRs
	// 0013 e 0014).
	if req.Role == RoleProfessional {
		req.Branch = strings.ToLower(strings.TrimSpace(req.Branch))
		if req.Branch == "" {
			req.Branch = professionals.BranchPhysicalEducation
		}
		req.CREFNumber = strings.TrimSpace(req.CREFNumber)
		req.CRNNumber = strings.TrimSpace(req.CRNNumber)
		switch req.Branch {
		case professionals.BranchNutrition:
			// Nutricionista nao tem CREF: o campo e ignorado (gravado NULL).
			req.CREFNumber = ""
			if req.CRNNumber == "" {
				return nil, fmt.Errorf("%w: informe o numero do CRN", ErrInvalidInput)
			}
		case professionals.BranchPhysicalEducation:
			if req.CREFNumber == "" {
				return nil, fmt.Errorf("%w: informe o numero do CREF", ErrInvalidInput)
			}
		default:
			return nil, fmt.Errorf("%w: ramo deve ser 'physical_education' ou 'nutrition'", ErrInvalidInput)
		}
	}

	// Criptografar senha
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("falha ao criptografar senha: %w", err)
	}

	userToCreate := &User{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         req.Role,
	}

	userProfile, err := s.repo.CreateUserWithProfile(ctx, userToCreate, req)
	if err != nil {
		return nil, err
	}

	// A verificacao roda em segundo plano: o cadastro responde antes do
	// conselho. Cada ramo valida o seu registro (ADR 0014).
	if req.Role == RoleProfessional {
		if req.Branch == professionals.BranchNutrition {
			go s.verifyCRN(context.WithoutCancel(ctx), userProfile.User.ID, req.CRNNumber)
		} else {
			go s.verifyCREF(context.WithoutCancel(ctx), userProfile.User.ID, req.CREFNumber)
		}
	}

	// Gerar JWT
	tokenString, expiresIn, err := s.generateToken(userProfile.User.ID, userProfile.User.Email, string(userProfile.User.Role))
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar token de acesso: %w", err)
	}

	return &AuthResponse{
		Token:     tokenString,
		User:      userProfile.User,
		Profile:   userProfile.Profile,
		ExpiresIn: expiresIn,
	}, nil
}

func (s *service) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// O login publico nunca aceita administradores e contas inativas/pendentes.
	// A mensagem permanece generica para nao revelar o estado da conta.
	if user.Role == RoleAdmin || user.Status == "inactive" || user.Status == "pending" {
		return nil, ErrInvalidCredentials
	}

	profileResp, err := s.repo.GetProfileByUserID(ctx, user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	tokenString, expiresIn, err := s.generateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar token de acesso: %w", err)
	}

	return &AuthResponse{
		Token:     tokenString,
		User:      *user,
		Profile:   profileResp.Profile,
		ExpiresIn: expiresIn,
	}, nil
}

func (s *service) GetMe(ctx context.Context, userID uuid.UUID, role Role) (*UserProfileResponse, error) {
	return s.repo.GetProfileByUserID(ctx, userID, role)
}

func (s *service) verifyCREF(ctx context.Context, userID uuid.UUID, number string) {
	if _, err := s.crefValidator.Validate(ctx, number); err != nil {
		return
	}
	_ = s.repo.SetCREFVerified(ctx, userID, number)
}

func (s *service) verifyCRN(ctx context.Context, userID uuid.UUID, number string) {
	if _, err := s.crnValidator.Validate(ctx, number); err != nil {
		return
	}
	_ = s.repo.SetCRNVerified(ctx, userID, number)
}

func (s *service) generateToken(userID uuid.UUID, email, role string) (string, int64, error) {
	expiration := time.Duration(s.jwtExpirationHours) * time.Hour
	expiresAt := time.Now().Add(expiration)

	claims := &middleware.JWTClaims{
		UserID: userID.String(),
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{"fitconecta-web"},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "fitconecta-plus",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", 0, err
	}

	return tokenString, int64(expiration.Seconds()), nil
}
