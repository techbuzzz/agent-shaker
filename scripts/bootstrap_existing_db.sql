-- Adopt an existing database into the migration chain.
--
-- Run this ONCE, against a database that already has the schema but has never
-- been tracked by golang-migrate. It is for the "I have data, I want to start
-- versioning from here" case.
--
--     docker compose cp scripts/bootstrap_existing_db.sql postgres:/tmp/boot.sql
--     docker compose exec -T postgres psql -U mcp -d mcp_tracker \
--       -v ON_ERROR_STOP=1 -f /tmp/boot.sql
--
-- WHY THIS FILE USED TO BE WRONG
--
-- It created `schema_migrations (version VARCHAR, applied_at, checksum)` and
-- inserted filenames like '001_init.sql'. That table shape belongs to a custom
-- runner that no longer exists. The real table — created by
-- github.com/golang-migrate/migrate/v4 — is:
--
--     schema_migrations (version bigint NOT NULL PRIMARY KEY, dirty boolean NOT NULL)
--
-- a single row holding one integer. So the old inserts failed on both counts:
-- a string into a bigint column, and an `applied_at` column that does not
-- exist. Every statement after the first raised, including the final SELECT.
-- This file never could have run successfully.
--
-- ON_ERROR_STOP=1 matters: without it psql reports each error, keeps going,
-- and exits 0 as long as the LAST statement succeeded — so a failing run still
-- reports success. An adoption script that exits 0 when it did nothing is
-- worse than no script at all.

-- Matches the golang-migrate postgres driver's own DDL exactly. A no-op when
-- the table already exists, which is the normal case.
CREATE TABLE IF NOT EXISTS schema_migrations (
    version bigint NOT NULL PRIMARY KEY,
    dirty   boolean NOT NULL
);

-- Recording the version is left to the CLI, deliberately.
--
-- It is tempting to accept a target version here, but `psql -v` variables are
-- not visible to current_setting() inside a dollar-quoted block, so the value
-- has to be threaded through with set_config() and quoted twice. That plumbing
-- is easy to get subtly wrong, and this file runs once against a database
-- holding real data.
--
-- The CLI has no such problem, and it is the supported path:
--
--     make migrate-force MIGRATION_VERSION=8
--
-- Pick N as the highest migration whose effects are ALREADY in the schema:
--
--     ls migrations/                  # 008_... is the highest that exists
--     make migrate-version            # what is recorded right now
--     docker compose exec -T postgres psql -U mcp -d mcp_tracker -c '\dt'
--
-- Too high and migrations are skipped silently, leaving the schema short.
-- Too low and one replays. Both fail quietly, so check before you commit.

-- Confirm the shape is right before forcing a version into it.
SELECT version, dirty FROM schema_migrations;
