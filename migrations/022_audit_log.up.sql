-- The append-only event journal.
--
-- Every capability after this one — audit, cost, approvals, meaningful
-- tracing — needs one thing this project did not have: a record of what
-- happened, written by the thing that happened it, that nobody can quietly
-- rewrite afterwards.
--
-- Why a table and not a log file: the journal is read through the same API as
-- everything else, filtered the same way, and written in the same transaction
-- as the state it describes. A log file gives none of those.
--
-- Why append-only: a trail that can be edited is not a trail. There is no
-- UPDATE or DELETE against this table anywhere in the codebase, and a test
-- pins that.
--
-- The hash chain is what makes "nobody can quietly rewrite it" checkable
-- rather than merely intended. Each row carries the hash of the row before it,
-- so editing or removing any row breaks every hash after it. The chain is per
-- project: one project's traffic is one chain, and the head of a chain is
-- published outside this database so that a database-level attacker cannot
-- rewrite the trail and the anchor together.
--
-- `before` and `after` are JSONB snapshots of the entity as it was and as it
-- became. They are NULL where the publisher did not know the prior state —
-- a creation has no before, and a domain event that carries only "this
-- happened" has no diff to record.

CREATE TABLE IF NOT EXISTS audit_events (
    id BIGSERIAL PRIMARY KEY,
    occurred_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Who did it. NULL for a change the server made on its own — a reaper
    -- releasing a stale agent's tasks has no actor to name.
    actor_type VARCHAR(50),
    actor_id UUID,

    -- What it happened in. NULL for a change that belongs to no project.
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,

    -- What it happened to.
    entity_type VARCHAR(50),
    entity_id UUID,

    -- What happened, in the domain's own vocabulary: task.claimed,
    -- context.published, agent.stale.
    action VARCHAR(100) NOT NULL,

    before JSONB,
    after JSONB,

    -- Correlation. The trace id follows one request; the request id is the
    -- server's own handle on it.
    trace_id VARCHAR(64),
    request_id VARCHAR(64),

    -- The chain. `hash` covers every other column of this row plus
    -- `prev_hash`, so the row's own content cannot be changed without
    -- breaking it.
    prev_hash VARCHAR(64),
    hash VARCHAR(64)
);

-- The journal is read newest-first within a project.
CREATE INDEX IF NOT EXISTS idx_audit_events_project_occurred
    ON audit_events (project_id, occurred_at DESC);

-- And by entity, for "what happened to this task".
CREATE INDEX IF NOT EXISTS idx_audit_events_entity
    ON audit_events (entity_type, entity_id);

-- The chain is walked per project, and the head of a chain is looked up by
-- project. Without this the lookup is a scan of the whole table.
CREATE INDEX IF NOT EXISTS idx_audit_events_chain
    ON audit_events (project_id, id DESC) WHERE hash IS NOT NULL;
