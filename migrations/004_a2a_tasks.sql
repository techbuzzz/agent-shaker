-- A2A protocol task store. Distinct from the project `tasks` table (001_init.sql)
-- which holds human-assigned work items; `a2a_tasks` holds the agent-runtime
-- state machine for an A2A request lifecycle.
--
-- Schema notes:
--   * id is the natural A2A task id (UUID string) and serves as PK.
--   * message / result / artifacts are JSONB so the A2A server can marshal
--     models.Message / models.Result / []models.Artifact without flattening.
--   * Indexes support the two common read paths: filter by status, and
--     order by recency.
CREATE TABLE IF NOT EXISTS a2a_tasks (
    id           TEXT PRIMARY KEY,
    status       TEXT NOT NULL DEFAULT 'pending',
    message      JSONB NOT NULL,
    result       JSONB,
    artifacts    JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS a2a_tasks_status_idx ON a2a_tasks(status);
CREATE INDEX IF NOT EXISTS a2a_tasks_updated_at_idx ON a2a_tasks(updated_at DESC);
