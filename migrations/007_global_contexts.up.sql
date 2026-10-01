-- 007_global_contexts.sql — Phase 3 of the mesh-app plan.
--
-- A unified table for "playbooks and docs that any agent may want to
-- read", with two scopes:
--
--   scope = 'global' : read by any agent on the server, regardless of
--                       project binding. Only PM-role agents can write.
--   scope = 'project': read by any agent in the matching project_id.
--                       Acts as a project-level doc (alternative to
--                       existing `contexts` table; we keep both for
--                       back-compat and because `contexts` is task-scoped).
--
-- The CHECK constraint enforces the invariant:
--   * scope='global'  ⇒ project_id IS NULL
--   * scope='project' ⇒ project_id IS NOT NULL
--
-- Uniqueness on (scope, title) when scope='global' prevents duplicate
-- playbook names. For scope='project', uniqueness is per-project.

CREATE TABLE IF NOT EXISTS global_contexts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope       VARCHAR(50)  NOT NULL
                          CHECK (scope IN ('global','project')),
    project_id  UUID REFERENCES projects(id) ON DELETE CASCADE,
    agent_id    UUID NOT NULL REFERENCES agents(id) ON DELETE RESTRICT,
    title       VARCHAR(255) NOT NULL,
    content     TEXT,
    tags        TEXT[]       NOT NULL DEFAULT ARRAY[]::TEXT[],
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT global_contexts_scope_chk
        CHECK ( (scope = 'global'  AND project_id IS NULL)
             OR (scope = 'project' AND project_id IS NOT NULL) )
);

CREATE INDEX IF NOT EXISTS idx_gctx_scope     ON global_contexts(scope);
CREATE INDEX IF NOT EXISTS idx_gctx_project   ON global_contexts(project_id);
CREATE INDEX IF NOT EXISTS idx_gctx_tags      ON global_contexts USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_gctx_updated   ON global_contexts(updated_at DESC);

-- Uniqueness: a global playbook name is unique. For project-scoped docs,
-- the same title may exist across projects, so the index is partial.
CREATE UNIQUE INDEX IF NOT EXISTS uq_gctx_global_title
    ON global_contexts (title)
    WHERE scope = 'global';

CREATE UNIQUE INDEX IF NOT EXISTS uq_gctx_project_title
    ON global_contexts (project_id, title)
    WHERE scope = 'project';
