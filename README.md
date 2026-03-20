# Persboard

Admin dashboard skeleton for team lead managers.

## Stack

- `frontend`: Vue 3 + TypeScript + Vite + Nginx
- `backend`: Go HTTP API
- `db`: PostgreSQL 16

## Quick Start

```bash
docker-compose up --build
```

Services:

- Frontend: `http://localhost:5173`
- Backend health: `http://localhost:8080/api/health`
- Backend metrics: `http://localhost:8080/api/v1/dashboard/metrics`
- Org structure: `http://localhost:8080/api/v1/org-structure`
- People stats: `http://localhost:8080/api/v1/people/stats`
- PostgreSQL: `localhost:5432` (`persboard/persboard`)

## Notes

- Frontend uses `/api/*` and proxies requests to `backend:8080` inside Docker network.
- Backend checks PostgreSQL availability in `/api/health`.
- Metrics are SQL-backed (`total teams`, `active people`).
- Seed data for teams/people is created by `backend/migrations/001_init.sql` on fresh Postgres volume.
- Calendar metric weights table is created by `backend/migrations/002_calendar_metric_weights.sql`.
- If you already started the project before migrations, run:
  - `docker-compose down -v`
  - `docker-compose up --build`

## API Write Examples

Create team:

```bash
curl -X POST http://localhost:8080/api/v1/teams \
  -H "Content-Type: application/json" \
  -d '{"name":"Data Team"}'
```

Create person:

```bash
curl -X POST http://localhost:8080/api/v1/people \
  -H "Content-Type: application/json" \
  -d '{"fullName":"Nikita Sokolov","role":"Data Analyst","velocity":70,"isActive":true,"teamId":1}'
```

## Calendar Metrics (EazyBI)

Backend exposes:

- `GET /api/v1/calendar/metrics?from=YYYY-MM-DD&to=YYYY-MM-DD`
- `PUT /api/v1/calendar/metric-weights` with body:
  `{"metricKey":"metric-1","weight":2.5}`

Если `EAZYBI_BASE_URL` и `CALENDAR_METRICS_JSON` не настроены, значения для календаря подставляются mock-данными (UI всё равно работает).

Чтобы включить реальные значения из self-hosted EazyBI:

- `EAZYBI_BASE_URL` (например `https://jira.example.com`)
- `EAZYBI_EXPORT_PREFIX` (для Data Center обычно `/plugins/servlet/eazybi`)
- `EAZYBI_ACCOUNT_ID`
- `EAZYBI_AUTH_MODE` (`basic` ожидается для self-hosted)
- `EAZYBI_USERNAME`, `EAZYBI_PASSWORD`
- `EAZYBI_ALLOWED_HOSTS` (comma-separated; требуется из-за SSRF защиты)
- `CALENDAR_METRICS_JSON` (JSON массив с определениями метрик)

Пример `CALENDAR_METRICS_JSON` (3 метрики, включая негативную):

```json
[
  {
    "key": "tickets",
    "title": "Tickets",
    "defaultWeight": 1,
    "metricType": "positive",
    "targetOperator": "gt",
    "targetValue": { "number": 100 },
    "eazybiReportId": 123,
    "eazybiFormat": "csv",
    "timeMemberFormat": "[Time].[%s]"
  },
  {
    "key": "storyPoints",
    "title": "Story Points",
    "defaultWeight": 0.5,
    "metricType": "neutral",
    "targetOperator": "eq",
    "targetValue": { "number": 50 },
    "eazybiReportId": 456,
    "eazybiFormat": "csv",
    "timeMemberFormat": "[Time].[%s]"
  },
  {
    "key": "defectsPerSprint",
    "title": "Defects / sprint",
    "defaultWeight": 1,
    "metricType": "negative",
    "targetOperator": "lt",
    "targetValue": { "number": 5 },
    "eazybiReportId": 789,
    "eazybiFormat": "csv",
    "timeMemberFormat": "[Time].[%s]"
  }
]
```
