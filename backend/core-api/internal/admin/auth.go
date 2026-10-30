package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"fitconecta/core-api/internal/auth"
	"fitconecta/core-api/internal/platform/middleware"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Login(ctx context.Context, email, password string) (*AuthResponse, error)
}

type AuthResponse struct {
	Token     string    `json:"token"`
	User      auth.User `json:"user"`
	ExpiresIn int64     `json:"expires_in"`
}

type authService struct {
	repo               auth.Repository
	jwtSecret          string
	jwtExpirationHours int
}

func NewAuthService(repo auth.Repository, jwtSecret string, jwtExpirationHours int) AuthService {
	if jwtExpirationHours <= 0 {
		jwtExpirationHours = 8
	}
	return &authService{
		repo:               repo,
		jwtSecret:          jwtSecret,
		jwtExpirationHours: jwtExpirationHours,
	}
}

func (s *authService) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return nil, ErrInvalidAdminLogin
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return nil, ErrInvalidAdminLogin
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidAdminLogin
	}
	if user.Role != auth.RoleAdmin || user.Status != "active" {
		return nil, ErrInvalidAdminLogin
	}

	expiration := time.Duration(s.jwtExpirationHours) * time.Hour
	expiresAt := time.Now().Add(expiration)
	claims := &middleware.JWTClaims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Role:   string(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			Audience:  jwt.ClaimStrings{middleware.AdminAudience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "fitconecta-admin",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token:     tokenString,
		User:      *user,
		ExpiresIn: int64(expiration.Seconds()),
	}, nil
}
