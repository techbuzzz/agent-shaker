-- Migration: remove the demo dataset seeded by 002_sample_data
--
-- `002_sample_data.up.sql` inserted 3 projects, 9 agents, 9 tasks and 4
-- contexts with fixed UUIDs. It was written to populate the Agents page
-- during development, but it sits in the forward-only migration chain, so
-- every fresh database — including a production one — came up pre-loaded
-- with "E-Commerce Platform", "React Frontend Agent" and eight other rows
-- of fiction, rendered by the UI identically to real data.
--
-- 002 is left untouched: it has been applied in existing environments, and
-- editing an applied migration makes environments diverge. This one undoes
-- the effect instead.
--
-- The dataset is not lost. It moved to scripts/seed_demo_data.sql, which is
-- opt-in:
--
--     docker compose exec -T postgres psql -U mcp -d mcp_tracker \
--       < scripts/seed_demo_data.sql
--
-- Deletion is keyed on the exact UUIDs 002 inserted, never on name or on
-- "everything in the table", so a real project that happens to be called
-- "E-Commerce Platform" is untouched and a database with real work in it
-- loses nothing.
--
-- Child rows go first. The FKs on tasks.created_by / contexts.agent_id do
-- not cascade, so ordering is explicit rather than left to the planner.

DELETE FROM contexts WHERE id IN (
    '880e8400-e29b-41d4-a716-446655440001',
    '880e8400-e29b-41d4-a716-446655440002',
    '880e8400-e29b-41d4-a716-446655440003',
    '880e8400-e29b-41d4-a716-446655440004'
);

DELETE FROM tasks WHERE id IN (
    '770e8400-e29b-41d4-a716-446655440001',
    '770e8400-e29b-41d4-a716-446655440002',
    '770e8400-e29b-41d4-a716-446655440003',
    '770e8400-e29b-41d4-a716-446655440004',
    '770e8400-e29b-41d4-a716-446655440005',
    '770e8400-e29b-41d4-a716-446655440006',
    '770e8400-e29b-41d4-a716-446655440007',
    '770e8400-e29b-41d4-a716-446655440008',
    '770e8400-e29b-41d4-a716-446655440009'
);

DELETE FROM agents WHERE id IN (
    '660e8400-e29b-41d4-a716-446655440001',
    '660e8400-e29b-41d4-a716-446655440002',
    '660e8400-e29b-41d4-a716-446655440003',
    '660e8400-e29b-41d4-a716-446655440004',
    '660e8400-e29b-41d4-a716-446655440005',
    '660e8400-e29b-41d4-a716-446655440006',
    '660e8400-e29b-41d4-a716-446655440007',
    '660e8400-e29b-41d4-a716-446655440008',
    '660e8400-e29b-41d4-a716-446655440009'
);

DELETE FROM projects WHERE id IN (
    '550e8400-e29b-41d4-a716-446655440001',
    '550e8400-e29b-41d4-a716-446655440002',
    '550e8400-e29b-41d4-a716-446655440003'
);
