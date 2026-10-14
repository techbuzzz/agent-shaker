-- Coordination Plane: worker heartbeat, task dependencies, blockers.
--
-- Three mechanisms that let agents coordinate without a human in the loop:
--   1. Worker heartbeat  — a lease every agent renews; a reaper takes the
--      lease away when it lapses and returns the agent's tasks to the backlog.
--   2. Task readiness      — a dependency DAG plus a structured handoff, so a
--      backend agent can hand work to a frontend agent with its context.
--   3. Blocker escalation  — a typed blocker a human resolves through REST,
--      and the agent is told over WebSocket.
--
-- Migration 014 added a CHECK constraint on tasks.status over
-- (pending, in_progress, blocked, done, cancelled). The readiness signal adds
-- two states to that vocabulary, so the constraint has to be widened here or
-- every INSERT of a ready task fails at the storage layer. internal/models/status.go
-- stays the single source of truth for the vocabulary; this file only makes the
-- database agree with it.
--
-- Forward-only, per docs/MIGRATIONS.md: there is no .down file.

-- === Worker heartbeat =======================================================

-- lease_expires_at is the deadline the server hands back with every heartbeat:
-- the agent is expected to ping again before it. NULL means the agent never
-- opted into a lease (agents created through the UI rather than an agent
-- runtime), and the reaper leaves those alone.
ALTER TABLE agents ADD COLUMN IF NOT EXISTS lease_expires_at TIMESTAMP;

-- current_task_id is a denormalised pointer to the task the agent is working
-- on. The heartbeat already reports the task; keeping it on the agent row is
-- what makes "who is working on what" a single index probe for the dashboard.
ALTER TABLE agents ADD COLUMN IF NOT EXISTS current_task_id UUID REFERENCES tasks(id) ON DELETE SET NULL;

-- The last heartbeat payload, kept so the dashboard can show the latest
-- progress/ETA without reading the whole history table.
ALTER TABLE agents ADD COLUMN IF NOT EXISTS heartbeat_payload JSONB;

-- Partial index: the reaper's query is "lease expired and not already
-- offline", and a partial index on exactly that predicate keeps it off the
-- full agents table as the fleet grows.
CREATE INDEX IF NOT EXISTS idx_agents_lease ON agents(lease_expires_at)
    WHERE lease_expires_at IS NOT NULL AND status != 'offline';

-- agent_heartbeats existed for the standup feature, which recorded liveness
-- without a project or a task. The coordination plane needs both: a heartbeat
-- is meaningless without knowing which project to broadcast it into.
ALTER TABLE agent_heartbeats ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE CASCADE;
ALTER TABLE agent_heartbeats ADD COLUMN IF NOT EXISTS task_id UUID REFERENCES tasks(id) ON DELETE SET NULL;

-- SMALLINT with an explicit CHECK rather than a 0-100 NUMERIC: the value is a
-- percentage, and a 0-100 float invites storing 42.5 for something that can
-- only be a percentage.
ALTER TABLE agent_heartbeats ADD COLUMN IF NOT EXISTS progress_pct SMALLINT
    CHECK (progress_pct BETWEEN 0 AND 100);

-- Retention: agent_heartbeats grows with every agent every 30s. Without a
-- policy it becomes the largest table in the database for no query benefit —
-- nothing reads history beyond "recent".
-- CREATE INDEX agent_heartbeats_agent_time ON agent_heartbeats(agent_id, heartbeat_time DESC);

-- === Task readiness: dependencies and handoff ================================

ALTER TABLE tasks ADD COLUMN IF NOT EXISTS predecessor_task_id UUID REFERENCES tasks(id) ON DELETE SET NULL;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS handoff_context TEXT;

CREATE TABLE IF NOT EXISTS task_dependencies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id         UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    depends_on_task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    -- One edge per pair: adding the same dependency twice is a no-op, not an
    -- error, because two agents racing to declare it is the normal case.
    UNIQUE(task_id, depends_on_task_id),
    -- A task cannot depend on itself. The recursive cycle check in the query
    -- layer catches longer cycles; this catches the trivial one in the
    -- database, where it costs nothing.
    CHECK(task_id <> depends_on_task_id)
);

CREATE INDEX IF NOT EXISTS idx_task_deps_task ON task_dependencies(task_id);
-- The reverse lookup ("what does this task unblock?") runs on every completion.
-- Without this index it is a sequential scan of the whole edge table.
CREATE INDEX IF NOT EXISTS idx_task_deps_depends ON task_dependencies(depends_on_task_id);
CREATE INDEX IF NOT EXISTS idx_tasks_predecessor ON tasks(predecessor_task_id);

-- "ready" is a distinct state from "pending": all dependencies are satisfied
-- and the task is claimable. Keeping it out of "pending" is what lets the
-- dashboard and the reaper tell "waiting on a dependency" apart from "nobody
-- has picked this up".
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_status_check;
ALTER TABLE tasks
    ADD CONSTRAINT tasks_status_check
    CHECK (status IN ('pending','blocked','ready','in_progress','done','cancelled'));

-- === Blockers ===============================================================

CREATE TABLE IF NOT EXISTS blockers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id       UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    agent_id      UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    project_id    UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    -- The vocabulary is a closed set because each type routes to a different
    -- human: "approval" is a deploy gate, "question" is a design decision.
    -- An unknown type would be a blocker nobody knows how to action.
    blocker_type  VARCHAR(50) NOT NULL
        CHECK (blocker_type IN ('question','dependency','resource','decision','approval')),
    title         VARCHAR(500) NOT NULL,
    description   TEXT,
    status        VARCHAR(50) NOT NULL DEFAULT 'open'
        CHECK (status IN ('open','in_review','resolved','dismissed')),
    resolution    TEXT,
    -- Deliberately not a foreign key: the resolver is whoever answered, and
    -- that may be a person in the directory, an API principal, or an operator
    -- working directly against the database. A FK would have to pick one and
    -- would reject the other two.
    resolved_by   UUID,
    resolved_at   TIMESTAMP,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_blockers_task ON blockers(task_id);
-- The dashboard's default view is "open blockers in this project", in the order
-- they were raised. The composite index serves both without a sort.
CREATE INDEX IF NOT EXISTS idx_blockers_project_status ON blockers(project_id, status, created_at);
CREATE INDEX IF NOT EXISTS idx_blockers_agent ON blockers(agent_id);