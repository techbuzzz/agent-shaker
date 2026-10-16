-- Heartbeat retention and the ETA an agent reports with it.
--
-- Two follow-ups to 015_coordination_plane:
--
--  1. eta. `send_heartbeat` accepts an ETA and the design lists it in the
--     event payload, but 015 added no column for it — the value was accepted,
--     answered with 200 and silently discarded. An operator reading "done in
--     twenty minutes" from a heartbeat should be reading a stored fact.
--
--  2. retention. agent_heartbeats gains roughly 2 880 rows per agent per day
--     at a 30s ping and nothing reads beyond "recent". The prune runs on a
--     schedule (see internal/coordination/scheduler.go) rather than as part of
--     a migration, because a one-off DELETE at deploy time only moves the
--     problem to the next deploy.

ALTER TABLE agent_heartbeats ADD COLUMN IF NOT EXISTS eta TIMESTAMP;

-- (1) is a small nullable column; a btree on it would serve nothing, because
-- nothing filters by ETA. The retention sweep reads heartbeat_time only.