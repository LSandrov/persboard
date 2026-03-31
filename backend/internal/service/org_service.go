package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"persboard/backend/internal/domain"
)

type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

type NotFoundError struct {
	Message string
}

func (e NotFoundError) Error() string {
	return e.Message
}

type OrgService struct {
	repo domain.Repository
}

func NewOrgService(repo domain.Repository) *OrgService {
	return &OrgService{repo: repo}
}

func (s *OrgService) Health(ctx context.Context) (map[string]string, error) {
	if err := s.repo.Ping(ctx); err != nil {
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

func (s *OrgService) DashboardMetrics(ctx context.Context) (domain.DashboardResponse, error) {
	totalTeams, err := s.repo.CountTeams(ctx)
	if err != nil {
		return domain.DashboardResponse{}, err
	}

	activePeople, err := s.repo.CountActivePeople(ctx)
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

func (s *OrgService) PeopleStats(ctx context.Context) (domain.PersonStats, error) {
	return s.repo.GetPeopleStats(ctx)
}

func (s *OrgService) OrgStructure(ctx context.Context) (domain.OrgStructureResponse, error) {
	teams, err := s.repo.GetTeams(ctx)
	if err != nil {
		return domain.OrgStructureResponse{}, err
	}

	people, err := s.repo.GetPeople(ctx)
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

func (s *OrgService) CreateTeam(ctx context.Context, input domain.CreateTeamInput) (int, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 120 {
		return 0, ValidationError{Message: "name is required and must be <= 120 chars"}
	}
	return s.repo.CreateTeam(ctx, input)
}

func (s *OrgService) CreatePerson(ctx context.Context, input domain.CreatePersonInput) (int, error) {
	input.FullName = strings.TrimSpace(input.FullName)
	input.Role = strings.TrimSpace(input.Role)

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

	return s.repo.CreatePerson(ctx, input)
}

func (s *OrgService) UpdateTeam(ctx context.Context, id int, input domain.UpdateTeamInput) error {
	if id <= 0 {
		return ValidationError{Message: "id must be positive"}
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 120 {
		return ValidationError{Message: "name is required and must be <= 120 chars"}
	}

	if err := s.repo.UpdateTeam(ctx, id, input); err != nil {
		if err == domain.ErrNotFound {
			return NotFoundError{Message: "team not found"}
		}
		return err
	}

	return nil
}

func (s *OrgService) DeleteTeam(ctx context.Context, id int) error {
	if id <= 0 {
		return ValidationError{Message: "id must be positive"}
	}

	if err := s.repo.DeleteTeam(ctx, id); err != nil {
		if err == domain.ErrNotFound {
			return NotFoundError{Message: "team not found"}
		}
		return err
	}

	return nil
}

func (s *OrgService) UpdatePerson(ctx context.Context, id int, input domain.UpdatePersonInput) error {
	if id <= 0 {
		return ValidationError{Message: "id must be positive"}
	}

	input.FullName = strings.TrimSpace(input.FullName)
	input.Role = strings.TrimSpace(input.Role)

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

	if err := s.repo.UpdatePerson(ctx, id, input); err != nil {
		if err == domain.ErrNotFound {
			return NotFoundError{Message: "person not found"}
		}
		return err
	}

	return nil
}

func (s *OrgService) DeletePerson(ctx context.Context, id int) error {
	if id <= 0 {
		return ValidationError{Message: "id must be positive"}
	}

	if err := s.repo.DeletePerson(ctx, id); err != nil {
		if err == domain.ErrNotFound {
			return NotFoundError{Message: "person not found"}
		}
		return err
	}

	return nil
}
