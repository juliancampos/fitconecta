package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"fitconecta/core-api/internal/cref"
	"fitconecta/core-api/internal/crn"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type stubCREFValidator struct {
	err error
}

func (s stubCREFValidator) Validate(ctx context.Context, number string) (*cref.Profile, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &cref.Profile{Number: number, Status: "ATIVO"}, nil
}

type stubCRNValidator struct {
	err error
}

func (s stubCRNValidator) Validate(ctx context.Context, number string) (*crn.Profile, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &crn.Profile{Number: number, Name: "NUTRICIONISTA TESTE"}, nil
}

type mockAuthRepo struct {
	users     map[string]*User
	verified  chan string
	crnChecks chan string
	last      *RegisterRequest
}

func (m *mockAuthRepo) SetCREFVerified(ctx context.Context, userID uuid.UUID, number string) error {
	if m.verified != nil {
		m.verified <- number
	}
	return nil
}

func (m *mockAuthRepo) SetCRNVerified(ctx context.Context, userID uuid.UUID, number string) error {
	if m.crnChecks != nil {
		m.crnChecks <- number
	}
	return nil
}

func (m *mockAuthRepo) CreateUserWithProfile(ctx context.Context, u *User, req *RegisterRequest) (*UserProfileResponse, error) {
	if _, exists := m.users[u.Email]; exists {
		return nil, ErrEmailAlreadyExists
	}
	u.ID = uuid.New()
	m.users[u.Email] = u
	m.last = req
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
	svc := NewService(repo, "test_jwt_secret_with_32_characters_minimum", 24, stubCREFValidator{}, stubCRNValidator{})

	// 1. Registro valido
	regReq := &RegisterRequest{
		Email:      "personal@fitconecta.com",
		Password:   "password123",
		Role:       RoleProfessional,
		Name:       "Carlos Personal",
		City:       "Belo Horizonte",
		State:      "MG",
		CREFNumber: "MG-012345",
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

func TestRegisterVerifiesCREFInBackground(t *testing.T) {
	verified := make(chan string, 1)
	repo := &mockAuthRepo{users: make(map[string]*User), verified: verified}
	svc := NewService(repo, "test_jwt_secret_with_32_characters_minimum", 24, stubCREFValidator{}, stubCRNValidator{})

	_, err := svc.Register(context.Background(), &RegisterRequest{
		Email:      "personal@fitconecta.com",
		Password:   "password123",
		Role:       RoleProfessional,
		Name:       "Carlos Personal",
		City:       "Belo Horizonte",
		State:      "MG",
		CREFNumber: "MG-012345",
	})
	if err != nil {
		t.Fatalf("esperava sucesso no registro, obteve: %v", err)
	}

	select {
	case got := <-verified:
		if got != "MG-012345" {
			t.Errorf("CREF verificado = %q, want %q", got, "MG-012345")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("validacao em segundo plano nao gravou o CREF")
	}
}

func TestRegisterDoesNotBlockOnCREFCheck(t *testing.T) {
	repo := &mockAuthRepo{users: make(map[string]*User)}
	svc := NewService(repo, "test_jwt_secret_with_32_characters_minimum", 24, stubCREFValidator{err: cref.ErrUnavailable}, stubCRNValidator{})

	_, err := svc.Register(context.Background(), &RegisterRequest{
		Email:      "sem.cref@fitconecta.com",
		Password:   "password123",
		Role:       RoleProfessional,
		Name:       "Sem CREF",
		City:       "Belo Horizonte",
		State:      "MG",
		CREFNumber: "000000",
	})
	if err != nil {
		t.Fatalf("cadastro nao deveria falhar por CREF indisponivel: %v", err)
	}
	if _, exists := repo.users["sem.cref@fitconecta.com"]; !exists {
		t.Error("usuario deveria ter sido criado")
	}
}

func TestRegisterNutritionBranch(t *testing.T) {
	tests := []struct {
		name    string
		branch  string
		crn     string
		cref    string
		wantErr string
	}{
		{name: "nutricao sem CRN", branch: "nutrition", crn: "", cref: "", wantErr: "informe o numero do CRN"},
		{name: "ramo invalido", branch: "physiotherapy", crn: "39104", cref: "", wantErr: "ramo deve ser 'physical_education' ou 'nutrition'"},
		{name: "EF sem CREF", branch: "physical_education", crn: "", cref: "", wantErr: "informe o numero do CREF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAuthRepo{users: make(map[string]*User)}
			svc := NewService(repo, "test_jwt_secret_with_32_characters_minimum", 24, stubCREFValidator{}, stubCRNValidator{})

			_, err := svc.Register(context.Background(), &RegisterRequest{
				Email:      "nutri@fitconecta.com",
				Password:   "password123",
				Role:       RoleProfessional,
				Name:       "Nutri Teste",
				City:       "Belo Horizonte",
				State:      "MG",
				Branch:     tt.branch,
				CRNNumber:  tt.crn,
				CREFNumber: tt.cref,
			})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("esperava erro %q, obteve: %v", tt.wantErr, err)
			}
		})
	}
}

func TestRegisterNutritionWithCRN(t *testing.T) {
	repo := &mockAuthRepo{users: make(map[string]*User)}
	svc := NewService(repo, "test_jwt_secret_with_32_characters_minimum", 24, stubCREFValidator{}, stubCRNValidator{})

	// cref preenchido e ignorado: nutricionista grava crn e cref NULL.
	_, err := svc.Register(context.Background(), &RegisterRequest{
		Email:      "nutri@fitconecta.com",
		Password:   "password123",
		Role:       RoleProfessional,
		Name:       "Nutri Teste",
		City:       "Belo Horizonte",
		State:      "MG",
		Branch:     "nutrition",
		CRNNumber:  "CRN3 51284",
		CREFNumber: "MG-012345",
	})
	if err != nil {
		t.Fatalf("nutricao com CRN deveria cadastrar sem exigir CREF, obteve: %v", err)
	}
	if repo.last.CREFNumber != "" {
		t.Errorf("cref_number deveria ser zerado para nutricao, obteve %q", repo.last.CREFNumber)
	}
	if repo.last.CRNNumber != "CRN3 51284" {
		t.Errorf("crn_number = %q, want %q", repo.last.CRNNumber, "CRN3 51284")
	}
}

func TestRegisterNutritionVerifiesCRNInBackground(t *testing.T) {
	crnChecks := make(chan string, 1)
	repo := &mockAuthRepo{users: make(map[string]*User), crnChecks: crnChecks, verified: make(chan string, 1)}
	svc := NewService(repo, "test_jwt_secret_with_32_characters_minimum", 24, stubCREFValidator{}, stubCRNValidator{})

	_, err := svc.Register(context.Background(), &RegisterRequest{
		Email:     "nutri@fitconecta.com",
		Password:  "password123",
		Role:      RoleProfessional,
		Name:      "Nutri Teste",
		City:      "Belo Horizonte",
		State:     "MG",
		Branch:    "nutrition",
		CRNNumber: "CRN3 51284",
	})
	if err != nil {
		t.Fatalf("esperava sucesso no registro, obteve: %v", err)
	}

	select {
	case got := <-crnChecks:
		if got != "CRN3 51284" {
			t.Errorf("CRN verificado = %q, want %q", got, "CRN3 51284")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("validacao em segundo plano nao gravou o CRN")
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
