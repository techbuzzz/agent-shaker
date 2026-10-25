#!/usr/bin/env bash
#
# Proves the idempotency keys on the mutating MCP tools.
#
# The proof is behavioural: a retry with the same key replays the first
# answer, and the task is claimed exactly once. The same key on a different
# task executes for real, because the key identifies one logical operation and
# the request's fingerprint says which one.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${ROOT}/docker-compose.test.yml"
PROJECT="agent-shaker-test"

compose() { docker compose -p "${PROJECT}" -f "${COMPOSE_FILE}" "$@"; }

cleanup() {
  compose down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "▸ building the image"
compose build api migrate

echo "▸ starting the throwaway database"
compose up -d --wait postgres

echo "▸ applying the migrations"
compose run --rm migrate

echo "▸ starting the API"
compose up -d --wait api

echo "▸ running the proof"
docker run --rm \
  --network "${PROJECT}_test" \
  -v "${ROOT}:/repo:ro" \
  -w /repo \
  -e AGENT_SHAKER_URL=http://api:8080 \
  node:22-slim node scripts/idempotency-e2e.mjs

echo "✓ a retry does not claim a task twice"
