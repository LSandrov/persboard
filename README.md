# Persboard

Persboard — каркас админ-панели для руководителей команд: структура организации, статистика людей и календарные метрики.

## Цели проекта
- Быстрый старт fullstack-проекта (Frontend + Backend + PostgreSQL) в Docker.
- Пример API с валидацией JSON, таймаутами и предсказуемыми ответами.
- Опциональная интеграция календарных метрик с self-hosted EazyBI.

## Стек
- `frontend`: Vue 3 + TypeScript + Vite + Nginx
- `backend`: Go HTTP API
- `db`: PostgreSQL 16

## Развертка

### Требования
- Docker + Docker Compose

### 1) Конфигурация окружения
Перед запуском `docker-compose` используйте корневой файл `.env`.

1. Скопируйте `./.env.example` -> `./.env`
2. Заполните значения (включая секреты для EazyBI, если вы хотите реальные данные).

`./.env` добавлен в `.gitignore`, чтобы секреты не попадали в репозиторий.

Ссылки по технологиям:
- Docker Compose: https://docs.docker.com/compose/
- Vue 3: https://vuejs.org/
- Go `net/http`: https://pkg.go.dev/net/http
- PostgreSQL: https://www.postgresql.org/docs/

### 2) Запуск
```bash
make init
make up
```

Сервисы:
- Frontend: `http://localhost:5173`
- Backend health: `http://localhost:8080/api/health`
- gRPC: `localhost:9090`
- PostgreSQL: `localhost:5432`

## Данные PostgreSQL
Данные БД хранятся в bind-mount `./.docker/postgres-data`.

Поэтому:
- обычные перезапуски/пересоздания контейнеров обычно не обнуляют БД
- для полного сброса удалите `./.docker/postgres-data`

## Логи backend
Файлы пишутся в `./.docker/logs` (в контейнере это `LOG_DIR`, по умолчанию `.docker/logs`, см. `docker-compose.yml` volume):

- `access.log` — одна JSON-строка на HTTP-запрос (метод, путь, query, статус, длительность, `X-Request-ID`).
- `app.log` — JSON-логи приложения (`slog`, включая ошибки `500` и предупреждения EazyBI).

Переменные:

- `LOG_DIR` — каталог для логов (`""` отключает запись в файлы, остаётся stderr).
- `LOG_DEBUG=true` или `LOG_LEVEL=debug` — включить `Debug`-логи по API handlers.

## Инициализация и миграции
При первом старте (или после удаления `./.docker/postgres-data`) выполняются:
- `backend/migrations/001_init.sql` — базовые таблицы команд/людей (без mock seed)
- `backend/migrations/002_calendar_metric_weights.sql` — таблица весов календарных метрик
- `backend/migrations/004_people_birth_and_employment_dates.sql` — поля даты рождения и трудоустройства

Если нужно вернуться к “чистой” базе, удалите `./.docker/postgres-data` и запустите проект заново.

## API (backend)
Все ответы — JSON.

### Health
- `GET /api/health`

### Дашборд
- `GET /api/v1/dashboard/metrics`

### Организация и люди
- `GET /api/v1/org-structure`
- `GET /api/v1/people/stats`
- `POST /api/v1/teams` (body: `{ "name": "Data Team", "leadId": 1 }`, поле `leadId` можно опустить или передать `null`)
- `PUT /api/v1/teams/{id}` (body: `{ "name": "Data Team", "leadId": 1 }`)
- `DELETE /api/v1/teams/{id}`
- `POST /api/v1/people` (body: `{ "fullName":"...", "role":"...", "velocity":70, "isActive":true, "teamId":1, "teamLeadId":2, "birthDate":"1990-06-15", "employmentDate":"2024-01-10" }`; `teamLeadId/birthDate/employmentDate` можно опустить или передать `null`)
- `PUT /api/v1/people/{id}` (body: `{ "fullName":"...", "role":"...", "velocity":70, "isActive":true, "teamId":1, "teamLeadId":2, "birthDate":"1990-06-15", "employmentDate":"2024-01-10" }`)
- `DELETE /api/v1/people/{id}`

### Календарные метрики
- `GET /api/v1/calendar/metrics?from=YYYY-MM-DD&to=YYYY-MM-DD`
- `PUT /api/v1/calendar/metric-weights` (body: `{ "metricKey":"tickets", "weight":2.5 }`)

Пояснение:
- окно дат ограничено 1..31 день
- если `from/to` не переданы, сервер берёт период по умолчанию (последние ~6 дней)

## EazyBI (опционально)
Если настроены `EAZYBI_BASE_URL`, `EAZYBI_ACCOUNT_ID` и `EAZYBI_EXPORT_PREFIX`, бэкенд будет пытаться получать данные из EazyBI.

Если конфигурация отсутствует/неполная — используются mock-значения, чтобы UI продолжал работать.
Если экспорт/парсинг EazyBI для метрики завершился ошибкой (сеть, токен, формат CSV, SSRF/IP и т.п.) — для этой метрики подставляются mock-значения, запрос к API не падает с `500`.

Важно из-за SSRF-защиты:
- `EAZYBI_ALLOWED_HOSTS` — allowlist hostname (comma-separated). Должен включать hostname из `EAZYBI_BASE_URL`.

Основные переменные:
- `EAZYBI_AUTH_MODE` (`jira_token` рекомендован; поддержаны `basic` и `embed`)
- `EAZYBI_JIRA_TOKEN` (для `jira_token`)
- `EAZYBI_USERNAME`, `EAZYBI_PASSWORD` (для `basic`)
- `EAZYBI_EMBED_TOKEN` (для `embed`)
- `CALENDAR_METRICS_JSON` — JSON-массив определений метрик (если оставить пустым, используется дефолт)

## Заметки по разработке
- Frontend проксирует запросы к бэкенду внутри Docker-сети, используя `VITE_API_BASE_URL`.
- Бэкенд декодирует JSON с `DisallowUnknownFields()`: лишние поля в request body вернут ошибку.
- При включении EazyBI используется валидация URL (разрешённые хосты + блокировка private/loopback/link-local IP).
- Контракты API вынесены в `backend/api/proto/persboard/v1/org.proto`.
- Для генерации gRPC и grpc-gateway кода: `make proto` (требуется установленный `buf`).
- Оркестрация бизнес-логики по организации вынесена в `backend/internal/usecase/org`.
- HTTP API для org/teams/people/calendar обслуживается через `grpc-gateway`, который проксирует вызовы в gRPC-сервис.
