# Makefile for Carina

.PHONY: build run test lint migrate migrate-down migrate-create clean

build:
	go build -o bin/api ./cmd/api

run:
	go run ./cmd/api

test:
	go test -v ./...

lint:
	golangci-lint run ./...

migrate:
	./scripts/migrate.sh up

migrate-down:
	./scripts/migrate.sh down

migrate-create:
	@read -p "Migration name: " name; \
	./scripts/migrate.sh create $$name

clean:
	rm -rf bin/