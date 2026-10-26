# --- Variables ---
-include .env

# --- Commands ---
.PHONY: build dev run race test test-race fmt vet lint docker-up docker-down reset-docker help

help:
	@echo "Usage: make [target]"
	@echo "Targets:"
	@echo "  build         Build the binary"
	@echo "  dev           Run the stack with live reload (Air, foreground)"
	@echo "  run           Run the built binary"
	@echo "  race          Run the API locally with the race detector"
	@echo "  test          Run all tests"
	@echo "  test-race     Run all tests with the race detector"
	@echo "  fmt           Format all Go code"
	@echo "  vet           Check code for common mistakes"
	@echo "  lint          Run golangci-lint"
	@echo "  docker-up     Start docker stack (detached)"
	@echo "  docker-down   Stop docker stack"
	@echo "  reset-docker  Wipe volumes and rebuild from zero"

build:
	go build -o bin/app ./cmd/api

dev:
	docker compose up --build

run:
	./bin/app

race:
	go run -race ./cmd/api

test:
	go test ./...

test-race:
	go test -race ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

docker-up:
	docker compose up -d

docker-down:
	docker compose down

reset-docker:
	docker compose down -v
	docker compose up --build -d