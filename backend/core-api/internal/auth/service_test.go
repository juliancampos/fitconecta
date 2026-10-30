package auth

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type mockAuthRepo struct {
	users map[string]*User
}

func (m *mockAuthRepo) CreateUserWithProfile(ctx context.Context, u *User, req *RegisterRequest) (*UserProfileResponse, error) {
	if _, exists := m.users[u.Email]; exists {
		return nil, ErrEmailAlreadyExists
	}
	u.ID = uuid.New()
	m.users[u.Email] = u
	return &UserProfileResponse{
		User:    *u,
		Profile: map[string]string{"name": req.Name},
	}, nil
}

func (m *mockAuthRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (m *mockAuthRepo) GetProfileByUserID(ctx context.Context, userID uuid.UUID, role Role) (*UserProfileResponse, error) {
	for _, u := range m.users {
		if u.ID == userID {
			return &UserProfileResponse{
				User:    *u,
				Profile: map[string]string{"user_id": userID.String()},
			}, nil
		}
	}
	return nil, ErrUserNotFound
}

func TestRegisterAndLoginFlow(t *testing.T) {
	repo := &mockAuthRepo{users: make(map[string]*User)}
	svc := NewService(repo, "test_jwt_secret_with_32_characters_minimum", 24)

	// 1. Registro valido
	regReq := &RegisterRequest{
		Email:    "personal@fitconecta.com",
		Password: "password123",
		Role:     RoleProfessional,
		Name:     "Carlos Personal",
		City:     "Belo Horizonte",
		State:    "MG",
	}

	authResp, err := svc.Register(context.Background(), regReq)
	if err != nil {
		t.Fatalf("esperava sucesso no registro, obteve: %v", err)
	}

	if authResp.Token == "" {
		t.Error("token JWT gerado esta vazio")
	}

	if authResp.User.Email != "personal@fitconecta.com" {
		t.Errorf("email retornado diverge: %s", authResp.User.Email)
	}

	// 2. Tentativa de cadastro duplicado
	_, err = svc.Register(context.Background(), regReq)
	if err == nil || !strings.Contains(err.Error(), "ja esta cadastrado") {
		t.Errorf("esperava erro de duplicidade de email, obteve: %v", err)
	}

	// 3. Login com senha correta
	loginResp, err := svc.Login(context.Background(), &LoginRequest{
		Email:    "personal@fitconecta.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("esperava sucesso no login, obteve: %v", err)
	}

	if loginResp.Token == "" {
		t.Error("token de login vazio")
	}

	// 4. Login com senha incorreta
	_, err = svc.Login(context.Background(), &LoginRequest{
		Email:    "personal@fitconecta.com",
		Password: "wrong_password",
	})
	if err != ErrInvalidCredentials {
		t.Errorf("esperava ErrInvalidCredentials, obteve: %v", err)
	}
}

func TestPasswordHashingSecurity(t *testing.T) {
	pwd := "senhaSegura123"
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("erro ao gerar hash: %v", err)
	}

	if string(hash) == pwd {
		t.Fatal("o hash nao pode ser igual a senha plana")
	}

	if err := bcrypt.CompareHashAndPassword(hash, []byte(pwd)); err != nil {
		t.Errorf("comparacao da senha correta falhou: %v", err)
	}
}
