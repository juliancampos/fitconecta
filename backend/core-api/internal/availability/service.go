package availability

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	CreateRule(ctx context.Context, profID uuid.UUID, req *CreateRuleRequest) (*AvailabilityRule, error)
	ListRules(ctx context.Context, profID uuid.UUID) ([]AvailabilityRule, error)
	DeleteRule(ctx context.Context, profID, ruleID uuid.UUID) error

	CreateException(ctx context.Context, profID uuid.UUID, req *CreateExceptionRequest) (*AvailabilityException, error)
	ListExceptions(ctx context.Context, profID uuid.UUID, from, to time.Time) ([]AvailabilityException, error)
	DeleteException(ctx context.Context, profID, excID uuid.UUID) error

	GenerateSlots(ctx context.Context, profID uuid.UUID, date time.Time) ([]Slot, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateRule(ctx context.Context, profID uuid.UUID, req *CreateRuleRequest) (*AvailabilityRule, error) {
	if req.Weekday < 0 || req.Weekday > 6 {
		return nil, fmt.Errorf("dia da semana invalido (0-6)")
	}
	if req.SlotMinutes <= 0 {
		req.SlotMinutes = 60
	}
	return s.repo.CreateRule(ctx, profID, req)
}

func (s *service) ListRules(ctx context.Context, profID uuid.UUID) ([]AvailabilityRule, error) {
	return s.repo.ListRules(ctx, profID)
}

func (s *service) DeleteRule(ctx context.Context, profID, ruleID uuid.UUID) error {
	return s.repo.DeleteRule(ctx, profID, ruleID)
}

func (s *service) CreateException(ctx context.Context, profID uuid.UUID, req *CreateExceptionRequest) (*AvailabilityException, error) {
	if !req.StartsAt.Before(req.EndsAt) {
		return nil, fmt.Errorf("o horario inicial da excecao deve ser anterior ao final")
	}
	return s.repo.CreateException(ctx, profID, req)
}

func (s *service) ListExceptions(ctx context.Context, profID uuid.UUID, from, to time.Time) ([]AvailabilityException, error) {
	return s.repo.ListExceptions(ctx, profID, from, to)
}

func (s *service) DeleteException(ctx context.Context, profID, excID uuid.UUID) error {
	return s.repo.DeleteException(ctx, profID, excID)
}

func (s *service) GenerateSlots(ctx context.Context, profID uuid.UUID, targetDate time.Time) ([]Slot, error) {
	tzName, err := s.repo.GetProfessionalTimezone(ctx, profID)
	if errors.Is(err, ErrProfessionalUnavailable) {
		return nil, err
	}
	if err != nil {
		tzName = "America/Sao_Paulo"
	}

	loc, err := time.LoadLocation(tzName)
	if err != nil {
		loc = time.UTC
	}

	// Extrai a data de calendario desejada e constroi meia-noite no fuso do profissional
	year, month, day := targetDate.Date()
	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, loc)
	endOfDay := startOfDay.Add(24 * time.Hour)

	weekday := int(startOfDay.Weekday())

	// Buscar regras para o dia da semana
	rules, err := s.repo.ListRules(ctx, profID)
	if err != nil {
		return nil, err
	}

	var dayRules []AvailabilityRule
	for _, r := range rules {
		if r.Weekday == weekday {
			dayRules = append(dayRules, r)
		}
	}

	if len(dayRules) == 0 {
		return []Slot{}, nil
	}

	// Buscar excecoes e agendamentos no dia
	exceptions, err := s.repo.ListExceptions(ctx, profID, startOfDay.UTC(), endOfDay.UTC())
	if err != nil {
		return nil, err
	}

	booked, err := s.repo.GetBookedIntervals(ctx, profID, startOfDay.UTC(), endOfDay.UTC())
	if err != nil {
		return nil, err
	}

	var slots []Slot

	for _, rule := range dayRules {
		// Parse horários da regra
		var startHour, startMin, endHour, endMin int
		_, _ = fmt.Sscanf(rule.StartTime, "%d:%d", &startHour, &startMin)
		_, _ = fmt.Sscanf(rule.EndTime, "%d:%d", &endHour, &endMin)

		slotDuration := time.Duration(rule.SlotMinutes) * time.Minute
		curStart := time.Date(year, month, day, startHour, startMin, 0, 0, loc)
		ruleEnd := time.Date(year, month, day, endHour, endMin, 0, 0, loc)

		for curStart.Add(slotDuration).Before(ruleEnd) || curStart.Add(slotDuration).Equal(ruleEnd) {
			curEnd := curStart.Add(slotDuration)

			slotStartUTC := curStart.UTC()
			slotEndUTC := curEnd.UTC()

			// Verifica se conflita com alguma excecao (folga, feriado)
			hasConflict := false
			for _, exc := range exceptions {
				if slotStartUTC.Before(exc.EndsAt) && slotEndUTC.After(exc.StartsAt) {
					hasConflict = true
					break
				}
			}

			// Verifica se conflita com algum agendamento existente
			if !hasConflict {
				for _, b := range booked {
					if slotStartUTC.Before(b.EndsAt) && slotEndUTC.After(b.StartsAt) {
						hasConflict = true
						break
					}
				}
			}

			if !hasConflict {
				slots = append(slots, Slot{
					StartsAt:  slotStartUTC,
					EndsAt:    slotEndUTC,
					Available: true,
				})
			}

			curStart = curEnd
		}
	}

	if slots == nil {
		slots = []Slot{}
	}

	return slots, nil
}
