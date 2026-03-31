package org

import (
	"context"
	"strconv"
	"strings"
	"time"

	"persboard/backend/internal/domain"
)

type UseCase struct {
	repo domain.Repository
}

func NewUseCase(repo domain.Repository) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) Health(ctx context.Context) (map[string]string, error) {
	if err := u.repo.Ping(ctx); err != nil {
		return map[string]string{
			"status": "degraded",
			"db":     "down",
		}, err
	}

	return map[string]string{
		"status": "ok",
		"db":     "up",
	}, nil
}

func (u *UseCase) DashboardMetrics(ctx context.Context) (domain.DashboardResponse, error) {
	totalTeams, err := u.repo.CountTeams(ctx)
	if err != nil {
		return domain.DashboardResponse{}, err
	}

	activePeople, err := u.repo.CountActivePeople(ctx)
	if err != nil {
		return domain.DashboardResponse{}, err
	}

	return domain.DashboardResponse{
		UpdatedAt: time.Now().UTC(),
		Metrics: []domain.Metric{
			{
				Key:   "total-teams",
				Title: "Total Teams",
				Value: strconv.Itoa(totalTeams),
				Trend: "Org structure coverage",
			},
			{
				Key:   "active-people",
				Title: "Active People",
				Value: strconv.Itoa(activePeople),
				Trend: "Contributors currently active",
			},
		},
	}, nil
}

func (u *UseCase) PeopleStats(ctx context.Context) (domain.PersonStats, error) {
	return u.repo.GetPeopleStats(ctx)
}

func (u *UseCase) OrgStructure(ctx context.Context) (domain.OrgStructureResponse, error) {
	teams, err := u.repo.GetTeams(ctx)
	if err != nil {
		return domain.OrgStructureResponse{}, err
	}

	people, err := u.repo.GetPeople(ctx)
	if err != nil {
		return domain.OrgStructureResponse{}, err
	}

	peopleByTeam := make(map[int][]domain.Person, len(teams))
	for _, person := range people {
		peopleByTeam[person.TeamID] = append(peopleByTeam[person.TeamID], person)
	}

	for i := range teams {
		teams[i].Members = peopleByTeam[teams[i].ID]
	}

	return domain.OrgStructureResponse{
		UpdatedAt: time.Now().UTC(),
		Teams:     teams,
	}, nil
}

func (u *UseCase) CreateTeam(ctx context.Context, input domain.CreateTeamInput) (int, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 120 {
		return 0, ValidationError{Message: "name is required and must be <= 120 chars"}
	}
	return u.repo.CreateTeam(ctx, input)
}

func (u *UseCase) UpdateTeam(ctx context.Context, id int, input domain.UpdateTeamInput) error {
	if id <= 0 {
		return ValidationError{Message: "id must be positive"}
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 120 {
		return ValidationError{Message: "name is required and must be <= 120 chars"}
	}

	if err := u.repo.UpdateTeam(ctx, id, input); err != nil {
		if err == domain.ErrNotFound {
			return NotFoundError{Message: "team not found"}
		}
		return err
	}

	return nil
}

func (u *UseCase) DeleteTeam(ctx context.Context, id int) error {
	if id <= 0 {
		return ValidationError{Message: "id must be positive"}
	}

	if err := u.repo.DeleteTeam(ctx, id); err != nil {
		if err == domain.ErrNotFound {
			return NotFoundError{Message: "team not found"}
		}
		return err
	}

	return nil
}

func (u *UseCase) CreatePerson(ctx context.Context, input domain.CreatePersonInput) (int, error) {
	input.FullName = strings.TrimSpace(input.FullName)
	input.Role = strings.TrimSpace(input.Role)
	input.BirthDate = normalizeDatePointer(input.BirthDate)
	input.EmploymentDate = normalizeDatePointer(input.EmploymentDate)

	switch {
	case input.FullName == "" || len(input.FullName) > 120:
		return 0, ValidationError{Message: "fullName is required and must be <= 120 chars"}
	case input.Role == "" || len(input.Role) > 60:
		return 0, ValidationError{Message: "role is required and must be <= 60 chars"}
	case input.TeamID <= 0:
		return 0, ValidationError{Message: "teamId must be positive"}
	case input.Velocity < 0 || input.Velocity > 100:
		return 0, ValidationError{Message: "velocity must be between 0 and 100"}
	}
	if err := validateDatePointer(input.BirthDate, "birthDate"); err != nil {
		return 0, err
	}
	if err := validateDatePointer(input.EmploymentDate, "employmentDate"); err != nil {
		return 0, err
	}

	return u.repo.CreatePerson(ctx, input)
}

func (u *UseCase) UpdatePerson(ctx context.Context, id int, input domain.UpdatePersonInput) error {
	if id <= 0 {
		return ValidationError{Message: "id must be positive"}
	}

	input.FullName = strings.TrimSpace(input.FullName)
	input.Role = strings.TrimSpace(input.Role)
	input.BirthDate = normalizeDatePointer(input.BirthDate)
	input.EmploymentDate = normalizeDatePointer(input.EmploymentDate)

	switch {
	case input.FullName == "" || len(input.FullName) > 120:
		return ValidationError{Message: "fullName is required and must be <= 120 chars"}
	case input.Role == "" || len(input.Role) > 60:
		return ValidationError{Message: "role is required and must be <= 60 chars"}
	case input.TeamID <= 0:
		return ValidationError{Message: "teamId must be positive"}
	case input.Velocity < 0 || input.Velocity > 100:
		return ValidationError{Message: "velocity must be between 0 and 100"}
	}
	if err := validateDatePointer(input.BirthDate, "birthDate"); err != nil {
		return err
	}
	if err := validateDatePointer(input.EmploymentDate, "employmentDate"); err != nil {
		return err
	}

	if err := u.repo.UpdatePerson(ctx, id, input); err != nil {
		if err == domain.ErrNotFound {
			return NotFoundError{Message: "person not found"}
		}
		return err
	}

	return nil
}

func normalizeDatePointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func validateDatePointer(value *string, field string) error {
	if value == nil {
		return nil
	}
	if _, err := time.Parse("2006-01-02", *value); err != nil {
		return ValidationError{Message: field + " must be in YYYY-MM-DD format"}
	}
	return nil
}

func (u *UseCase) DeletePerson(ctx context.Context, id int) error {
	if id <= 0 {
		return ValidationError{Message: "id must be positive"}
	}

	if err := u.repo.DeletePerson(ctx, id); err != nil {
		if err == domain.ErrNotFound {
			return NotFoundError{Message: "person not found"}
		}
		return err
	}

	return nil
}
