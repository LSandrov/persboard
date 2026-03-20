package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"persboard/backend/internal/domain"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *Repository) CountTeams(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM teams").Scan(&count)
	return count, err
}

func (r *Repository) CountActivePeople(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM people WHERE is_active = TRUE").Scan(&count)
	return count, err
}

func (r *Repository) GetPeopleStats(ctx context.Context) (domain.PersonStats, error) {
	stats := domain.PersonStats{}
	err := r.db.QueryRowContext(
		ctx,
		`SELECT
			COUNT(*) AS total_people,
			COUNT(*) FILTER (WHERE is_active = TRUE) AS active_people,
			COALESCE(AVG(velocity) FILTER (WHERE is_active = TRUE), 0)
		 FROM people`,
	).Scan(&stats.TotalPeople, &stats.ActivePeople, &stats.AverageVelocity)
	return stats, err
}

func (r *Repository) GetTeams(ctx context.Context) ([]domain.Team, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, name, lead_id
		 FROM teams
		 ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	teams := make([]domain.Team, 0)
	for rows.Next() {
		var team domain.Team
		if err := rows.Scan(&team.ID, &team.Name, &team.LeadID); err != nil {
			return nil, err
		}
		team.Members = []domain.Person{}
		teams = append(teams, team)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return teams, nil
}

func (r *Repository) GetPeople(ctx context.Context) ([]domain.Person, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, full_name, role, velocity, is_active, team_id, team_lead_id
		 FROM people
		 ORDER BY team_id, id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	people := make([]domain.Person, 0)
	for rows.Next() {
		var person domain.Person
		if err := rows.Scan(
			&person.ID,
			&person.FullName,
			&person.Role,
			&person.Velocity,
			&person.IsActive,
			&person.TeamID,
			&person.TeamLeadID,
		); err != nil {
			return nil, err
		}
		people = append(people, person)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return people, nil
}

func (r *Repository) CreateTeam(ctx context.Context, input domain.CreateTeamInput) (int, error) {
	var id int
	err := r.db.QueryRowContext(
		ctx,
		`INSERT INTO teams (name, lead_id)
		 VALUES ($1, $2)
		 RETURNING id`,
		input.Name,
		input.LeadID,
	).Scan(&id)
	return id, err
}

func (r *Repository) CreatePerson(ctx context.Context, input domain.CreatePersonInput) (int, error) {
	var id int
	err := r.db.QueryRowContext(
		ctx,
		`INSERT INTO people (full_name, role, velocity, is_active, team_id, team_lead_id)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		input.FullName,
		input.Role,
		input.Velocity,
		input.IsActive,
		input.TeamID,
		input.TeamLeadID,
	).Scan(&id)
	return id, err
}

func (r *Repository) UpsertMetricWeights(ctx context.Context, defs []domain.CalendarMetricDefinition) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, def := range defs {
		_, err := tx.ExecContext(
			ctx,
			`INSERT INTO eazybi_metric_weights (metric_key, title, weight)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (metric_key) DO UPDATE
			   SET title = EXCLUDED.title,
			       updated_at = NOW()`,
			def.Key,
			def.Title,
			def.DefaultWeight,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) GetMetricWeights(ctx context.Context, keys []string) (map[string]domain.MetricWeight, error) {
	if len(keys) == 0 {
		return map[string]domain.MetricWeight{}, nil
	}

	args := make([]any, 0, len(keys))
	placeholders := make([]string, 0, len(keys))
	for i, k := range keys {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		args = append(args, k)
	}

	q := `SELECT metric_key, title, weight
	      FROM eazybi_metric_weights
		  WHERE metric_key IN (` + stringsJoin(placeholders, ",") + `)`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]domain.MetricWeight, len(keys))
	for rows.Next() {
		var w domain.MetricWeight
		if err := rows.Scan(&w.Key, &w.Title, &w.Weight); err != nil {
			return nil, err
		}
		out[w.Key] = w
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) SetMetricWeight(ctx context.Context, input domain.UpdateMetricWeightInput, title string) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO eazybi_metric_weights (metric_key, title, weight)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (metric_key) DO UPDATE
		   SET title = EXCLUDED.title,
		       weight = EXCLUDED.weight,
		       updated_at = NOW()`,
		input.MetricKey,
		title,
		input.Weight,
	)
	return err
}

// stringsJoin is a tiny helper to avoid importing strings just for Join.
func stringsJoin(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += sep + parts[i]
	}
	return out
}
