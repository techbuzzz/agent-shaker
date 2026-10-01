# syntax=docker/dockerfile:1.7

# ---- build stage ----
# The go directive in go.mod is 1.26.0 and the toolchain in use is 1.27.1, so
# the builder must be >= 1.27 or `go build` refuses the module.
FROM golang:1.27-alpine AS build

WORKDIR /src

# Dependencies first, so the module cache layer survives source-only edits.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 yields a static binary that runs on a bare alpine image.
# -trimpath strips build paths; -s -w drops the symbol table and DWARF, which
# is roughly a 30% size reduction on this binary.
#
# main.version / main.commit / main.buildTime are declared in cmd/server/main.go
# and surfaced on the "build" log line at boot, so a running container can be
# traced back to an exact commit without an extra endpoint.
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags "-s -w \
          -X main.version=${VERSION} \
          -X main.commit=${COMMIT} \
          -X main.buildTime=${BUILD_TIME}" \
        -o /out/mcp-server ./cmd/server \
 && CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags "-s -w" \
        -o /out/migrate ./cmd/migrate

# ---- runtime stage ----
# alpine rather than scratch: the binary needs CA certificates to validate a
# managed Postgres certificate (sslmode=verify-full) and an OTLP endpoint, and
# a HEALTHCHECK needs a shell.
FROM alpine:3.21

# ca-certificates : TLS to Postgres / the OTLP collector
# tzdata           : correct timestamps in logs and standup date bucketing
# curl             : used by HEALTHCHECK below
RUN apk add --no-cache ca-certificates tzdata curl \
 && addgroup -S -g 10001 app \
 && adduser  -S -u 10001 -G app -h /app -s /sbin/nologin app

WORKDIR /app

COPY --from=build /out/mcp-server /app/mcp-server
COPY --from=build /out/migrate    /app/migrate

# Migrations ship with the image so the compose one-shot `migrate` job (and an
# operator running the image by hand) can apply them without a bind mount.
COPY migrations /app/migrations

# The frontend is a separate Nuxt SSR service (see web/Dockerfile) and is
# deliberately NOT baked in here. The Go server only serves ./web/dist when
# that directory happens to exist, which in this topology it does not — the
# single public origin is the Nuxt service.

USER app:app

ENV PORT=8080 \
    ENV=production \
    GOMAXPROCS=0

EXPOSE 8080

# /healthz is a liveness probe (always 200). Readiness is /readyz, which pings
# the database; gate traffic on that one, not on this one.
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -fsS "http://127.0.0.1:${PORT}/healthz" || exit 1

CMD ["./mcp-server"]
