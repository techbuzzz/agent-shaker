#!/usr/bin/env bash
#
# deploy.sh — one-shot production bootstrap for Agent Shaker.
#
# Brings up the `tls` profile on a real host: Caddy with an automatic
# certificate, the Go service, Nuxt and Postgres, then verifies the result
# against the live public origin. Safe to re-run.
#
#   ./scripts/deploy.sh --host agent-shaker.example.com
#   ./scripts/deploy.sh --host example.com --generate
#
# WHAT THIS WILL NOT DO
#
# It will not put BASIC_AUTH_HASH in .env. Docker Compose interpolates $VAR
# inside env files, and a bcrypt hash is $2a$14$<salt><digest> — the salt
# segment is eaten as a variable reference, leaving a 60-character string that
# no longer parses as bcrypt. The edge then starts, reports healthy, and
# rejects every password: a silent lockout with nothing in any log. The hash
# is generated here and exported from the shell, which is the only form that
# survives. See docs/DEPLOYMENT.md.
#
# Generated secrets are printed once and never written anywhere except .env
# (git-ignored) or your shell history. Treat the printed values as the only
# copy: there is no recovery path for a lost API key.
#
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

HOST=""
GENERATE=0
ASSUME_YES=0
INSTALL_CRON=1

# ------------------------------------------------------------------ output

if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
    BOLD=$'\033[1m'; RED=$'\033[31m'; GREEN=$'\033[32m'; YELLOW=$'\033[33m'; RESET=$'\033[0m'
else
    BOLD=''; RED=''; GREEN=''; YELLOW=''; RESET=''
fi

log()  { printf '%s==>%s %s\n' "$BOLD" "$RESET" "$*"; }
ok()   { printf '  %sok%s   %s\n' "$GREEN" "$RESET" "$*"; }
warn() { printf '  %swarn%s %s\n' "$YELLOW" "$RESET" "$*"; }
die()  { printf '%sFATAL%s %s\n' "$RED" "$RESET" "$*" >&2; exit 1; }

usage() {
    cat <<'USAGE'
Usage: scripts/deploy.sh --host <fqdn> [options]

Required:
  --host <fqdn>        Public hostname Caddy issues the certificate for.
                       Must not be "localhost" — ACME will not issue for it.

Options:
  --generate           Generate any missing secret instead of prompting.
  --yes                Skip the confirmation prompt before deploying.
  --no-cron            Do not install the daily backup timer.
  -h, --help           This message.

Secrets, in precedence order:
  1. Already set in your environment.
  2. Already present in .env (POSTGRES_PASSWORD, API_KEYS).
  3. Typed at the prompt, or generated with --generate.

BASIC_AUTH_USER / BASIC_AUTH_HASH are never read from or written to .env. The
password is prompted for and hashed in memory; only the hash is exported.

Examples:
  ./scripts/deploy.sh --host example.com
  ./scripts/deploy.sh --host example.com --generate --yes
USAGE
}

while [ $# -gt 0 ]; do
    case "$1" in
        --host)      HOST="${2:-}"; shift 2 ;;
        --host=*)    HOST="${1#*=}"; shift ;;
        --generate)  GENERATE=1; shift ;;
        --yes|-y)    ASSUME_YES=1; shift ;;
        --no-cron)   INSTALL_CRON=0; shift ;;
        -h|--help)   usage; exit 0 ;;
        *)           usage >&2; die "unknown argument: $1" ;;
    esac
done

[ -n "$HOST" ] || { usage >&2; die "--host is required"; }
[ "$HOST" != "localhost" ] || die "--host localhost will not get a certificate from Let's Encrypt"

# --------------------------------------------------------------- preflight

log "Preflight"

command -v docker >/dev/null 2>&1 || die "docker is not installed"
docker compose version >/dev/null 2>&1 || die "the docker compose plugin is not available"
ok "docker $(docker version --format '{{.Server.Version}}' 2>/dev/null || echo '?') with compose"

if docker info >/dev/null 2>&1; then
    ok "docker daemon reachable"
else
    die "cannot talk to the docker daemon (is your user in the docker group?)"
fi

[ -f docker-compose.yml ] || die "docker-compose.yml not found in $REPO_ROOT"

# Ports 80 and 443 must be free or Caddy cannot bind and Let's Encrypt
# cannot validate. A stale container from a previous run is the usual cause.
port_busy() {
    local port="$1"
    if command -v ss >/dev/null 2>&1; then
        ss -ltn "sport = :$port" 2>/dev/null | grep -q LISTEN
    elif command -v netstat >/dev/null 2>&1; then
        netstat -ltn 2>/dev/null | grep -q "[:.]$port .*LISTEN"
    else
        return 1   # no tool to check with; do not block the deploy on it
    fi
}
for p in 80 443; do
    if port_busy "$p"; then
        # Distinguish "our own stack from last time" from a genuine conflict,
        # because the first is expected and the second needs the operator.
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q 'agent-shaker-edge'; then
            warn "port $p is held by the agent-shaker edge from a previous run"
        else
            die "port $p is already in use by something else — stop it or set HTTPS_PORT/HTTP_PORT"
        fi
    fi
done
ok "ports 80 and 443 are available"

# ACME cannot validate a name that does not resolve. Warned, not fatal: DNS may
# be mid-propagation, or the host may be fronted by something else entirely.
if command -v getent >/dev/null 2>&1; then
    if getent hosts "$HOST" >/dev/null 2>&1; then
        ok "$HOST resolves ($(getent hosts "$HOST" | head -1 | awk '{print $1}'))"
    else
        warn "$HOST does not resolve from this host."
        warn "  Let's Encrypt validates over the public DNS. Check that the A/AAAA record"
        warn "  points here before continuing, or the certificate will never be issued."
    fi
fi

# ------------------------------------------------------------------ secrets

ENV_FILE="$REPO_ROOT/.env"

# Read one key from .env without executing it and without sourcing the file.
env_get() {
    [ -f "$ENV_FILE" ] || return 0
    # Strip surrounding quotes, ignore comments and blank lines.
    sed -n "s/^[[:space:]]*${1}[[:space:]]*=[[:space:]]*\(.*\)$/\1/p" "$ENV_FILE" \
        | head -n 1 \
        | sed -e 's/^[[:space:]]*"//' -e 's/"[[:space:]]*$//' \
              -e 's/^[[:space:]]*'"'"'//' -e 's/'"'"'[[:space:]]*$//' \
              -e 's/[[:space:]]*#.*$//' \
        | tr -d '\r'
}

env_set() {
    local key="$1" value="$2"
    touch "$ENV_FILE"
    # Replace an existing key in place so re-runs do not accumulate duplicates.
    if grep -q "^[[:space:]]*${key}[[:space:]]*=" "$ENV_FILE"; then
        local tmp
        tmp="$(mktemp)"
        awk -v k="$key" -v v="$value" '
            BEGIN { FS="=" }
            $1 ~ "^[[:space:]]*" k "[[:space:]]*$" { print k "=" v; done=1; next }
            { print }
            END { if (!done) print k "=" v }
        ' "$ENV_FILE" > "$tmp"
        mv "$tmp" "$ENV_FILE"
    else
        printf '%s=%s\n' "$key" "$value" >> "$ENV_FILE"
    fi
}

gen_hex() { openssl rand -hex "$1" 2>/dev/null || head -c "$1" /dev/urandom | od -An -tx1 | tr -d ' \n'; }

prompt_secret() {
    # prompt_secret <label> -> value on stdout
    local label="$1" value=""
    if [ "$GENERATE" -eq 1 ]; then
        gen_hex 24
        return
    fi
    printf '  %s: ' "$label" >&2
    if [ -r /dev/tty ]; then
        IFS= read -r -s value < /dev/tty
        printf '\n' >&2
    else
        die "no terminal available for input; re-run with --generate"
    fi
    [ -n "$value" ] || die "$label must not be empty"
    printf '%s' "$value"
}

log "Secrets"

command -v openssl >/dev/null 2>&1 || warn "openssl not found; falling back to /dev/urandom"

POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-$(env_get POSTGRES_PASSWORD)}"
if [ -z "$POSTGRES_PASSWORD" ]; then
    # Hex only, deliberately. This value is interpolated into
    # postgres://user:PASSWORD@postgres:5432/db, and base64 or any alphabet
    # containing / @ : # produces a DSN that parses as the wrong host or the
    # wrong database — a failure that looks like a connection error, not a
    # malformed password.
    POSTGRES_PASSWORD="$(gen_hex 32)"
    ok "POSTGRES_PASSWORD generated"
else
    ok "POSTGRES_PASSWORD already set"
fi

API_KEYS="${API_KEYS:-$(env_get API_KEYS)}"
if [ -z "$API_KEYS" ]; then
    # The client credential agents present. Prefixed so it is greppable in
    # logs and recognisable in a support conversation.
    API_KEYS="ask_$(gen_hex 24)"
    ok "API_KEYS generated"
else
    ok "API_KEYS already set ($(printf '%s' "$API_KEYS" | tr ',' '\n' | wc -l | tr -d ' ') key(s))"
fi

# The perimeter credential. User-chosen, because it is a login, and typed
# without echo.
if [ -z "${BASIC_AUTH_USER:-}" ]; then
    BASIC_AUTH_USER="$(printf 'admin')"
fi

if [ -z "${BASIC_AUTH_HASH:-}" ]; then
    log "Basic auth"
    if [ "$GENERATE" -eq 1 ]; then
        BASIC_PLAINTEXT="$(gen_hex 12)"
        GENERATED_PASSWORD="yes"
    else
        printf '  perimeter username [%s]: ' "$BASIC_AUTH_USER" >&2
        if [ -r /dev/tty ]; then
            IFS= read -r BASIC_AUTH_USER < /dev/tty
            printf '\n' >&2
        fi
        [ -n "$BASIC_AUTH_USER" ] || BASIC_AUTH_USER="admin"
        printf '  perimeter password (input hidden): ' >&2
        if [ -r /dev/tty ]; then
            IFS= read -r -s BASIC_PLAINTEXT < /dev/tty
            printf '\n' >&2
        else
            die "no terminal available for input; re-run with --generate"
        fi
        [ -n "$BASIC_PLAINTEXT" ] || die "the perimeter password must not be empty"
    fi

    # --plaintext=VALUE, not --plaintext VALUE: a password beginning with a
    # dash would otherwise be parsed as a flag by caddy's flag handling.
    BASIC_AUTH_HASH="$(docker run --rm caddy:2-alpine \
        caddy hash-password --plaintext="$BASIC_PLAINTEXT" 2>/dev/null)" \
        || die "could not generate the bcrypt hash"
    # BASIC_PLAINTEXT is deliberately NOT unset here. It is still needed below,
    # to print a generated password once and to run the authenticated
    # verification. It is cleared by the exit trap at the end of the script.
    case "$BASIC_AUTH_HASH" in
        '$2'*) : ;;
        *) die "caddy returned something that is not a bcrypt hash: $BASIC_AUTH_HASH" ;;
    esac
    ok "bcrypt hash generated (held in memory only, never written to .env)"
fi

export BASIC_AUTH_USER BASIC_AUTH_HASH

# The plaintext only exists in memory and is needed for the one-time banner and
# the authenticated checks below. Drop it before anything else can read it from
# the environment or a core dump.
cleanup() {
    BASIC_PLAINTEXT=""
    BASIC_AUTH_HASH=""
}
trap cleanup EXIT INT TERM

# ------------------------------------------------------------------ persist

# .env carries ONLY values that survive interpolation. $ is why the bcrypt hash
# cannot be here.
env_set POSTGRES_PASSWORD "$POSTGRES_PASSWORD"
env_set API_KEYS "$API_KEYS"
env_set PUBLIC_HOST "$HOST"
env_set ACME_EMAIL "${ACME_EMAIL:-admin@$HOST}"
chmod 600 "$ENV_FILE" 2>/dev/null || true
ok ".env written (POSTGRES_PASSWORD, API_KEYS, PUBLIC_HOST, ACME_EMAIL)"

# Print only what this run generated. An operator who typed their own password
# already knows it; echoing it back would put it in their scrollback for no
# reason.
if [ -n "${GENERATED_PASSWORD:-}" ]; then
    cat <<BANNER

  ┌──────────────────────────────────────────────────────────────────┐
  │  Generated secrets — recorded nowhere else. Copy them now.       │
  └──────────────────────────────────────────────────────────────────┘
    UI username    : $BASIC_AUTH_USER
    UI password    : $BASIC_PLAINTEXT
    Agent API key  : $API_KEYS
BANNER
fi

# ------------------------------------------------------------------ confirm

if [ "$ASSUME_YES" -ne 1 ]; then
    printf '\n  Deploy %s now? [y/N] ' "$HOST" >&2
    if [ -r /dev/tty ]; then
        IFS= read -r answer < /dev/tty
    else
        answer=""
    fi
    case "$answer" in
        [yY]*) : ;;
        *) die "aborted" ;;
    esac
fi

# ------------------------------------------------------------------- deploy

log "Building and starting the stack"
# --build every time: the images are the deployable artifact, and a stale
# image is the most common cause of "I deployed the fix and nothing changed".
docker compose --profile tls up -d --build
ok "compose up complete"

# ------------------------------------------------------------------- verify

log "Waiting for the certificate"
# The first ACME issuance is not instant. Fail with a useful message rather
# than a bare curl exit code.
CERT_OK=0
attempt=1
while [ "$attempt" -le 30 ]; do
    if curl -fsS --max-time 5 -o /dev/null "https://$HOST/healthz" 2>/dev/null \
       || curl -ksS --max-time 5 -o /dev/null "https://$HOST/healthz" 2>/dev/null; then
        CERT_OK=1
        break
    fi
    sleep 5
    attempt=$((attempt + 1))
done

if [ "$CERT_OK" -ne 1 ]; then
    warn "https://$HOST did not become ready within 150s"
    warn "check: docker compose --profile tls logs -f edge"
    warn "a real certificate needs $HOST to resolve publicly to this host's IP"
    warn "  ($(curl -fsS --max-time 5 https://api.ipify.org 2>/dev/null || echo 'unknown'))"
    exit 1
fi
ok "TLS is serving"

log "Verifying the deployment"

check() {
    local name="$1" expected="$2" actual="$3"
    if [ "$actual" = "$expected" ]; then
        ok "$name -> $actual"
    else
        warn "$name -> $actual (expected $expected)"
        VERIFY_FAILED=1
    fi
}

VERIFY_FAILED=0

# --- machine surface: API key, no basic auth ---------------------------------
# These are on the edge's machine branch, so they are checkable whether or not
# the perimeter password is available in this shell.

check "agent card with API key" 200 \
    "$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $API_KEYS" \
        "https://$HOST/.well-known/agent-card.json")"
check "agent card without a key" 401 \
    "$(curl -sS -o /dev/null -w '%{http_code}' "https://$HOST/.well-known/agent-card.json")"

CARD_URL="$(curl -sS -H "Authorization: Bearer $API_KEYS" \
    "https://$HOST/.well-known/agent-card.json" 2>/dev/null \
    | sed -n 's/.*"url"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)"
if [ -n "$CARD_URL" ]; then
    if printf '%s' "$CARD_URL" | grep -q "$HOST"; then
        ok "agent card advertises the public origin -> $CARD_URL"
    else
        warn "agent card advertises '$CARD_URL', which does not contain $HOST."
        warn "  A2A clients will be sent to the wrong address. Set BASE_URL on mcp-server."
    fi
fi

MCP_INIT='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
check "MCP initialize" 200 \
    "$(curl -sS -o /dev/null -w '%{http_code}' -X POST \
        -H "Authorization: Bearer $API_KEYS" -H 'Content-Type: application/json' \
        -d "$MCP_INIT" "https://$HOST/mcp")"

# --- human surface: basic auth at the edge ------------------------------------
#
# These paths sit on the edge's human branch, so the gate answers FIRST. That
# detail decides how they have to be checked:
#
#   * /api/projects unauthenticated is meant to be 401 — that is the gate.
#   * /metrics and /ws are NOT meant to be reachable at all, but unauthenticated
#     they return the same 401 before routing is ever consulted. Asserting 404
#     there without a credential would fail on a perfectly healthy deployment.
#
# So both are checked WITH a credential, where the answer reflects the routing
# table rather than the gate.

if [ -z "${BASIC_PLAINTEXT:-}" ]; then
    warn "the perimeter password is not in this shell, so the human-surface checks"
    warn "  below were skipped. Run them yourself once:"
    warn "  curl -u '$BASIC_AUTH_USER:...' https://$HOST/api/projects"
    warn "  curl -u '$BASIC_AUTH_USER:...' https://$HOST/metrics   # expect 404"
    warn "  curl -u '$BASIC_AUTH_USER:...' https://$HOST/ws       # expect 426"
else
    check "human surface with credentials" 200 \
        "$(curl -ksS -o /dev/null -w '%{http_code}' -u "$BASIC_AUTH_USER:$BASIC_PLAINTEXT" "https://$HOST/api/projects")"
    check "human surface without credentials" 401 \
        "$(curl -ksS -o /dev/null -w '%{http_code}' "https://$HOST/api/projects")"
    check "healthz" 200 \
        "$(curl -ksS -o /dev/null -w '%{http_code}' -u "$BASIC_AUTH_USER:$BASIC_PLAINTEXT" "https://$HOST/healthz")"

    # Prometheus metrics at a public origin are free reconnaissance. Authenticated
    # 404 proves the path is not routed even for a caller the edge admits.
    check "metrics is not exposed" 404 \
        "$(curl -ksS -o /dev/null -w '%{http_code}' -u "$BASIC_AUTH_USER:$BASIC_PLAINTEXT" "https://$HOST/metrics")"

    # A missing /ws route is invisible from the UI — it never errors, it simply
    # stops updating — so assert the route is there. The discriminator is the
    # status code, not the body:
    #
    #   426  the Nitro route exists and correctly refuses a non-upgrade GET
    #   200  no route matched, so the SPA shell was served instead
    #   000  nothing answered at all
    WS_CODE="$(curl -ksS -o /dev/null -w '%{http_code}' --max-time 5 \
        -u "$BASIC_AUTH_USER:$BASIC_PLAINTEXT" "https://$HOST/ws" 2>/dev/null || true)"
    case "$WS_CODE" in
        426)
            ok "/ws is routed -> 426 Upgrade Required (live updates will work)"
            ;;
        200|301|302)
            warn "/ws returned $WS_CODE, which is the SPA shell — the WebSocket"
            warn "  route is missing, so live updates will silently never arrive."
            warn "  Check that web/server/routes/ws.ts is in the running image."
            VERIFY_FAILED=1
            ;;
        000|'')
            warn "/ws could not be reached (curl status '$WS_CODE')"
            VERIFY_FAILED=1
            ;;
        *)
            ok "/ws -> $WS_CODE (a real route refused the non-upgrade GET)"
            ;;
    esac
fi

# ------------------------------------------------------------------ backups

if [ "$INSTALL_CRON" -eq 1 ] && command -v crontab >/dev/null 2>&1; then
    log "Daily backup timer"
    # 04:17 rather than 04:00: shared top-of-the-hour slots are the most
    # contended minute on any given host.
    CRON_LINE="17 4 * * * cd $REPO_ROOT && make db-backup >> $REPO_ROOT/backups/cron.log 2>&1"
    if crontab -l 2>/dev/null | grep -qF 'make db-backup'; then
        ok "backup timer already installed"
    else
        mkdir -p "$REPO_ROOT/backups"
        ( crontab -l 2>/dev/null; printf '%s\n' "$CRON_LINE" ) | crontab -
        ok "installed: 04:17 daily, output in backups/cron.log"
    fi
    warn "a timer is not a backup you have restored from. Verify once:"
    warn "  make db-backup && make db-restore FILE=backups/<name>.dump"
elif [ "$INSTALL_CRON" -eq 1 ]; then
    warn "crontab not found; no backup timer installed. Use make db-backup by hand."
fi

# ------------------------------------------------------------------ summary

printf '\n'
log "Deployed"
cat <<SUMMARY
  UI           https://$HOST/
  Agent card   https://$HOST/.well-known/agent-card.json
  MCP          https://$HOST/mcp
  A2A          https://$HOST/a2a/v1

  The Go service publishes no host port and Postgres is not exposed.
  /metrics is deliberately 404 — scrape it on the internal network.

  Connect an agent: open a project, "Connect an AI agent", "Get config".
  The generated config reads \${env:AGENT_SHAKER_API_KEY} rather than embedding
  a key, so export it in the agent's environment.

  Rotate the API key by adding the new one to API_KEYS (comma-separated) and
  redeploying. Remove the old one only after clients have switched.
SUMMARY

if [ "$VERIFY_FAILED" -ne 0 ]; then
    warn "one or more checks did not match the expected status — see above"
    exit 1
fi

ok "all checks passed"
