#!/usr/bin/env bash
#
# Every check this project has, run in containers.
#
# The rule this script exists to enforce: nothing is installed on the host and
# nothing is run on the host. A checkout is not a toolchain, and a check that
# depends on what the developer happens to have installed is a check that
# passes on one machine and fails on another.
#
# The Go module cache and the npm cache live in named volumes so a second run
# is fast; everything else is rebuilt from scratch.
#
# Usage:
#   scripts/verify-docker.sh              # every check
#   scripts/verify-docker.sh go           # the Go checks only
#   scripts/verify-docker.sh web          # the frontend checks only
#   scripts/verify-docker.sh integration  # the integration tests only

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${ROOT}/docker-compose.test.yml"
COMPOSE_PROJECT="agent-shaker-test"

GO_IMAGE="${GO_IMAGE:-golang:1.26-alpine}"
LINT_IMAGE="${LINT_IMAGE:-golangci/golangci-lint:v2.15.0}"
NODE_IMAGE="${NODE_IMAGE:-node:22-slim}"

GO_CACHE_VOLUME="agent-shaker-go-cache"
GO_MOD_CACHE_VOLUME="agent-shaker-go-mod-cache"
NPM_CACHE_VOLUME="agent-shaker-npm-cache"

log() { printf '\n\033[1m▸ %s\033[0m\n' "$*"; }
fail() { printf '\n\033[31m✗ %s\033[0m\n' "$*" >&2; exit 1; }

# run_go executes a shell command inside the Go container with the repository
# mounted and the module caches attached.
run_go() {
  docker run --rm \
    -v "${ROOT}:/repo" \
    -v "${GO_CACHE_VOLUME}:/root/.cache/go-build" \
    -v "${GO_MOD_CACHE_VOLUME}:/go/pkg/mod" \
    -w /repo \
    -e CGO_ENABLED=1 \
    -e GOFLAGS=-mod=mod \
    "${GO_IMAGE}" \
    sh -c "apk add --no-cache gcc musl-dev >/dev/null && $*"
}

run_lint() {
  docker run --rm \
    -v "${ROOT}:/repo" \
    -v "${GO_CACHE_VOLUME}:/root/.cache/go-build" \
    -v "${GO_MOD_CACHE_VOLUME}:/go/pkg/mod" \
    -w /repo \
    "${LINT_IMAGE}" \
    golangci-lint run "$@"
}

run_web() {
  # The frontend is copied into the container rather than bind-mounted.
  #
  # `npm ci` deletes and recreates node_modules, so a bind mount would write
  # that tree into the checkout — and a Go test walking ./... at the same
  # moment would trip over a directory that vanished under it. The copy keeps
  # the host tree untouched.
  docker run --rm \
    -v "${ROOT}/web:/src:ro" \
    -v "${NPM_CACHE_VOLUME}:/root/.npm" \
    -w /app \
    "${NODE_IMAGE}" \
    sh -c "cp -r /src/. /app/ && $*"
}

check_go() {
  log "gofmt"
  # The check runs inside the container because gofmt's output depends on the
  # Go version, and the version is part of the toolchain this project pins.
  run_go "gofmt -l \$(find cmd internal tests -name '*.go') | tee /tmp/unformatted | (! grep . /tmp/unformatted)" \
    || fail "files need gofmt"

  log "go vet"
  run_go "go vet ./..." || fail "go vet found problems"

  log "go build"
  run_go "go build ./..." || fail "the build failed"

  log "golangci-lint"
  run_lint || fail "golangci-lint found problems"

  log "go test"
  # The package list is explicit rather than ./...: the Go module has no
  # packages under web/, and ./... walks web/node_modules when it exists —
  # which is both slow and a race with any npm run in the same tree.
  run_go "go test -count=1 ./cmd/... ./internal/... ./tests/..." || fail "the tests failed"

  log "go test -race"
  run_go "go test -race -count=1 ./cmd/... ./internal/... ./tests/..." || fail "the race detector found problems"
}

check_web() {
  log "web typecheck"
  run_web "npm ci --no-audit --no-fund && npm run typecheck" || fail "the frontend typecheck failed"

  log "web test"
  run_web "npm ci --no-audit --no-fund && npm test" || fail "the frontend tests failed"
}

check_integration() {
  log "integration stack"
  docker compose -p "${COMPOSE_PROJECT}" -f "${COMPOSE_FILE}" up -d --wait postgres \
    || fail "the test database did not start"

  trap 'docker compose -p "${COMPOSE_PROJECT}" -f "${COMPOSE_FILE}" down -v --remove-orphans >/dev/null 2>&1 || true' EXIT

  log "migrations"
  docker compose -p "${COMPOSE_PROJECT}" -f "${COMPOSE_FILE}" run --rm migrate \
    || fail "the migrations did not apply"

  log "integration tests"
  docker run --rm \
    -v "${ROOT}:/repo" \
    -v "${GO_CACHE_VOLUME}:/root/.cache/go-build" \
    -v "${GO_MOD_CACHE_VOLUME}:/go/pkg/mod" \
    -w /repo \
    -e CGO_ENABLED=1 \
    -e DATABASE_URL="postgres://mcp:test-only@postgres:5432/agent_shaker_test?sslmode=disable" \
    --network "${COMPOSE_PROJECT}_test" \
    "${GO_IMAGE}" \
    sh -c "apk add --no-cache gcc musl-dev >/dev/null && go test -tags=integration -count=1 ./..." \
    || fail "the integration tests failed"
}

case "${1:-all}" in
  go)          check_go ;;
  web)         check_web ;;
  integration) check_integration ;;
  all)
    check_go
    check_web
    check_integration
    ;;
  *)
    fail "unknown target: $1 (expected go, web, integration or all)"
    ;;
esac

log "all checks passed"
