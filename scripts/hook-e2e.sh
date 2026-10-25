#!/usr/bin/env bash
#
# End-to-end proof of the hook layer.
#
# It runs the real server against a throwaway database, binds a hook to a
# task lifecycle event, completes the task, and checks that the run log shows
# the command's own captured output.
#
# A hook that only ever ran in a unit test would not have proved that the
# broker, the dispatcher, the runner and the run log are wired to each other.
# This does.
#
# Everything happens in containers, in the `agent-shaker-test` compose project,
# so nothing here can reach a developer's running stack.

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

echo "▸ starting the API with the hook layer enabled"
compose up -d --wait api

echo "▸ running the proof"
docker run --rm \
  --network "${PROJECT}_test" \
  -v "${ROOT}/scripts:/scripts:ro" \
  -e AGENT_SHAKER_URL=http://api:8080 \
  -e AGENT_SHAKER_API_KEY=e2e-only \
  node:22-slim node /scripts/hook-e2e.mjs

echo "✓ the hook layer is proven end to end"
