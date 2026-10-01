# Agent Shaker — developer entry points.
#
# `make` is the supported way to build, test and run this project; CI runs the
# same commands. Individual tool invocations are an implementation detail.
#
# Quick start:
#   make check          # verify the toolchain and formatting
#   make dev            # run the Go API with hot reload
#   make web-dev        # run the Nuxt frontend with hot reload
#   make up             # full production-like stack via docker compose

SHELL := /bin/bash
.DEFAULT_GOAL := help

# Build metadata, stamped into the binary and reported on the "build" log line.
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -s -w \
              -X main.version=$(VERSION) \
              -X main.commit=$(COMMIT) \
              -X main.buildTime=$(BUILD_TIME)

GO_FILES   := $(shell find cmd internal tests -name '*.go' 2>/dev/null)
WEB_DIR    := web

.PHONY: help check fmt fmt-check vet lint build run clean \
        test test-race test-integration cover \
        web-install web-build web-typecheck web-dev \
        migrate-up migrate-version migrate-force \
        docker-build docker-up docker-down docker-logs docker-config \
        caddy-validate caddy-fmt caddy-fmt-check \
        edge-up edge-down \
        dev demo deps

help: ## Show this help message
	@echo 'Agent Shaker — available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ''
	@echo 'Stack: Postgres -> Go API (:8080) -> Nuxt SSR (:3000, single public origin)'
	@echo 'Edge:  make edge-up  (adds Caddy: TLS + basic auth + MCP/A2A on one origin)'

# --------------------------------------------------------------------- checks

check: fmt-check vet test ## Run the full pre-commit gate (fmt, vet, tests)
	@echo "✓ check passed"

fmt: ## Format Go source files (normalises to LF)
	gofmt -w .

fmt-check: ## Fail if any Go file needs gofmt
	@out=$$(gofmt -l $(dir $(GO_FILES))); \
	if [ -n "$$out" ]; then \
		echo "✗ files need gofmt:"; echo "$$out"; exit 1; \
	fi
	@echo "✓ gofmt clean"

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint v2 (fails on findings)
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.0 run

test: ## Run all unit tests
	go test -count=1 ./...

test-race: ## Run all tests under the race detector
	go test -race -count=1 ./...

test-integration: ## Run integration-tagged tests (DATABASE_URL required)
	@test -n "$$DATABASE_URL" || { echo "DATABASE_URL is required"; exit 1; }
	go test -tags=integration -count=1 ./...

cover: ## Generate the coverage report
	go test -count=1 -coverprofile=cover.out ./...
	go tool cover -html=cover.out -o cover.html
	@echo "Coverage report: cover.html"

# ------------------------------------------------------------------ go build

build: ## Compile the server binary into bin/ with build metadata
	@mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/mcp-server ./cmd/server
	@echo "✓ bin/mcp-server  ($(VERSION) / $(COMMIT))"

run: build ## Build and run the API in the foreground
	./bin/mcp-server

dev: ## Run the API with hot reload (requires air: go install github.com/cosmtrek/air@latest)
	go run github.com/cosmtrek/air@latest

# ----------------------------------------------------------------- web build

web-install: ## Install frontend dependencies from the lockfile
	cd $(WEB_DIR) && npm ci

web-build: web-install ## Build the Nuxt SSR bundle
	cd $(WEB_DIR) && npm run build

web-typecheck: web-install ## Typecheck the frontend (gates CI)
	cd $(WEB_DIR) && npm run typecheck

web-dev: web-install ## Run the Nuxt dev server with HMR
	cd $(WEB_DIR) && npm run dev

# ---------------------------------------------------------------- migrations

migrate-up: ## Apply all pending migrations (DATABASE_URL required)
	go run ./cmd/migrate -cmd up

migrate-version: ## Print the current schema version
	go run ./cmd/migrate -cmd version

migrate-force: ## Force the schema version (recover from a half-applied migration)
	go run ./cmd/migrate -cmd force -version $(VERSION)

# -------------------------------------------------------------------- docker

docker-build: ## Build both production images
	docker build -t agent-shaker-api:local \
		--build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_TIME=$(BUILD_TIME) .
	docker build -t agent-shaker-web:local -f $(WEB_DIR)/Dockerfile $(WEB_DIR)

docker-config: ## Validate the compose file, both with and without the tls profile
	@POSTGRES_PASSWORD=ci-only API_KEYS=ci-only docker compose config --quiet
	@POSTGRES_PASSWORD=ci-only API_KEYS=ci-only BASIC_AUTH_USER=ci BASIC_AUTH_HASH=ci-only \
		docker compose --profile tls config --quiet
	@echo "✓ compose config valid (default + tls profiles)"

# The Caddyfile is a production input, so it gets the same treatment as Go
# source: a validator and a formatter, both runnable locally and in CI.
#
# BASIC_AUTH_HASH is generated here rather than read from a .env file on
# purpose. Docker Compose interpolates $$VAR inside env files, which silently
# eats the $$<salt> segment of a bcrypt hash — see docs/DEPLOYMENT.md. Exporting
# a literal value is the form that survives.
CADDY_IMAGE ?= caddy:2-alpine
CADDYFILE   ?= deploy/Caddyfile

caddy-validate: ## Validate deploy/Caddyfile
	@H=$$(docker run --rm $(CADDY_IMAGE) caddy hash-password --plaintext 'validate-only' 2>/dev/null); \
	docker run --rm -e BASIC_AUTH_USER=validate -e BASIC_AUTH_HASH="$$H" -e PUBLIC_HOST=localhost \
		-v "$(CURDIR)/$(CADDYFILE):/etc/caddy/Caddyfile:ro" \
		$(CADDY_IMAGE) caddy validate --config /etc/caddy/Caddyfile
	@echo "✓ Caddyfile valid"

caddy-fmt: ## Rewrite deploy/Caddyfile in canonical form
	@H=$$(docker run --rm $(CADDY_IMAGE) caddy hash-password --plaintext 'fmt-only' 2>/dev/null); \
	id=$$(docker create $(CADDY_IMAGE)); \
	docker cp "$(CURDIR)/$(CADDYFILE)" "$$id:/in"; \
	docker start -a "$$id" > /dev/null; \
	docker exec "$$id" cp /in /tmp/Caddyfile; \
	docker exec "$$id" caddy fmt --overwrite /tmp/Caddyfile; \
	docker cp "$$id:/tmp/Caddyfile" "$(CURDIR)/$(CADDYFILE)"; \
	docker rm -f "$$id" > /dev/null
	@echo "✓ Caddyfile formatted"

caddy-fmt-check: ## Fail if deploy/Caddyfile is not caddy-fmt clean
	@H=$$(docker run --rm $(CADDY_IMAGE) caddy hash-password --plaintext 'fmt-only' 2>/dev/null); \
	docker run --rm -e BASIC_AUTH_USER=validate -e BASIC_AUTH_HASH="$$H" -e PUBLIC_HOST=localhost \
		-v "$(CURDIR)/$(CADDYFILE):/etc/caddy/Caddyfile:ro" \
		$(CADDY_IMAGE) caddy fmt --diff /etc/caddy/Caddyfile > /dev/null
	@echo "✓ Caddyfile format clean"

docker-up: ## Start the full stack (Postgres + API + web)
	docker compose up -d --build
	@echo "→ UI    http://localhost:$${WEB_PORT:-3000}"
	@echo "→ API   http://localhost:8080"

# The TLS edge is opt-in. BASIC_AUTH_HASH must be exported in the shell, not
# placed in .env — see docs/DEPLOYMENT.md.
edge-up: ## Start the stack with the TLS edge (requires exported BASIC_AUTH_USER + BASIC_AUTH_HASH)
	@test -n "$$BASIC_AUTH_USER" || { echo "BASIC_AUTH_USER must be exported (see docs/DEPLOYMENT.md)"; exit 1; }
	@test -n "$$BASIC_AUTH_HASH" || { echo "BASIC_AUTH_HASH must be exported (see docs/DEPLOYMENT.md)"; exit 1; }
	docker compose --profile tls up -d --build
	@echo "→ https://$${PUBLIC_HOST:-localhost}"

edge-down: ## Stop the TLS-edge stack and remove its volumes
	docker compose --profile tls down -v

docker-down: ## Stop the stack
	docker compose down

docker-logs: ## Tail compose logs
	docker compose logs -f

# ---------------------------------------------------------------------- misc

deps: ## Download Go module dependencies
	go mod download

demo: ## Run the demo script against a running stack
	./demo.sh

clean: ## Remove build artefacts and coverage output
	rm -rf bin/ cover.out cover.html
	rm -rf $(WEB_DIR)/.output $(WEB_DIR)/.nuxt $(WEB_DIR)/dist
