package availability

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockAvailRepo struct {
	rules      []AvailabilityRule
	exceptions []AvailabilityException
	booked     []BookedInterval
}

func (m *mockAvailRepo) CreateRule(ctx context.Context, profID uuid.UUID, req *CreateRuleRequest) (*AvailabilityRule, error) {
	return nil, nil
}
func (m *mockAvailRepo) ListRules(ctx context.Context, profID uuid.UUID) ([]AvailabilityRule, error) {
	return m.rules, nil
}
func (m *mockAvailRepo) DeleteRule(ctx context.Context, profID, ruleID uuid.UUID) error {
	return nil
}
func (m *mockAvailRepo) CreateException(ctx context.Context, profID uuid.UUID, req *CreateExceptionRequest) (*AvailabilityException, error) {
	return nil, nil
}
func (m *mockAvailRepo) ListExceptions(ctx context.Context, profID uuid.UUID, from, to time.Time) ([]AvailabilityException, error) {
	return m.exceptions, nil
}
func (m *mockAvailRepo) DeleteException(ctx context.Context, profID, excID uuid.UUID) error {
	return nil
}
func (m *mockAvailRepo) GetBookedIntervals(ctx context.Context, profID uuid.UUID, from, to time.Time) ([]BookedInterval, error) {
	return m.booked, nil
}
func (m *mockAvailRepo) GetProfessionalTimezone(ctx context.Context, profID uuid.UUID) (string, error) {
	return "America/Sao_Paulo", nil
}

func TestGenerateSlots(t *testing.T) {
	profID := uuid.New()
	targetDate, _ := time.Parse("2006-01-02", "2026-10-12") // Segunda-feira (Weekday = 1)
	loc, _ := time.LoadLocation("America/Sao_Paulo")

	repo := &mockAvailRepo{
		rules: []AvailabilityRule{
			{
				ID:             uuid.New(),
				ProfessionalID: profID,
				Weekday:        1, // Segunda-feira
				StartTime:      "08:00",
				EndTime:        "11:00", // 3 slots de 60 min: 08-09, 09-10, 10-11
				SlotMinutes:    60,
			},
		},
		// 1 agendamento existente das 09:00 as 10:00
		booked: []BookedInterval{
			{
				StartsAt: time.Date(2026, 10, 12, 9, 0, 0, 0, loc).UTC(),
				EndsAt:   time.Date(2026, 10, 12, 10, 0, 0, 0, loc).UTC(),
			},
		},
	}

	svc := NewService(repo)
	slots, err := svc.GenerateSlots(context.Background(), profID, targetDate)
	if err != nil {
		t.Fatalf("erro ao gerar slots: %v", err)
	}

	// Devem sobrar 2 slots livres (08-09 e 10-11)
	if len(slots) != 2 {
		t.Fatalf("esperava 2 slots disponiveis, obteve %d", len(slots))
	}

	slot1StartLocal := slots[0].StartsAt.In(loc)
	if slot1StartLocal.Hour() != 8 {
		t.Errorf("primeiro slot deveria comecar as 08h, obteve %d", slot1StartLocal.Hour())
	}

	slot2StartLocal := slots[1].StartsAt.In(loc)
	if slot2StartLocal.Hour() != 10 {
		t.Errorf("segundo slot deveria comecar as 10h, obteve %d", slot2StartLocal.Hour())
	}
}
