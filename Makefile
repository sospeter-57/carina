.PHONY: build run test lint migrate

build:
	go build -o bin/api ./cmd/api

run:
	go run ./cmd/api

test:
	go test ./...

migrate:
	bash scripts/migrate.sh

lint:
	golangci-lint run ./...
