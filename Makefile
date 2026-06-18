.PHONY: build run test test-integration lint migrate mocks swagger tidy docker-up docker-down

BINARY      := bin/server
PKG         := ./...
MIGRATIONS  := ./migrations
DB_DSN      ?=

build:
	go build -o $(BINARY) ./cmd/server

run: build
	./$(BINARY)

test:
	go test -race -count=1 $(shell go list ./... | grep -v /test/integration)

test-integration:
	go test -race -count=1 -tags=integration ./test/integration/...

lint:
	golangci-lint run

migrate:
	@test -n "$(DB_DSN)" || { echo "DB_DSN is required, e.g. make migrate DB_DSN=postgres://..."; exit 1; }
	goose -dir $(MIGRATIONS) postgres "$(DB_DSN)" up

migrate-down:
	@test -n "$(DB_DSN)" || { echo "DB_DSN is required, e.g. make migrate-down DB_DSN=postgres://..."; exit 1; }
	goose -dir $(MIGRATIONS) postgres "$(DB_DSN)" down

mocks:
	go generate ./...

swagger:
	@echo "Swagger spec is hand-written at api/openapi.yaml and served at /swagger"

tidy:
	go mod tidy

up:
	docker compose -f deployments/docker-compose.yml up --build

down:
	docker compose -f deployments/docker-compose.yml down -v
