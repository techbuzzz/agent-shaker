-- 006_project_repos.sql — Phase 2 of the mesh-app plan.
--
-- A "Project = one product surface, N repositories" model. Each row in
-- project_repos is one git repo (or monorepo subdir) participating in the
-- project, optionally tied to a specific agent (the agent that owns the
-- code, e.g. backend-agent). PM-owned repos (no agent_id) are also allowed
-- (cross-cutting infra).
--
-- role is an open enum stored as VARCHAR so we can add more values
-- (docs|infra|design) without a migration. Today only 'code' is used
-- widely; everything else falls through.
--
-- Cascades:
--   * project_id ON DELETE CASCADE — project removal removes its repos
--   * agent_id ON DELETE SET NULL — deleting an agent unlinks its repos
--     so the repo row is preserved as "unowned" (PM can reassign).

CREATE TABLE IF NOT EXISTS project_repos (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    url        TEXT NOT NULL,
    branch     VARCHAR(255) DEFAULT 'main',
    role       VARCHAR(50)  DEFAULT 'code',
    agent_id   UUID REFERENCES agents(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_repos_project ON project_repos(project_id);
CREATE INDEX IF NOT EXISTS idx_repos_agent   ON project_repos(agent_id);
