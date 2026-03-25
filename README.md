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
docker-compose up --build
```

Сервисы:
- Frontend: `http://localhost:5173`
- Backend health: `http://localhost:8080/api/health`
- PostgreSQL: `localhost:5432`

## Данные PostgreSQL
Данные БД хранятся в bind-mount `./.docker/postgres-data`.

Поэтому:
- обычные перезапуски/пересоздания контейнеров обычно не обнуляют БД
- для полного сброса удалите `./.docker/postgres-data`

## Инициализация и миграции
При первом старте (или после удаления `./.docker/postgres-data`) выполняются:
- `backend/migrations/001_init.sql` — seed-данные команд/людей
- `backend/migrations/002_calendar_metric_weights.sql` — таблица весов календарных метрик

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
- `POST /api/v1/people` (body: `{ "fullName":"...", "role":"...", "velocity":70, "isActive":true, "teamId":1, "teamLeadId": 2 }`, поле `teamLeadId` можно опустить или передать `null`)

### Календарные метрики
- `GET /api/v1/calendar/metrics?from=YYYY-MM-DD&to=YYYY-MM-DD`
- `PUT /api/v1/calendar/metric-weights` (body: `{ "metricKey":"tickets", "weight":2.5 }`)

Пояснение:
- окно дат ограничено 1..31 день
- если `from/to` не переданы, сервер берёт период по умолчанию (последние ~6 дней)

## EazyBI (опционально)
Если настроены `EAZYBI_BASE_URL`, `EAZYBI_ACCOUNT_ID` и `EAZYBI_EXPORT_PREFIX`, бэкенд будет пытаться получать данные из EazyBI.

Если конфигурация отсутствует/неполная — используются mock-значения, чтобы UI продолжал работать.

Важно из-за SSRF-защиты:
- `EAZYBI_ALLOWED_HOSTS` — allowlist hostname (comma-separated). Должен включать hostname из `EAZYBI_BASE_URL`.

Основные переменные:
- `EAZYBI_AUTH_MODE` (`basic` ожидается для self-hosted; есть режим `embed`)
- `EAZYBI_USERNAME`, `EAZYBI_PASSWORD` (для `basic`)
- `EAZYBI_EMBED_TOKEN` (для `embed`)
- `CALENDAR_METRICS_JSON` — JSON-массив определений метрик (если оставить пустым, используется дефолт)

## Заметки по разработке
- Frontend проксирует запросы к бэкенду внутри Docker-сети, используя `VITE_API_BASE_URL`.
- Бэкенд декодирует JSON с `DisallowUnknownFields()`: лишние поля в request body вернут ошибку.
- При включении EazyBI используется валидация URL (разрешённые хосты + блокировка private/loopback/link-local IP).
