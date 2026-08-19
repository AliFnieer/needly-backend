# Needly Backend — Makefile

.PHONY: build run test test-cover lint seed docker-up docker-down clean help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build both binaries
	CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/needly-api ./cmd/api
	CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/needly-seed ./cmd/seed

run: ## Run the API server locally
	go run ./cmd/api

test: ## Run tests with race detector
	go test -race ./...

test-cover: ## Run tests with coverage report
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint: ## Run golangci-lint
	golangci-lint run --timeout=5m

seed: ## Seed the database with demo data
	go run ./cmd/seed

docker-up: ## Start local dev environment (Postgres + Redis + API)
	docker compose up --build -d

docker-down: ## Stop local dev environment
	docker compose down

docker-logs: ## Follow API logs
	docker compose logs -f api

clean: ## Remove build artifacts
	rm -rf bin/ coverage.out coverage.html
