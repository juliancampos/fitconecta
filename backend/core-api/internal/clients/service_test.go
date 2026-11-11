package clients

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeRepository struct {
	profile *ClientPersonalProfile
	missing bool
}

func (f *fakeRepository) GetByUserID(context.Context, uuid.UUID) (*ClientPersonalProfile, error) {
	if f.missing {
		return nil, ErrProfileNotFound
	}
	if f.profile == nil {
		f.profile = &ClientPersonalProfile{}
	}
	return f.profile, nil
}

func (f *fakeRepository) Update(_ context.Context, _ uuid.UUID, data *ClientPersonalProfile) (*ClientPersonalProfile, error) {
	if f.missing {
		return nil, ErrProfileNotFound
	}
	f.profile = data
	return data, nil
}

// 1 de novembro de 2026, 15h UTC (12h em America/Sao_Paulo).
var fixedNow = time.Date(2026, time.November, 1, 15, 0, 0, 0, time.UTC)

func newTestService(repo Repository) Service {
	return &service{repo: repo, now: func() time.Time { return fixedNow }}
}

func strPtr(v string) *string     { return &v }
func intPtr(v int) *int           { return &v }
func floatPtr(v float64) *float64 { return &v }

func TestUpdateMyProfileValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdatePersonalDataRequest
		wantErr string
	}{
		{
			name: "todos os campos nulos",
			req:  UpdatePersonalDataRequest{},
		},
		{
			name: "strings vazias contam como nulo",
			req: UpdatePersonalDataRequest{
				BirthDate:     strPtr(""),
				Sex:           strPtr(""),
				ActivityLevel: strPtr(""),
				Objective:     strPtr(""),
			},
		},
		{
			name: "completo",
			req: UpdatePersonalDataRequest{
				BirthDate:     strPtr("1990-05-17"),
				Sex:           strPtr("female"),
				HeightCM:      intPtr(168),
				WeightKG:      floatPtr(64.5),
				ActivityLevel: strPtr("casual"),
				Objective:     strPtr("Correr 5 km sem parar"),
			},
		},
		{
			name: "limites inferiores validos",
			req: UpdatePersonalDataRequest{
				HeightCM: intPtr(100),
				WeightKG: floatPtr(30),
			},
		},
		{
			name: "limites superiores validos",
			req: UpdatePersonalDataRequest{
				HeightCM: intPtr(250),
				WeightKG: floatPtr(300),
			},
		},
		{
			name: "idade exatamente 120 e valida",
			req:  UpdatePersonalDataRequest{BirthDate: strPtr("1905-11-02")},
		},
		{
			name:    "data no futuro",
			req:     UpdatePersonalDataRequest{BirthDate: strPtr("2026-11-02")},
			wantErr: "A data de nascimento nao pode estar no futuro.",
		},
		{
			name:    "data antes de 1900",
			req:     UpdatePersonalDataRequest{BirthDate: strPtr("1899-12-31")},
			wantErr: "Informe uma data de nascimento valida.",
		},
		{
			name:    "idade acima de 120",
			req:     UpdatePersonalDataRequest{BirthDate: strPtr("1905-10-31")},
			wantErr: "Informe uma data de nascimento valida.",
		},
		{
			name:    "formato de data invalido",
			req:     UpdatePersonalDataRequest{BirthDate: strPtr("17/05/1990")},
			wantErr: "Informe uma data de nascimento valida.",
		},
		{
			name:    "altura abaixo da faixa",
			req:     UpdatePersonalDataRequest{HeightCM: intPtr(99)},
			wantErr: "Informe a altura em centimetros, entre 100 e 250.",
		},
		{
			name:    "altura acima da faixa",
			req:     UpdatePersonalDataRequest{HeightCM: intPtr(251)},
			wantErr: "Informe a altura em centimetros, entre 100 e 250.",
		},
		{
			name:    "peso abaixo da faixa",
			req:     UpdatePersonalDataRequest{WeightKG: floatPtr(29.9)},
			wantErr: "Informe o peso em quilos, entre 30 e 300.",
		},
		{
			name:    "peso acima da faixa",
			req:     UpdatePersonalDataRequest{WeightKG: floatPtr(300.1)},
			wantErr: "Informe o peso em quilos, entre 30 e 300.",
		},
		{
			name:    "sexo invalido",
			req:     UpdatePersonalDataRequest{Sex: strPtr("F")},
			wantErr: "Selecione uma opcao valida para sexo.",
		},
		{
			name:    "nivel de atividade invalido",
			req:     UpdatePersonalDataRequest{ActivityLevel: strPtr("sometimes")},
			wantErr: "Selecione uma opcao valida para frequencia de atividades.",
		},
		{
			name: "objetivo com 500 caracteres e valido",
			req:  UpdatePersonalDataRequest{Objective: strPtr(strings.Repeat("a", 500))},
		},
		{
			name:    "objetivo acima de 500 caracteres",
			req:     UpdatePersonalDataRequest{Objective: strPtr(strings.Repeat("a", 501))},
			wantErr: "Descreva seu objetivo em ate 500 caracteres.",
		},
		{
			name: "cpf formatado e valido",
			req:  UpdatePersonalDataRequest{CPF: strPtr("529.982.247-25")},
		},
		{
			name:    "cpf com 10 digitos",
			req:     UpdatePersonalDataRequest{CPF: strPtr("5299822472")},
			wantErr: "Informe um CPF valido, com 11 numeros.",
		},
		{
			name:    "cpf com letras",
			req:     UpdatePersonalDataRequest{CPF: strPtr("5299822472a")},
			wantErr: "Informe um CPF valido, com 11 numeros.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(&fakeRepository{})

			_, err := svc.UpdateMyProfile(context.Background(), uuid.New(), &tt.req)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("UpdateMyProfile() erro inesperado: %v", err)
				}
				return
			}

			var validation *ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("UpdateMyProfile() erro = %v, want ValidationError %q", err, tt.wantErr)
			}
			if validation.Message != tt.wantErr {
				t.Errorf("mensagem = %q, want %q", validation.Message, tt.wantErr)
			}
		})
	}
}

func TestUpdateMyProfileClearsFieldsWhenNull(t *testing.T) {
	repo := &fakeRepository{profile: &ClientPersonalProfile{
		Sex:       strPtr(SexMale),
		HeightCM:  int16Ptr(180),
		Objective: strPtr("Ganhar massa"),
		CPF:       strPtr("52998224725"),
	}}
	svc := newTestService(repo)

	got, err := svc.UpdateMyProfile(context.Background(), uuid.New(), &UpdatePersonalDataRequest{})
	if err != nil {
		t.Fatalf("UpdateMyProfile() erro inesperado: %v", err)
	}

	if got.BirthDate != nil || got.Sex != nil || got.HeightCM != nil || got.WeightKG != nil ||
		got.ActivityLevel != nil || got.Objective != nil || got.CPF != nil {
		t.Errorf("UpdateMyProfile() manteve campos antigos: %+v", got)
	}
	if got.BMI != nil || got.Age != nil {
		t.Errorf("UpdateMyProfile() derivados = %v/%v, want nil/nil", got.BMI, got.Age)
	}
}

func TestGetMyProfileEmpty(t *testing.T) {
	svc := newTestService(&fakeRepository{})

	got, err := svc.GetMyProfile(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetMyProfile() erro inesperado: %v", err)
	}

	if got.BirthDate != nil || got.Sex != nil || got.HeightCM != nil || got.WeightKG != nil ||
		got.ActivityLevel != nil || got.Objective != nil || got.CPF != nil {
		t.Errorf("GetMyProfile() perfil vazio devolveu dados: %+v", got)
	}
	if got.BMI != nil || got.Age != nil {
		t.Errorf("GetMyProfile() derivados = %v/%v, want nil/nil", got.BMI, got.Age)
	}
}

func TestUpdateMyProfileDerivedValues(t *testing.T) {
	tests := []struct {
		name      string
		req       UpdatePersonalDataRequest
		wantBMI   *float64
		wantAge   *int
		wantBirth string
	}{
		{
			name: "imc e idade completos",
			req: UpdatePersonalDataRequest{
				BirthDate: strPtr("1990-05-17"),
				HeightCM:  intPtr(168),
				WeightKG:  floatPtr(64.5),
			},
			wantBMI:   floatPtr(22.9),
			wantAge:   intPtr(36),
			wantBirth: "1990-05-17",
		},
		{
			name: "sem peso nao calcula imc",
			req: UpdatePersonalDataRequest{
				BirthDate: strPtr("1990-05-17"),
				HeightCM:  intPtr(168),
			},
			wantAge:   intPtr(36),
			wantBirth: "1990-05-17",
		},
		{
			name: "sem altura nao calcula imc",
			req: UpdatePersonalDataRequest{
				WeightKG: floatPtr(64.5),
			},
		},
		{
			name: "aniversario ainda nao chegou no ano",
			req: UpdatePersonalDataRequest{
				BirthDate: strPtr("1990-12-25"),
			},
			wantAge:   intPtr(35),
			wantBirth: "1990-12-25",
		},
		{
			name: "aniversario e hoje",
			req: UpdatePersonalDataRequest{
				BirthDate: strPtr("1990-11-01"),
			},
			wantAge:   intPtr(36),
			wantBirth: "1990-11-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(&fakeRepository{})

			got, err := svc.UpdateMyProfile(context.Background(), uuid.New(), &tt.req)
			if err != nil {
				t.Fatalf("UpdateMyProfile() erro inesperado: %v", err)
			}

			if !equalPtr(got.BMI, tt.wantBMI) {
				t.Errorf("bmi = %v, want %v", deref(got.BMI), deref(tt.wantBMI))
			}
			if !equalPtr(got.Age, tt.wantAge) {
				t.Errorf("age = %v, want %v", deref(got.Age), deref(tt.wantAge))
			}
			if tt.wantBirth != "" && (got.BirthDate == nil || *got.BirthDate != tt.wantBirth) {
				t.Errorf("birth_date = %v, want %q", deref(got.BirthDate), tt.wantBirth)
			}
		})
	}
}

func TestAgeAt(t *testing.T) {
	today := dateOnly(fixedNow)
	tests := []struct {
		name  string
		birth string
		want  int
	}{
		{name: "mesmo dia", birth: "1990-11-01", want: 36},
		{name: "um dia depois do aniversario", birth: "1990-10-31", want: 36},
		{name: "aniversario amanha", birth: "1990-11-02", want: 35},
		{name: "nascido hoje", birth: "2026-11-01", want: 0},
		{name: "29 de fevereiro", birth: "2000-02-29", want: 26},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			birth, err := time.Parse("2006-01-02", tt.birth)
			if err != nil {
				t.Fatalf("data invalida no teste: %v", err)
			}
			if got := ageAt(birth, today); got != tt.want {
				t.Errorf("ageAt(%s) = %d, want %d", tt.birth, got, tt.want)
			}
		})
	}
}

func TestGetMyProfileMissingProfile(t *testing.T) {
	svc := newTestService(&fakeRepository{missing: true})

	if _, err := svc.GetMyProfile(context.Background(), uuid.New()); !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("GetMyProfile() erro = %v, want ErrProfileNotFound", err)
	}
}

func TestPersonalProfileResponseJSON(t *testing.T) {
	empty, err := json.Marshal(&PersonalProfileResponse{})
	if err != nil {
		t.Fatalf("json.Marshal() erro: %v", err)
	}
	wantEmpty := `{"birth_date":null,"sex":null,"height_cm":null,"weight_kg":null,"activity_level":null,"objective":null,"cpf":null,"bmi":null,"age":null}`
	if string(empty) != wantEmpty {
		t.Errorf("perfil vazio = %s, want %s", empty, wantEmpty)
	}

	full, err := json.Marshal(&PersonalProfileResponse{
		BirthDate:     strPtr("1990-05-17"),
		Sex:           strPtr(SexFemale),
		HeightCM:      int16Ptr(168),
		WeightKG:      floatPtr(64.5),
		ActivityLevel: strPtr(ActivityCasual),
		Objective:     strPtr("Correr 5 km sem parar"),
		CPF:           strPtr("52998224725"),
		BMI:           floatPtr(22.9),
		Age:           intPtr(36),
	})
	if err != nil {
		t.Fatalf("json.Marshal() erro: %v", err)
	}
	wantFull := `{"birth_date":"1990-05-17","sex":"female","height_cm":168,"weight_kg":64.5,"activity_level":"casual","objective":"Correr 5 km sem parar","cpf":"52998224725","bmi":22.9,"age":36}`
	if string(full) != wantFull {
		t.Errorf("perfil completo = %s, want %s", full, wantFull)
	}
}

func TestNormalizeCPF(t *testing.T) {
	tests := []struct {
		name    string
		in      *string
		want    *string
		wantErr bool
	}{
		{name: "nulo", in: nil},
		{name: "vazio", in: strPtr("   ")},
		{name: "só digitos", in: strPtr("52998224725"), want: strPtr("52998224725")},
		{name: "com mascara", in: strPtr("529.982.247-25"), want: strPtr("52998224725")},
		{name: "curto", in: strPtr("5299822472"), wantErr: true},
		{name: "com letra", in: strPtr("5299822472a"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeCPF(tt.in)
			if tt.wantErr {
				var validation *ValidationError
				if !errors.As(err, &validation) {
					t.Fatalf("normalizeCPF() erro = %v, want ValidationError", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeCPF() erro inesperado: %v", err)
			}
			if !equalPtr(got, tt.want) {
				t.Errorf("normalizeCPF() = %v, want %v", deref(got), deref(tt.want))
			}
		})
	}
}

func int16Ptr(v int16) *int16 { return &v }

func deref[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}

func equalPtr[T comparable](got, want *T) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}
	return *got == *want
}
