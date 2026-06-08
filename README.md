# reshka-backend

Go-бэкенд для PMS «Орёл и Решка». Gin + pgx/sqlx + Postgres.

## Требования

- Go 1.22+
- Docker + docker compose
- (опц.) golang-migrate CLI: `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`

## Быстрый старт

```bash
cp .env.example .env
make up           # поднять Postgres
make migrate      # применить миграции
make seed         # сидинг свойств/номеров/тарифов/админа
make seed-photos  # перенос фото из ../reshka-frontend/src/photos
make run          # запуск API на :8080
```

## Структура

- `cmd/server` — HTTP-сервер
- `cmd/seed` — сидинг данных (повторяемо)
- `cmd/seed-photos` — копирование фото с фронта
- `internal/auth` — bcrypt, JWT, middleware
- `internal/httpapi` — роутер, handlers, DTO
- `internal/repo` — SQL-запросы
- `internal/service` — бизнес-логика
- `migrations/` — SQL-миграции
- `static/photos/` — фото, отдаются по `/photos/...`

## Сценарий разработки

1. Меняешь код
2. `make run` (или `air` если установлен)
3. Тесты: `make test`
