-- Full-text search over the contexts table.
--
-- Why this exists: the product promises that an agent can get fresh context
-- without guessing or scraping. With list_contexts as the only read path,
-- honouring that promise means listing every context in a project and reading
-- all of them to find one. This migration is what makes selection possible.
--
-- Design notes:
--
--   * The vector is a GENERATED column rather than a trigger-maintained one.
--     A trigger can drift out of step with the expression it was written
--     against; a generated column cannot, and Postgres maintains it on every
--     insert and update without our involvement. It also means existing rows
--     are backfilled by the ALTER itself, so the index is usable immediately
--     after the migration.
--
--   * The configuration is 'simple': no stemming, no stop-word list. There is
--     no ratified standard for agent memory to align a language configuration
--     with, and stemming on agent-written notes produces matches that look
--     wrong ("task"/"tasks" helps; "run"/"running" across unrelated notes
--     does not). Changing this later means reindexing, so it is worth being
--     deliberate rather than copying a default.
--
--   * The index is GIN, which is the right structure for tsvector containment
--     and ranking queries. It is larger and slower to build than a B-tree but
--     turns a sequential scan over every context in a project into an index
--     scan.
--
-- Migrations in this repository are forward-only and there is no .down file;
-- reverting means writing a later migration that drops the column and index.
ALTER TABLE contexts
    ADD COLUMN IF NOT EXISTS search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('simple', coalesce(title, '') || ' ' || coalesce(content, ''))
    ) STORED;

CREATE INDEX IF NOT EXISTS idx_contexts_search_vector ON contexts USING GIN (search_vector);

-- list_contexts orders by created_at within a project, and search ranks within
-- a project too. Both are served by this composite index; without it Postgres
-- falls back to filtering on search_vector and then sorting, which grows with
-- the number of matches rather than staying flat.
CREATE INDEX IF NOT EXISTS idx_contexts_project_created_at ON contexts (project_id, created_at DESC);
