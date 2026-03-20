CREATE TABLE IF NOT EXISTS teams (
  id SERIAL PRIMARY KEY,
  name VARCHAR(120) NOT NULL UNIQUE,
  lead_id INTEGER NULL
);

CREATE TABLE IF NOT EXISTS people (
  id SERIAL PRIMARY KEY,
  full_name VARCHAR(120) NOT NULL,
  role VARCHAR(60) NOT NULL,
  velocity NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (velocity >= 0 AND velocity <= 100),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  team_lead_id INTEGER NULL REFERENCES people(id) ON DELETE SET NULL
);

INSERT INTO teams (name)
VALUES
  ('Platform Team'),
  ('Frontend Team')
ON CONFLICT (name) DO NOTHING;

INSERT INTO people (full_name, role, velocity, is_active, team_id)
SELECT 'Ivan Petrov', 'Team Lead', 78.5, TRUE, t.id
FROM teams t
WHERE t.name = 'Platform Team'
  AND NOT EXISTS (SELECT 1 FROM people p WHERE p.full_name = 'Ivan Petrov');

INSERT INTO people (full_name, role, velocity, is_active, team_id)
SELECT 'Anna Smirnova', 'Team Lead', 81.2, TRUE, t.id
FROM teams t
WHERE t.name = 'Frontend Team'
  AND NOT EXISTS (SELECT 1 FROM people p WHERE p.full_name = 'Anna Smirnova');

UPDATE teams t
SET lead_id = p.id
FROM people p
WHERE (
    t.name = 'Platform Team' AND p.full_name = 'Ivan Petrov'
  ) OR (
    t.name = 'Frontend Team' AND p.full_name = 'Anna Smirnova'
  );

INSERT INTO people (full_name, role, velocity, is_active, team_id, team_lead_id)
SELECT 'Max Orlov', 'Backend Developer', 64.0, TRUE, t.id, t.lead_id
FROM teams t
WHERE t.name = 'Platform Team'
  AND NOT EXISTS (SELECT 1 FROM people p WHERE p.full_name = 'Max Orlov');

INSERT INTO people (full_name, role, velocity, is_active, team_id, team_lead_id)
SELECT 'Olga Romanova', 'QA Engineer', 59.0, TRUE, t.id, t.lead_id
FROM teams t
WHERE t.name = 'Platform Team'
  AND NOT EXISTS (SELECT 1 FROM people p WHERE p.full_name = 'Olga Romanova');

INSERT INTO people (full_name, role, velocity, is_active, team_id, team_lead_id)
SELECT 'Dmitry Kuznetsov', 'Frontend Developer', 73.5, TRUE, t.id, t.lead_id
FROM teams t
WHERE t.name = 'Frontend Team'
  AND NOT EXISTS (SELECT 1 FROM people p WHERE p.full_name = 'Dmitry Kuznetsov');

INSERT INTO people (full_name, role, velocity, is_active, team_id, team_lead_id)
SELECT 'Elena Volkova', 'Product Designer', 66.8, FALSE, t.id, t.lead_id
FROM teams t
WHERE t.name = 'Frontend Team'
  AND NOT EXISTS (SELECT 1 FROM people p WHERE p.full_name = 'Elena Volkova');
