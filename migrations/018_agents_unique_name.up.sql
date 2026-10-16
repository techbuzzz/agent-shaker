-- Uniqueness on (project_id, name), which the agents table has always been
-- treated as having but never actually declared.
--
-- Why this migration exists: executeRegisterSelf in internal/mcp/handler.go
-- does `INSERT ... ON CONFLICT (project_id, name) DO UPDATE`, an upsert whose
-- whole purpose is that an agent which restarts and registers again under the
-- same name gets the same row instead of a duplicate. Postgres rejects an
-- ON CONFLICT whose target matches no index, so with no unique index the tool
-- failed for every caller with:
--
--     ERROR: there is no unique or exclusion constraint matching the ON
--     CONFLICT specification (SQLSTATE 42P10)
--
-- It was therefore never usable, and it is the documented onboarding path for
-- an agent whose connection carries no agent_id.
--
-- Declaring the constraint rather than rewriting the statement to
-- SELECT-then-INSERT: a read-then-write in application code has the same race
-- the database was supposed to close, and still lets two concurrent
-- registrations create two rows. The invariant belongs in the schema.
--
-- This migration adds no dedupe step on purpose. If duplicates somehow exist,
-- it fails loudly rather than deleting an agent to make itself apply — and
-- forward-only is what this project's migrations are, so a failure here is a
-- stopped deploy rather than a half-migrated database.

CREATE UNIQUE INDEX IF NOT EXISTS idx_agents_project_name_unique
    ON agents (project_id, name);