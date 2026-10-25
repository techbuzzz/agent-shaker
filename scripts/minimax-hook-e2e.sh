#!/usr/bin/env bash
#
# Proves the MiniMax Code → Agent Shaker bridge.
#
# It brings up the throwaway stack, then drives the hook script the way
# MiniMax Code would: the payload on stdin, the environment in place. The
# script is ours, so proving it is proving the part we own — it reads the
# payload, resolves the agent, renews the lease and pushes the compacted
# transcript.
#
# The hook itself stays inert until an operator enables it in
# Settings → Hooks → Project; proving that needs the host.

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

echo "▸ running the bridge proof"
docker run --rm \
  --network "${PROJECT}_test" \
  -v "${ROOT}:/repo:ro" \
  -w /repo \
  -e AGENT_SHAKER_URL=http://api:8080 \
  node:22-slim node scripts/minimax-hook-e2e.mjs

echo "✓ the MiniMax Code hook bridge is proven"
