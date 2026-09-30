.PHONY: help build run test test-race cover vet lint fmt migrate-up migrate-down clean docker-build docker-up docker-down docker-logs demo deps air check

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Compile the server binary
	go build -o bin/mcp-server ./cmd/server

run: build ## Build and run the server locally
	./bin/mcp-server

test: ## Run all unit tests
	go test -count=1 ./...

test-race: ## Run all tests with the race detector (requires cgo)
	go test -race -count=1 ./...

test-integration: ## Run integration tests against a real Postgres (DATABASE_URL required)
	DATABASE_URL=$$DATABASE_URL go test -tags=integration -count=1 ./...

cover: ## Run tests with coverage and open the HTML report
	go test -count=1 -coverprofile=cover.out ./...
	go tool cover -html=cover.out -o cover.html
	@echo "Coverage report: cover.html"

vet: ## Run go vet
	go vet ./...

fmt: ## Format Go source files
	gofmt -w .

lint: ## Run golangci-lint (installs on demand via go run)
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run || true

migrate-up: ## Apply database migrations using cmd/migrate (DATABASE_URL required)
	go run ./cmd/migrate -cmd up

migrate-down: ## Roll back one migration
	go run ./cmd/migrate -cmd down -steps 1

migrate-version: ## Print current migration version
	go run ./cmd/migrate -cmd version

migrate-force: ## Force the migration version (use with care)
	go run ./cmd/migrate -cmd force -version $(VERSION)

air: ## Run the server with hot reload (requires github.com/cosmtrek/air)
	go run github.com/cosmtrek/air@latest -c .air.toml

clean: ## Remove build artefacts and the cover profile
	rm -rf bin/ cover.out cover.html

docker-build: ## Build the production Docker image
	docker build -t mcp-task-tracker .

docker-up: ## Start services via docker-compose
	docker compose up -d

docker-down: ## Stop services via docker-compose
	docker compose down

docker-logs: ## Tail docker-compose logs
	docker compose logs -f

demo: ## Run the demo script (requires running services)
	./demo.sh

deps: ## Download Go module dependencies
	go mod download

check: ## Probe the local server and report status
	@curl -sf http://localhost:8080/healthz >/dev/null && echo "✓ server /healthz OK" || echo "✗ server /healthz not reachable"
	@curl -sf http://localhost:8080/readyz >/dev/null && echo "✓ server /readyz OK" || echo "✗ server /readyz reports not ready"
	@curl -sf http://localhost:8080/metrics >/dev/null && echo "✓ /metrics OK" || echo "✗ /metrics not reachable"
