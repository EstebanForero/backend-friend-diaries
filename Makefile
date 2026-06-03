.PHONY: dev build run test tidy lint migrate-up migrate-down infra-up infra-down

-include .env
export

dev:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

tidy:
	go mod tidy

test:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down 1

infra-up:
	docker compose -f ../docker-compose-infra.yaml up -d

infra-down:
	docker compose -f ../docker-compose-infra.yaml down
