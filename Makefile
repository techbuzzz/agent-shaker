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
# The directories holding them, de-duplicated. `$(dir $(GO_FILES))` alone emits
# one entry per *file*, so the argument list grows with the codebase and a
# 100-file tree passes the same 20 directories fifty times over. `sort` also
# makes the gofmt invocation stable, which keeps CI logs readable.
GO_DIRS    := $(sort $(dir $(GO_FILES)))
WEB_DIR    := web

.PHONY: help check check-all fmt fmt-check vet lint build run clean \
        test test-race test-integration cover \
        web-install web-build web-typecheck web-test web-dev \
        migrate-up migrate-version migrate-force \
        db-backup db-backups db-restore \
        docker-build docker-up docker-down docker-logs docker-config \
        caddy-validate caddy-fmt caddy-fmt-check \
        edge-up edge-down deploy \
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

# The whole gate, including the frontend. `check` stays Go-only so it stays fast
# enough to run on every save; this is the one to run before pushing.
check-all: check web-typecheck web-test ## Full gate: Go + frontend typecheck + frontend tests
	@echo "✓ check-all passed"

fmt: ## Format Go source files (normalises to LF)
	gofmt -w .

fmt-check: ## Fail if any Go file needs gofmt
	@out=$$(gofmt -l $(GO_DIRS)); \
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

web-test: web-install ## Run the frontend unit tests (gates CI)
	cd $(WEB_DIR) && npm test

web-dev: web-install ## Run the Nuxt dev server with HMR
	cd $(WEB_DIR) && npm run dev

# ---------------------------------------------------------------- migrations

migrate-up: ## Apply all pending migrations (DATABASE_URL required)
	go run ./cmd/migrate -cmd up

migrate-version: ## Print the current schema version
	go run ./cmd/migrate -cmd version

# MIGRATION_VERSION, not VERSION. VERSION is build metadata and expands to
# something like `v0.3.5-68-g85cbbf7-dirty`, which the -version int flag
# cannot parse — so this target used to fail on every invocation, including
# the one case it exists for: recovering from a half-applied migration.
MIGRATION_VERSION ?=

migrate-force: ## Force the recorded schema version. MIGRATION_VERSION=8
	@test -n "$(MIGRATION_VERSION)" || { echo "usage: make migrate-force MIGRATION_VERSION=8"; exit 2; }
	@test "$(MIGRATION_VERSION)" -eq "$(MIGRATION_VERSION)" 2>/dev/null \
		|| { echo "MIGRATION_VERSION must be an integer (got '$(MIGRATION_VERSION)')"; exit 2; }
	go run ./cmd/migrate -cmd force -version $(MIGRATION_VERSION)

# ------------------------------------------------------------------- backups
#
# A live deployment holds real project, agent and task data, and
# `docker compose down -v` deletes all of it. These targets are the only
# supported way in or out of that state.
#
# The dump is written to the HOST, never into the container: a file on the
# postgres volume dies with the volume it was meant to protect you from.
# `backups/` is git-ignored, because a dump is the entire dataset in plaintext
# (base64 inside -Fc, which is not encryption).

BACKUP_DIR ?= backups
PG_SERVICE  ?= postgres

db-backup: ## Dump the database to backups/<timestamp>.dump (host-local, git-ignored)
	@mkdir -p $(BACKUP_DIR)
	@ts=$$(date -u +%Y%m%dT%H%M%SZ); \
	file=$(BACKUP_DIR)/agent-shaker-$$ts.dump; \
	echo "→ dumping to $$file"; \
	docker compose exec -T $(PG_SERVICE) \
		pg_dump -U "$${POSTGRES_USER:-mcp}" -d "$${POSTGRES_DB:-mcp_tracker}" -Fc > "$$file"; \
	# A truncated dump is worse than no dump: it fails at restore time, when
	# you are already out of options. Fail here instead.
	if [ ! -s "$$file" ]; then echo "✗ dump is empty"; exit 1; fi
	sha256sum "$$file" > "$$file.sha256"
	@ls -lh "$$file" | awk '{print "✓ $$5  " $$9}'
	@echo "  checksum: $$(cut -d" " -f1 < "$$file.sha256")"

db-backups: ## List local backups, newest first
	@ls -1t $(BACKUP_DIR)/*.dump 2>/dev/null || { echo "no backups in $(BACKUP_DIR)/"; exit 0; }

# Restoring REPLACES the database contents. It is not a merge. The guard is
# deliberate friction: a restore typed with the wrong filename, or run twice
# by accident, is how an otherwise good backup becomes the thing that lost the
# data.
db-restore: ## Replace the database from a dump. FILE=backups/<name>.dump
	@test -n "$(FILE)" || { echo "usage: make db-restore FILE=backups/<name>.dump"; exit 2; }
	@test -f "$(FILE)" || { echo "✗ no such file: $(FILE)"; exit 1; }
	# Compare the hash of the file being restored against the recorded one.
	#
	# Deliberately NOT `sha256sum -c`: that verifies whatever FILENAME the
	# .sha256 file names, not the file passed to this target. Move or rename a
	# dump and it either errors confusingly or, worse, cheerfully verifies a
	# different, intact file and lets a corrupt one through. Comparing the two
	# hashes directly is unambiguous.
	@if [ -f "$(FILE).sha256" ]; then \
		want=$$(cut -d' ' -f1 < "$(FILE).sha256" | tr -d '*[:space:]'); \
		got=$$(sha256sum "$(FILE)" | cut -d' ' -f1 | tr -d '*[:space:]'); \
		if [ -n "$$want" ] && [ "$$want" = "$$got" ]; then \
			echo "✓ checksum verified"; \
		else \
			echo "✗ checksum mismatch — refusing to restore a corrupt dump"; \
			echo "  expected $$want"; \
			echo "  actual   $$got"; \
			exit 1; \
		fi; \
	else \
		echo "! no .sha256 alongside $(FILE); restoring unverified"; \
	fi
	@echo "→ this DROPS and recreates the schema and all data in the database."
	@echo "→ press Ctrl-C now to abort."
	@sleep 3
	docker compose exec -T $(PG_SERVICE) \
		psql -U "$${POSTGRES_USER:-mcp}" -d "$${POSTGRES_DB:-mcp_tracker}" \
		-v ON_ERROR_STOP=1 -c 'DROP SCHEMA public CASCADE; CREATE SCHEMA public;' >/dev/null
	docker compose cp - $(PG_SERVICE):/tmp/restore.dump < "$(FILE)"
	docker compose exec -T $(PG_SERVICE) \
		pg_restore -U "$${POSTGRES_USER:-mcp}" -d "$${POSTGRES_DB:-mcp_tracker}" \
		--no-owner --no-privileges /tmp/restore.dump
	docker compose exec -T $(PG_SERVICE) rm -f /tmp/restore.dump
	@echo "✓ restored from $(FILE)"

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

# The full first-time bootstrap for a real host: checks the ports, resolves and
# records secrets, builds, starts, then verifies the live public origin. The
# script rather than a Makefile recipe because it prompts, loops and prints a
# report — all of which read as noise in a stack of shell lines.
deploy: ## Bootstrap the production stack on this host. HOST=example.com [GENERATE=1]
	@test -n "$(HOST)" || { echo "usage: make deploy HOST=example.com"; exit 2; }
	bash scripts/deploy.sh --host $(HOST) $(if $(GENERATE),--generate,)

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
