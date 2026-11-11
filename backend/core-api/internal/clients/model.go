package clients

import "time"

// Valores dos enums client_sex e client_activity_level (migrations 0013 e 0014).
const (
	SexMale   = "male"
	SexFemale = "female"
	SexOther  = "other"
)

const (
	ActivitySedentary = "sedentary"
	ActivityCasual    = "casual"
	ActivityFrequent  = "frequent"
)

var (
	validSexes          = []string{SexMale, SexFemale, SexOther}
	validActivityLevels = []string{ActivitySedentary, ActivityCasual, ActivityFrequent}
)

// maxObjectiveLength casa com o CHECK chk_client_objective_length (0014).
const maxObjectiveLength = 500

// ClientPersonalProfile e o dado fisico como esta no banco; nil = nao informado.
type ClientPersonalProfile struct {
	BirthDate     *time.Time
	Sex           *string
	HeightCM      *int16
	WeightKG      *float64
	ActivityLevel *string
	Objective     *string
	CPF           *string
}

// UpdatePersonalDataRequest substitui os campos; ausente ou null grava NULL.
type UpdatePersonalDataRequest struct {
	BirthDate     *string  `json:"birth_date"`
	Sex           *string  `json:"sex"`
	HeightCM      *int     `json:"height_cm"`
	WeightKG      *float64 `json:"weight_kg"`
	ActivityLevel *string  `json:"activity_level"`
	Objective     *string  `json:"objective"`
	CPF           *string  `json:"cpf"`
}

// PersonalProfileResponse e o contrato de saida, com IMC e idade derivados.
type PersonalProfileResponse struct {
	BirthDate     *string  `json:"birth_date"`
	Sex           *string  `json:"sex"`
	HeightCM      *int16   `json:"height_cm"`
	WeightKG      *float64 `json:"weight_kg"`
	ActivityLevel *string  `json:"activity_level"`
	Objective     *string  `json:"objective"`
	CPF           *string  `json:"cpf"`
	BMI           *float64 `json:"bmi"`
	Age           *int     `json:"age"`
}
