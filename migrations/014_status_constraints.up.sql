-- Status vocabularies as database constraints.
--
-- The task and agent status vocabularies used to be enforced only in Go, and
-- inconsistently: the REST validator accepted one set, the MCP tools another,
-- and the database accepted any string at all. An agent following our own
-- generated onboarding instructions could therefore write a status the REST
-- API would reject, and nothing at the storage layer noticed.
--
-- These constraints close that gap. The Go side (internal/models/status.go)
-- remains the source of truth for the vocabulary; this migration makes the
-- database agree with it rather than defining a third version.
--
-- Forward-only, per docs/MIGRATIONS.md: there is no .down file.

-- Existing rows are normalised before the constraint is added. Anything the
-- MCP surface used to accept but the REST validator did not ("working" for an
-- agent) is mapped to the closest canonical value rather than deleted: an
-- agent that reported "working" is working, so "active" is the honest
-- translation, and dropping the row would destroy task history.
UPDATE agents SET status = 'active' WHERE status = 'working';
UPDATE tasks  SET status = 'done'      WHERE status = 'completed';
UPDATE tasks  SET status = 'blocked'   WHERE status = 'failed';

-- Anything still outside the vocabulary (hand-edited data, an older release)
-- would make ADD CONSTRAINT fail and block the migration. Fall back to the
-- initial state rather than failing the deploy; the row is still reachable
-- and the operator can correct it.
UPDATE tasks  SET status = 'pending' WHERE status NOT IN ('pending','in_progress','blocked','done','cancelled');
UPDATE agents SET status = 'idle'    WHERE status NOT IN ('active','idle','blocked','offline');

ALTER TABLE tasks
    ADD CONSTRAINT tasks_status_check
    CHECK (status IN ('pending','in_progress','blocked','done','cancelled'));

ALTER TABLE agents
    ADD CONSTRAINT agents_status_check
    CHECK (status IN ('active','idle','blocked','offline'));

-- The project dashboard and the per-agent inbox both filter tasks by project
-- and status, and idx_tasks_status (status alone, from 001) cannot serve the
-- project predicate. This composite index covers the pair the queries actually
-- use. The claim guard itself is an index probe on the primary key — that part
-- was already fast and is not why this index exists.
CREATE INDEX IF NOT EXISTS tasks_project_status_idx ON tasks (project_id, status);