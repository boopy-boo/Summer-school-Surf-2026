.PHONY: run build test lint tidy swagger docker-up docker-down migrate-up

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

swagger:
	swag init -g ./cmd/api/main.go -o ./docs

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v

migrate-up:
	docker compose exec -T postgres psql -U postgres -d pottery -a -f /migrations/001_init.up.sql