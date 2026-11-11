package professionals

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"fitconecta/core-api/internal/cref"
	"fitconecta/core-api/internal/crn"
	"github.com/google/uuid"
)

type Service interface {
	List(ctx context.Context, f FilterParams) ([]ProfessionalProfile, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ProfessionalProfile, error)
	GetMyProfile(ctx context.Context, userID uuid.UUID) (*ProfessionalProfile, error)
	UpdateMyProfile(ctx context.Context, userID uuid.UUID, req *UpdateProfileRequest) (*ProfessionalProfile, error)
	ValidateCREF(ctx context.Context, number string) (*cref.Profile, error)
	ValidateCRN(ctx context.Context, number string) (*crn.Profile, error)
	ListSpecialties(ctx context.Context) ([]Specialty, error)
	SetMySpecialties(ctx context.Context, userID uuid.UUID, specialtyIDs []int) error

	ListCatalogs(ctx context.Context) (*CatalogResponse, error)
	SetMyObjectives(ctx context.Context, userID uuid.UUID, objectiveIDs []int) error
	SetMyAudiences(ctx context.Context, userID uuid.UUID, req *SetAudiencesRequest) error
	SetMyLocations(ctx context.Context, userID uuid.UUID, locationIDs []int) error

	ListNeighborhoods(ctx context.Context, city, state string) ([]Neighborhood, error)
	ListMyNeighborhoods(ctx context.Context, userID uuid.UUID) ([]Neighborhood, error)
	SetMyNeighborhoods(ctx context.Context, userID uuid.UUID, neighborhoodIDs []int) error

	UpdateMyAvatar(ctx context.Context, userID uuid.UUID, avatarURL string) (*ProfessionalProfile, error)
	ListMyPhotos(ctx context.Context, userID uuid.UUID) ([]ProfessionalPhoto, error)
	AddMyPhoto(ctx context.Context, userID uuid.UUID, url string) (*ProfessionalPhoto, error)
	DeleteMyPhoto(ctx context.Context, userID uuid.UUID, photoID uuid.UUID) error
}

// MaxPhotos limita a galeria de fotos do profissional.
const MaxPhotos = 3

// MaxNeighborhoods limita quantos bairros um profissional pode registrar.
const MaxNeighborhoods = 10

// audienceOtherName e a linha do catalogo que exige o texto livre.
const audienceOtherName = "Outro público específico"

var ErrAudienceOtherRequired = errors.New("informe o outro publico especifico atendido")

// ErrInvalidCatalogItem indica id que nao existe no catalogo correspondente.
var ErrInvalidCatalogItem = errors.New("opcao invalida no catalogo")

type service struct {
	repo          Repository
	crefValidator cref.Validator
	crnValidator  crn.Validator
}

func NewService(repo Repository, crefValidator cref.Validator, crnValidator crn.Validator) Service {
	return &service{repo: repo, crefValidator: crefValidator, crnValidator: crnValidator}
}

func (s *service) List(ctx context.Context, f FilterParams) ([]ProfessionalProfile, error) {
	return s.repo.List(ctx, f)
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*ProfessionalProfile, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) GetMyProfile(ctx context.Context, userID uuid.UUID) (*ProfessionalProfile, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *service) UpdateMyProfile(ctx context.Context, userID uuid.UUID, req *UpdateProfileRequest) (*ProfessionalProfile, error) {
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.DisplayName == "" {
		return nil, errors.New("nome de exibicao e obrigatorio")
	}
	if req.SessionPriceCents <= 0 {
		return nil, errors.New("o valor da sessao deve ser maior que zero")
	}
	req.City = strings.TrimSpace(req.City)
	req.State = strings.TrimSpace(req.State)
	if req.City == "" {
		return nil, errors.New("informe a cidade")
	}
	if req.State == "" {
		return nil, errors.New("informe a UF")
	}
	req.CREFNumber = strings.TrimSpace(req.CREFNumber)
	req.CRNNumber = strings.TrimSpace(req.CRNNumber)

	current, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	// So o registro do proprio ramo e salvo (ADR 0014); o outro e ignorado.
	if current.Branch == BranchNutrition {
		req.CREFNumber = ""
	} else {
		req.CRNNumber = ""
	}
	// A verificacao vale para o numero consultado: se ele mudou, o perfil
	// volta para "em analise" ate a nova consulta terminar.
	crefChanged := digitsOnly(req.CREFNumber) != digitsOnly(derefString(current.CREFNumber))
	crefVerifiedAt := current.CREFVerifiedAt
	if crefChanged {
		crefVerifiedAt = nil
	}
	crnChanged := digitsOnly(req.CRNNumber) != digitsOnly(derefString(current.CRNNumber))
	crnVerifiedAt := current.CRNVerifiedAt
	if crnChanged {
		crnVerifiedAt = nil
	}

	updated, err := s.repo.Update(ctx, userID, req, crefVerifiedAt, crnVerifiedAt)
	if err != nil {
		return nil, err
	}

	// Consulta em segundo plano: a resposta sai antes do conselho responder.
	// Cada ramo revalida apenas o seu registro (ADR 0014).
	if current.Branch == BranchNutrition {
		if req.CRNNumber != "" && (crnChanged || crnVerifiedAt == nil) {
			go s.verifyCRN(context.WithoutCancel(ctx), userID, req.CRNNumber)
		}
	} else if req.CREFNumber != "" && (crefChanged || crefVerifiedAt == nil) {
		go s.verifyCREF(context.WithoutCancel(ctx), userID, req.CREFNumber)
	}
	return updated, nil
}

func (s *service) verifyCREF(ctx context.Context, userID uuid.UUID, number string) {
	if _, err := s.crefValidator.Validate(ctx, number); err != nil {
		return
	}
	verifiedAt := time.Now()
	_ = s.repo.SetCREFVerified(ctx, userID, number, &verifiedAt)
}

func (s *service) verifyCRN(ctx context.Context, userID uuid.UUID, number string) {
	if _, err := s.crnValidator.Validate(ctx, number); err != nil {
		return
	}
	verifiedAt := time.Now()
	_ = s.repo.SetCRNVerified(ctx, userID, number, &verifiedAt)
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// digitsOnly compara so os digitos: os portais dos conselhos ignoram UF,
// categoria e zeros a esquerda, entao "012345-G/MG" e "012345" sao o mesmo
// registro.
// shortcut: trocar UF/categoria nao invalida o selo; se os portais passarem a
// diferenciar a categoria, comparar a forma normalizada completa.
func digitsOnly(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ValidateCREF consulta o portal do CREF6. O resultado e informativo: quem
// chama (cadastro/edicao) nao bloqueia o salvamento por causa dele.
func (s *service) ValidateCREF(ctx context.Context, number string) (*cref.Profile, error) {
	return s.crefValidator.Validate(ctx, number)
}

// ValidateCRN consulta a CNN do CFN. O resultado e informativo: quem chama
// (cadastro/edicao) nao bloqueia o salvamento por causa dele.
func (s *service) ValidateCRN(ctx context.Context, number string) (*crn.Profile, error) {
	return s.crnValidator.Validate(ctx, number)
}

func (s *service) ListSpecialties(ctx context.Context) ([]Specialty, error) {
	return s.repo.ListSpecialties(ctx)
}

func (s *service) SetMySpecialties(ctx context.Context, userID uuid.UUID, specialtyIDs []int) error {
	prof, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.SetSpecialties(ctx, prof.ID, specialtyIDs)
}

func (s *service) ListCatalogs(ctx context.Context) (*CatalogResponse, error) {
	services, err := s.repo.ListSpecialties(ctx)
	if err != nil {
		return nil, err
	}
	objectives, err := s.repo.ListObjectives(ctx)
	if err != nil {
		return nil, err
	}
	audiences, err := s.repo.ListAudiences(ctx)
	if err != nil {
		return nil, err
	}
	locations, err := s.repo.ListServiceLocations(ctx)
	if err != nil {
		return nil, err
	}
	return &CatalogResponse{
		Services:   services,
		Objectives: objectives,
		Audiences:  audiences,
		Locations:  locations,
	}, nil
}

func (s *service) SetMyObjectives(ctx context.Context, userID uuid.UUID, objectiveIDs []int) error {
	catalog, err := s.repo.ListObjectives(ctx)
	if err != nil {
		return err
	}
	if !allIDsExist(objectiveIDs, idsOf(catalog, func(o Objective) int { return o.ID })) {
		return ErrInvalidCatalogItem
	}

	prof, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.SetObjectives(ctx, prof.ID, objectiveIDs)
}

func (s *service) SetMyAudiences(ctx context.Context, userID uuid.UUID, req *SetAudiencesRequest) error {
	req.AudienceOther = strings.TrimSpace(req.AudienceOther)
	req.AudienceExperience = strings.TrimSpace(req.AudienceExperience)

	catalog, err := s.repo.ListAudiences(ctx)
	if err != nil {
		return err
	}
	if !allIDsExist(req.AudienceIDs, idsOf(catalog, func(a Audience) int { return a.ID })) {
		return ErrInvalidCatalogItem
	}
	if missingAudienceOther(catalog, req.AudienceIDs, req.AudienceOther) {
		return ErrAudienceOtherRequired
	}

	prof, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.SetAudiences(ctx, prof.ID, req)
}

func (s *service) SetMyLocations(ctx context.Context, userID uuid.UUID, locationIDs []int) error {
	catalog, err := s.repo.ListServiceLocations(ctx)
	if err != nil {
		return err
	}
	if !allIDsExist(locationIDs, idsOf(catalog, func(l ServiceLocation) int { return l.ID })) {
		return ErrInvalidCatalogItem
	}

	prof, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.SetLocations(ctx, prof.ID, locationIDs)
}

func (s *service) ListNeighborhoods(ctx context.Context, city, state string) ([]Neighborhood, error) {
	return s.repo.ListNeighborhoods(ctx, strings.TrimSpace(city), strings.TrimSpace(state))
}

// missingAudienceOther indica se "Outro publico especifico" foi selecionado sem descricao.
func missingAudienceOther(catalog []Audience, ids []int, other string) bool {
	for _, a := range catalog {
		if a.Name == audienceOtherName && slices.Contains(ids, a.ID) {
			return other == ""
		}
	}
	return false
}

// allIDsExist garante que todos os ids enviados pertencem ao catalogo.
func allIDsExist(ids []int, valid []int) bool {
	allowed := make(map[int]struct{}, len(valid))
	for _, id := range valid {
		allowed[id] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := allowed[id]; !ok {
			return false
		}
	}
	return true
}

func idsOf[T any](items []T, id func(T) int) []int {
	out := make([]int, len(items))
	for i, item := range items {
		out[i] = id(item)
	}
	return out
}

func (s *service) ListMyNeighborhoods(ctx context.Context, userID uuid.UUID) ([]Neighborhood, error) {
	prof, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetNeighborhoods(ctx, prof.ID)
}

func (s *service) SetMyNeighborhoods(ctx context.Context, userID uuid.UUID, neighborhoodIDs []int) error {
	if len(neighborhoodIDs) > MaxNeighborhoods {
		return fmt.Errorf("limite de %d bairros atingido", MaxNeighborhoods)
	}
	prof, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.SetNeighborhoods(ctx, prof.ID, neighborhoodIDs)
}

func (s *service) UpdateMyAvatar(ctx context.Context, userID uuid.UUID, avatarURL string) (*ProfessionalProfile, error) {
	return s.repo.UpdateAvatar(ctx, userID, strings.TrimSpace(avatarURL))
}

func (s *service) ListMyPhotos(ctx context.Context, userID uuid.UUID) ([]ProfessionalPhoto, error) {
	prof, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListPhotos(ctx, prof.ID)
}

func (s *service) AddMyPhoto(ctx context.Context, userID uuid.UUID, url string) (*ProfessionalPhoto, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, errors.New("url da foto e obrigatoria")
	}

	prof, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	photos, err := s.repo.ListPhotos(ctx, prof.ID)
	if err != nil {
		return nil, err
	}
	if len(photos) >= MaxPhotos {
		return nil, fmt.Errorf("limite de %d fotos atingido", MaxPhotos)
	}

	return s.repo.AddPhoto(ctx, prof.ID, url)
}

func (s *service) DeleteMyPhoto(ctx context.Context, userID uuid.UUID, photoID uuid.UUID) error {
	prof, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.DeletePhoto(ctx, prof.ID, photoID)
}
