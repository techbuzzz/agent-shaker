-- Hook layer: task-lifecycle events run a command.
--
-- Two tables. `hooks` is the configuration — what to run, when, and with what
-- matcher. `hook_runs` is what happened when it ran.
--
-- Why this exists: the server already knows when a task is claimed, completed,
-- handed off, blocked or released, and already broadcasts those changes to the
-- browser. But a broadcast is a notification shaped for a UI, and it is
-- consumed by whoever is looking at the screen. A hook needs the same facts
-- in a form a command can read — which task, which project, which agent, what
-- changed — and it needs to be able to react to the change rather than
-- display it.
--
-- Forward-only, per docs/MIGRATIONS.md: there is no .down file.

-- === hooks ==================================================================
--
-- The command is an argument vector, not a string, and that is the single
-- most important decision in this table.
--
-- A shell string means the server has to quote, escape and interpolate its
-- way through `sh -c`, and every one of those steps is a place where a task
-- title containing a quote becomes a command injection. An argument vector
-- has no quoting step at all: the server hands the process the arguments it
-- stored, and the shell is never involved. The stored form is therefore also
-- the executed form.
--
-- `stdin` holds how the event reaches the process. `{"mode":"event"}` writes
-- the whole event as JSON on stdin, which is what a plain script wants.
-- `{"mode":"prompt","template":"..."}` renders a prompt instead, which is what
-- a headless agent CLI wants — it reads a prompt, not an event.
--
-- `post` is the feedback channel: it says what to do with the command's
-- stdout. With `{"kind":"context"}` the output is written back into the
-- domain as a context on the entity's project, which is what turns a hook
-- from a notification into a reviewer whose verdict the worker reads on its
-- next turn.

CREATE TABLE IF NOT EXISTS hooks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    -- A closed set, because a hook bound to an event the server can never
    -- publish is a hook that silently never fires. The vocabulary is
    -- internal/events' KnownTypes and the two must agree.
    event           VARCHAR(50) NOT NULL CHECK (event IN (
                        'task.claimed', 'task.completed', 'task.handoff',
                        'task.reassigned', 'task.blocked', 'task.ready',
                        'blocker.resolved', 'agent.stale', 'context.published'
                    )),
    -- Optional narrowing: a regular expression over one named field of the
    -- entity. NULL matches every event of this type.
    matcher         JSONB,
    -- The argument vector. Never a shell string — see the note above.
    command         TEXT[] NOT NULL CHECK (cardinality(command) > 0),
    -- Environment variables for the command. A value written as
    -- "{secret:NAME}" is a reference rather than a value: the server resolves
    -- it at run time from its own secret store, injects it into the command's
    -- environment, and redacts it from the captured output. The secret itself
    -- is never stored in this column and never appears in the hook row.
    env             JSONB,
    stdin           JSONB NOT NULL DEFAULT '{"mode":"event"}'::jsonb,
    post            JSONB,
    -- Bounded so a hook cannot pin a worker for an hour. 3600 is the ceiling
    -- and 120 the default; the runner kills the process group when the
    -- deadline passes.
    timeout_seconds INTEGER NOT NULL DEFAULT 120
                    CHECK (timeout_seconds > 0 AND timeout_seconds <= 3600),
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    -- No foreign key, for the same reason blockers.resolved_by has none: the
    -- author is whoever made the request, which may be a person in the
    -- directory, an API principal, or an operator working against the
    -- database directly. A foreign key would have to pick one of the three
    -- and would reject the other two.
    created_by      UUID,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- The dispatcher's query is "the enabled hooks for this project and this
-- event", and the partial index on exactly that predicate keeps the lookup
-- off the whole table as the hook count grows.
CREATE INDEX IF NOT EXISTS idx_hooks_project_event ON hooks(project_id, event)
    WHERE enabled;

-- === hook_runs =============================================================
--
-- The run log. It outlives the hook it belongs to: hook_id is nullable and
-- set to NULL on delete, because "this hook ran twice and then someone
-- deleted it" is exactly the row an operator needs and exactly the row a
-- cascading delete would erase.

CREATE TABLE IF NOT EXISTS hook_runs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hook_id      UUID REFERENCES hooks(id) ON DELETE SET NULL,
    -- The id of the delivery this run belongs to. Two runs of the same hook
    -- for the same delivery are the same attempt, not two events.
    delivery_id  UUID NOT NULL,
    project_id   UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    event        VARCHAR(50) NOT NULL,
    entity_type  VARCHAR(50),
    entity_id    UUID,
    -- The normalised outcome. The raw exit code is kept beside it because the
    -- two vocabularies disagree: mcode exec uses 2 for an invalid invocation
    -- and Claude Code uses 2 for a blocking decision.
    status       VARCHAR(20) NOT NULL CHECK (status IN (
                     'succeeded', 'failed', 'timeout', 'cancelled', 'blocked'
                 )),
    exit_code    INTEGER,
    -- Truncated on capture. A hook that prints a megabyte must not be able to
    -- write a megabyte per run into the database.
    stdout       TEXT,
    stderr       TEXT,
    duration_ms  INTEGER,
    started_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    finished_at  TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_hook_runs_hook_started ON hook_runs(hook_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_hook_runs_project_started ON hook_runs(project_id, started_at DESC);

-- === Per-project opt-in =====================================================
--
-- The global switch turns the hook layer on for the process; this column
-- turns it on for one project. Both have to agree before a command runs.
--
-- The default is false, and the reason is that a hook is arbitrary command
-- execution. Enabling the feature globally should not silently enable it for
-- every project in the deployment: an operator turns it on where they have
-- decided the commands are safe, one project at a time.

ALTER TABLE projects ADD COLUMN IF NOT EXISTS hooks_enabled BOOLEAN NOT NULL DEFAULT FALSE;

