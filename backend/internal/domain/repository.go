package domain

import "context"

type Repository interface {
	Ping(ctx context.Context) error
	CountTeams(ctx context.Context) (int, error)
	CountActivePeople(ctx context.Context) (int, error)
	GetPeopleStats(ctx context.Context) (PersonStats, error)
	GetTeams(ctx context.Context) ([]Team, error)
	GetPeople(ctx context.Context) ([]Person, error)
	CreateTeam(ctx context.Context, input CreateTeamInput) (int, error)
	CreatePerson(ctx context.Context, input CreatePersonInput) (int, error)
	UpsertMetricWeights(ctx context.Context, defs []CalendarMetricDefinition) error
	GetMetricWeights(ctx context.Context, keys []string) (map[string]MetricWeight, error)
	SetMetricWeight(ctx context.Context, input UpdateMetricWeightInput, title string) error
}
