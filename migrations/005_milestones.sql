-- 005_milestones.sql — Phase 1 of the mesh-app plan.
--
-- A milestone is a time-bounded group of tasks under a project, owned by
-- the PM (or any agent created_by). Tasks can be optionally linked via
-- milestone_id. Status transitions are enforced in Go (handlers layer),
-- not the database, because the "all linked tasks must be done|cancelled"
-- rule needs to inspect joined rows.
--
-- Cascades:
--   * project_id ON DELETE CASCADE — deleting a project removes its milestones
--   * created_by ON DELETE RESTRICT — we never silently orphan a milestone's creator
--   * tasks.milestone_id ON DELETE SET NULL — deleting a milestone unlinks its tasks
--     rather than deleting them, so task history survives a milestone rollback.

CREATE TABLE IF NOT EXISTS milestones (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title       VARCHAR(255) NOT NULL,
    description TEXT,
    status      VARCHAR(50)  NOT NULL DEFAULT 'planned'
                            CHECK (status IN ('planned','active','done','dropped')),
    target_date DATE,
    created_by  UUID NOT NULL REFERENCES agents(id) ON DELETE RESTRICT,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_milestones_project ON milestones(project_id);
CREATE INDEX IF NOT EXISTS idx_milestones_status  ON milestones(status);
CREATE INDEX IF NOT EXISTS idx_milestones_target  ON milestones(target_date)
    WHERE target_date IS NOT NULL;

-- Link tasks to milestones. Backfill-safe: existing tasks simply have NULL.
ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS milestone_id UUID REFERENCES milestones(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_tasks_milestone ON tasks(milestone_id);

-- Add a free-form `tags` column to tasks so features can be represented
-- as `feature:<name>` tags without a dedicated features table. Mirrors
-- the `contexts.tags` column and unlocks the Nuxt `FeatureList` derived
-- view (groups tasks by `feature:*` prefix).
ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS tags TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[];

CREATE INDEX IF NOT EXISTS idx_tasks_tags ON tasks USING GIN(tags);
