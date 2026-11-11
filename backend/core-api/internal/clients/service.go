package clients

import (
	"context"
	"math"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	maxAgeYears = 120
	minHeightCM = 100
	maxHeightCM = 250
	minWeightKG = 30
	maxWeightKG = 300
)

// ValidationError carrega a mensagem pt-BR exibida ao cliente (HTTP 400).
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

var (
	errFutureBirthDate  = &ValidationError{"A data de nascimento nao pode estar no futuro."}
	errInvalidBirthDate = &ValidationError{"Informe uma data de nascimento valida."}
	errInvalidHeight    = &ValidationError{"Informe a altura em centimetros, entre 100 e 250."}
	errInvalidWeight    = &ValidationError{"Informe o peso em quilos, entre 30 e 300."}
	errInvalidSex       = &ValidationError{"Selecione uma opcao valida para sexo."}
	errInvalidActivity  = &ValidationError{"Selecione uma opcao valida para frequencia de atividades."}
	errLongObjective    = &ValidationError{"Descreva seu objetivo em ate 500 caracteres."}
	errInvalidCPF       = &ValidationError{"Informe um CPF valido, com 11 numeros."}
)

// Idade e "data futura" sao calculadas no fuso do publico (ADR 0012 §4.1).
var americaSaoPaulo = func() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return time.FixedZone("BRT", -3*60*60)
	}
	return loc
}()

var minBirthDate = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)

type Service interface {
	GetMyProfile(ctx context.Context, userID uuid.UUID) (*PersonalProfileResponse, error)
	UpdateMyProfile(ctx context.Context, userID uuid.UUID, req *UpdatePersonalDataRequest) (*PersonalProfileResponse, error)
}

type service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) Service {
	return &service{repo: repo, now: time.Now}
}

func (s *service) GetMyProfile(ctx context.Context, userID uuid.UUID) (*PersonalProfileResponse, error) {
	p, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toResponse(p, s.now()), nil
}

func (s *service) UpdateMyProfile(ctx context.Context, userID uuid.UUID, req *UpdatePersonalDataRequest) (*PersonalProfileResponse, error) {
	now := s.now()

	data, err := req.validate(now)
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.Update(ctx, userID, data)
	if err != nil {
		return nil, err
	}
	return toResponse(updated, now), nil
}

// validate normaliza a substituicao total dos cinco campos e aplica as faixas.
func (req *UpdatePersonalDataRequest) validate(now time.Time) (*ClientPersonalProfile, error) {
	today := dateOnly(now.In(americaSaoPaulo))

	birthDate, err := parseBirthDate(req.BirthDate, today)
	if err != nil {
		return nil, err
	}

	var height *int16
	if req.HeightCM != nil {
		if *req.HeightCM < minHeightCM || *req.HeightCM > maxHeightCM {
			return nil, errInvalidHeight
		}
		h := int16(*req.HeightCM)
		height = &h
	}

	var weight *float64
	if req.WeightKG != nil {
		if *req.WeightKG < minWeightKG || *req.WeightKG > maxWeightKG {
			return nil, errInvalidWeight
		}
		w := *req.WeightKG
		weight = &w
	}

	sex, err := normalizeEnum(req.Sex, validSexes, errInvalidSex)
	if err != nil {
		return nil, err
	}

	activity, err := normalizeEnum(req.ActivityLevel, validActivityLevels, errInvalidActivity)
	if err != nil {
		return nil, err
	}

	objective, err := normalizeText(req.Objective, maxObjectiveLength, errLongObjective)
	if err != nil {
		return nil, err
	}

	cpf, err := normalizeCPF(req.CPF)
	if err != nil {
		return nil, err
	}

	return &ClientPersonalProfile{
		BirthDate:     birthDate,
		Sex:           sex,
		HeightCM:      height,
		WeightKG:      weight,
		ActivityLevel: activity,
		Objective:     objective,
		CPF:           cpf,
	}, nil
}

func parseBirthDate(value *string, today time.Time) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	raw := strings.TrimSpace(*value)
	if raw == "" {
		return nil, nil
	}

	birth, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, errInvalidBirthDate
	}
	if birth.After(today) {
		return nil, errFutureBirthDate
	}
	if birth.Before(minBirthDate) || ageAt(birth, today) > maxAgeYears {
		return nil, errInvalidBirthDate
	}
	return &birth, nil
}

// normalizeEnum trata string vazia como "nao informado".
func normalizeEnum(value *string, allowed []string, invalid error) (*string, error) {
	if value == nil {
		return nil, nil
	}
	v := strings.TrimSpace(*value)
	if v == "" {
		return nil, nil
	}
	if !slices.Contains(allowed, v) {
		return nil, invalid
	}
	return &v, nil
}

// normalizeText trata string vazia como "nao informado" e limita o tamanho em
// caracteres, igual ao char_length do CHECK.
func normalizeText(value *string, maxLen int, tooLong error) (*string, error) {
	if value == nil {
		return nil, nil
	}
	v := strings.TrimSpace(*value)
	if v == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(v) > maxLen {
		return nil, tooLong
	}
	return &v, nil
}

// normalizeCPF aceita "000.000.000-00" ou "00000000000" e grava só os digitos.
func normalizeCPF(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	raw := strings.TrimSpace(*value)
	if raw == "" {
		return nil, nil
	}

	digits := strings.NewReplacer(".", "", "-", "", " ", "").Replace(raw)
	if len(digits) != 11 {
		return nil, errInvalidCPF
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return nil, errInvalidCPF
		}
	}
	return &digits, nil
}

// ageAt conta anos completos entre duas datas de calendario.
func ageAt(birth, today time.Time) int {
	years := today.Year() - birth.Year()
	if today.Month() < birth.Month() || (today.Month() == birth.Month() && today.Day() < birth.Day()) {
		years--
	}
	return years
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func toResponse(p *ClientPersonalProfile, now time.Time) *PersonalProfileResponse {
	resp := &PersonalProfileResponse{
		Sex:           p.Sex,
		HeightCM:      p.HeightCM,
		WeightKG:      p.WeightKG,
		ActivityLevel: p.ActivityLevel,
		Objective:     p.Objective,
		CPF:           p.CPF,
	}

	if p.BirthDate != nil {
		formatted := p.BirthDate.Format("2006-01-02")
		age := ageAt(*p.BirthDate, dateOnly(now.In(americaSaoPaulo)))
		resp.BirthDate = &formatted
		resp.Age = &age
	}

	if p.WeightKG != nil && p.HeightCM != nil {
		meters := float64(*p.HeightCM) / 100
		bmi := math.Round(*p.WeightKG/(meters*meters)*10) / 10
		resp.BMI = &bmi
	}

	return resp
}
