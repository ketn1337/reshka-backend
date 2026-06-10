.PHONY: up down logs migrate seed seed-photos import-bnovo import-bnovo-discover import-bnovo-wipe run test fmt vet build hooks lint

DB_URL ?= postgres://reshka:reshka@localhost:5432/reshka?sslmode=disable

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

migrate:
	migrate -path migrations -database "$(DB_URL)" up

seed:
	go run ./cmd/seed

seed-photos:
	go run ./cmd/seed-photos

run:
	go run ./cmd/server

dev:
	go run ./cmd/server

import-bnovo: ## импорт броней из Bnovo PMS (--mode=all|import|wipe|discover, --config=PATH)
	go run ./cmd/import-bnovo --config=./bnovo-rooms.json

import-bnovo-discover: ## дамп уникальных bnovo_room_id из Bnovo в JSON
	go run ./cmd/import-bnovo --config=./bnovo-rooms.json --mode=discover

import-bnovo-wipe: ## wipe (--dry-run чтобы посмотреть)
	go run ./cmd/import-bnovo --config=./bnovo-rooms.json --mode=wipe

import-bnovo-dry: ## dry-run полного импорта
	go run ./cmd/import-bnovo --config=./bnovo-rooms.json --mode=all --dry-run

test:
	go test ./...

fmt:
	gofmt -s -w .

vet:
	go vet ./...

build:
	go build -o bin/server ./cmd/server
	go build -o bin/seed ./cmd/seed
	go build -o bin/seed-photos ./cmd/seed-photos
	go build -o bin/import-bnovo ./cmd/import-bnovo

# Включает локальные git-хуки из .githooks/ через core.hooksPath.
hooks:
	git config core.hooksPath .githooks

# Локальные проверки без записи файлов. То же самое делает pre-commit хук.
lint:
	@output=$$(gofmt -l .); \
	if [ -n "$$output" ]; then \
		echo "Files not gofmt-clean:"; \
		echo "$$output"; \
		exit 1; \
	fi
	go vet ./...
