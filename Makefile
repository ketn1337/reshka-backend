.PHONY: up down logs migrate seed seed-photos run test fmt vet build

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
