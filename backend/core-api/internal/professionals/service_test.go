package professionals

import (
	"context"
	"testing"
	"time"

	"fitconecta/core-api/internal/cref"
	"fitconecta/core-api/internal/crn"
	"github.com/google/uuid"
)

func TestMissingAudienceOther(t *testing.T) {
	catalog := []Audience{
		{ID: 1, Name: "Adultos em geral"},
		{ID: 2, Name: audienceOtherName, IsSpecific: true},
	}

	tests := []struct {
		name  string
		ids   []int
		other string
		want  bool
	}{
		{name: "sem outro publico", ids: []int{1}, other: "", want: false},
		{name: "outro publico sem descricao", ids: []int{1, 2}, other: "", want: true},
		{name: "outro publico com descricao", ids: []int{1, 2}, other: "Atletas masters", want: false},
		{name: "nenhum publico selecionado", ids: nil, other: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := missingAudienceOther(catalog, tt.ids, tt.other); got != tt.want {
				t.Errorf("missingAudienceOther() = %v, want %v", got, tt.want)
			}
		})
	}
}

type updateFakeRepo struct {
	Repository
	current     *ProfessionalProfile
	gotVerify   *time.Time
	gotCRNCheck *time.Time
	verifiedCh  chan string
	crnCh       chan string
}

func (f *updateFakeRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*ProfessionalProfile, error) {
	return f.current, nil
}

func (f *updateFakeRepo) Update(ctx context.Context, userID uuid.UUID, req *UpdateProfileRequest, crefVerifiedAt, crnVerifiedAt *time.Time) (*ProfessionalProfile, error) {
	f.gotVerify = crefVerifiedAt
	f.gotCRNCheck = crnVerifiedAt
	return &ProfessionalProfile{
		DisplayName: req.DisplayName, CREFNumber: &req.CREFNumber, CREFVerifiedAt: crefVerifiedAt,
		CRNNumber: &req.CRNNumber, CRNVerifiedAt: crnVerifiedAt,
	}, nil
}

func (f *updateFakeRepo) SetCREFVerified(ctx context.Context, userID uuid.UUID, number string, verifiedAt *time.Time) error {
	f.verifiedCh <- number
	return nil
}

func (f *updateFakeRepo) SetCRNVerified(ctx context.Context, userID uuid.UUID, number string, verifiedAt *time.Time) error {
	f.crnCh <- number
	return nil
}

type okCREFValidator struct{}

func (okCREFValidator) Validate(ctx context.Context, number string) (*cref.Profile, error) {
	return &cref.Profile{Number: number, Status: "ATIVO"}, nil
}

type okCRNValidator struct{}

func (okCRNValidator) Validate(ctx context.Context, number string) (*crn.Profile, error) {
	return &crn.Profile{Number: number, Name: "NUTRICIONISTA TESTE"}, nil
}

func strPtr(v string) *string { return &v }

func newUpdateReq(crefNumber, crnNumber string) *UpdateProfileRequest {
	return &UpdateProfileRequest{DisplayName: "Ana Personal", City: "Belo Horizonte", State: "MG", SessionPriceCents: 10000, CREFNumber: crefNumber, CRNNumber: crnNumber}
}

func TestUpdateMyProfileRequiresCityAndState(t *testing.T) {
	repo := &updateFakeRepo{
		current:    &ProfessionalProfile{CREFNumber: strPtr("012345")},
		verifiedCh: make(chan string, 1),
		crnCh:      make(chan string, 1),
	}
	svc := NewService(repo, okCREFValidator{}, okCRNValidator{})

	req := newUpdateReq("012345", "")
	req.City = ""
	if _, err := svc.UpdateMyProfile(context.Background(), uuid.New(), req); err == nil || err.Error() != "informe a cidade" {
		t.Fatalf("esperava erro de cidade, obteve: %v", err)
	}
	req = newUpdateReq("012345", "")
	req.State = ""
	if _, err := svc.UpdateMyProfile(context.Background(), uuid.New(), req); err == nil || err.Error() != "informe a UF" {
		t.Fatalf("esperava erro de UF, obteve: %v", err)
	}
}

func TestUpdateMyProfileKeepsCREFStatusWhenNumberUnchanged(t *testing.T) {
	verifiedAt := time.Now().Add(-time.Hour)
	repo := &updateFakeRepo{
		current:    &ProfessionalProfile{CREFNumber: strPtr("MG 012345"), CREFVerifiedAt: &verifiedAt},
		verifiedCh: make(chan string, 1),
		crnCh:      make(chan string, 1),
	}
	svc := NewService(repo, okCREFValidator{}, okCRNValidator{})

	if _, err := svc.UpdateMyProfile(context.Background(), uuid.New(), newUpdateReq("012345", "")); err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if repo.gotVerify == nil || !repo.gotVerify.Equal(verifiedAt) {
		t.Error("status de verificacao deveria ser preservado quando o numero nao muda")
	}
	select {
	case number := <-repo.verifiedCh:
		t.Errorf("nao deveria revalidar sem mudanca de numero, revalidou %q", number)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestUpdateMyProfileClearsAndRevalidatesChangedCREF(t *testing.T) {
	verifiedAt := time.Now().Add(-time.Hour)
	repo := &updateFakeRepo{
		current:    &ProfessionalProfile{CREFNumber: strPtr("012345"), CREFVerifiedAt: &verifiedAt},
		verifiedCh: make(chan string, 1),
		crnCh:      make(chan string, 1),
	}
	svc := NewService(repo, okCREFValidator{}, okCRNValidator{})

	if _, err := svc.UpdateMyProfile(context.Background(), uuid.New(), newUpdateReq("999999", "")); err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if repo.gotVerify != nil {
		t.Error("status deveria ser limpo ao trocar o numero do CREF")
	}
	select {
	case number := <-repo.verifiedCh:
		if number != "999999" {
			t.Errorf("revalidou %q, quer %q", number, "999999")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("numero alterado nao foi revalidado em segundo plano")
	}
}

func TestUpdateMyProfileRevalidatesOldProfileWithoutStatus(t *testing.T) {
	repo := &updateFakeRepo{
		current:    &ProfessionalProfile{CREFNumber: strPtr("012345")},
		verifiedCh: make(chan string, 1),
		crnCh:      make(chan string, 1),
	}
	svc := NewService(repo, okCREFValidator{}, okCRNValidator{})

	if _, err := svc.UpdateMyProfile(context.Background(), uuid.New(), newUpdateReq("012345", "")); err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	select {
	case number := <-repo.verifiedCh:
		if number != "012345" {
			t.Errorf("revalidou %q, quer %q", number, "012345")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("perfil sem status deveria ser revalidado")
	}
}

func TestUpdateMyProfileKeepsCRNStatusWhenNumberUnchanged(t *testing.T) {
	verifiedAt := time.Now().Add(-time.Hour)
	repo := &updateFakeRepo{
		current:    &ProfessionalProfile{Branch: BranchNutrition, CRNNumber: strPtr("CRN3 51284"), CRNVerifiedAt: &verifiedAt},
		verifiedCh: make(chan string, 1),
		crnCh:      make(chan string, 1),
	}
	svc := NewService(repo, okCREFValidator{}, okCRNValidator{})

	if _, err := svc.UpdateMyProfile(context.Background(), uuid.New(), newUpdateReq("", "CRN3 51284")); err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if repo.gotCRNCheck == nil || !repo.gotCRNCheck.Equal(verifiedAt) {
		t.Error("status de verificacao do CRN deveria ser preservado quando o numero nao muda")
	}
	select {
	case number := <-repo.crnCh:
		t.Errorf("nao deveria revalidar sem mudanca de numero, revalidou %q", number)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestUpdateMyProfileIgnoCouncilOfOtherBranch(t *testing.T) {
	repo := &updateFakeRepo{
		current:    &ProfessionalProfile{Branch: BranchNutrition, CRNNumber: strPtr("CRN3 51284")},
		verifiedCh: make(chan string, 1),
		crnCh:      make(chan string, 1),
	}
	svc := NewService(repo, okCREFValidator{}, okCRNValidator{})

	updated, err := svc.UpdateMyProfile(context.Background(), uuid.New(), newUpdateReq("999999-G/MG", "CRN3 51284"))
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if got := derefString(updated.CREFNumber); got != "" {
		t.Errorf("nutricionista nao deveria salvar CREF, obteve %q", got)
	}

	repo.current = &ProfessionalProfile{CREFNumber: strPtr("012345")}
	if updated, err = svc.UpdateMyProfile(context.Background(), uuid.New(), newUpdateReq("012345", "CRN9 39104")); err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if got := derefString(updated.CRNNumber); got != "" {
		t.Errorf("EF nao deveria salvar CRN, obteve %q", got)
	}
}

func TestUpdateMyProfileClearsAndRevalidatesChangedCRN(t *testing.T) {
	verifiedAt := time.Now().Add(-time.Hour)
	repo := &updateFakeRepo{
		current:    &ProfessionalProfile{Branch: BranchNutrition, CRNNumber: strPtr("51284"), CRNVerifiedAt: &verifiedAt},
		verifiedCh: make(chan string, 1),
		crnCh:      make(chan string, 1),
	}
	svc := NewService(repo, okCREFValidator{}, okCRNValidator{})

	if _, err := svc.UpdateMyProfile(context.Background(), uuid.New(), newUpdateReq("", "99999")); err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if repo.gotCRNCheck != nil {
		t.Error("status do CRN deveria ser limpo ao trocar o numero")
	}
	select {
	case number := <-repo.crnCh:
		if number != "99999" {
			t.Errorf("revalidou %q, quer %q", number, "99999")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("numero de CRN alterado nao foi revalidado em segundo plano")
	}
}
