-- What an agent's client says it can do.
--
-- An agent running a plugin can accept a context injection, author a
-- compaction summary and enforce a policy gate. An agent running a bare MCP
-- client can do none of those. The server has to know which, or it routes work
-- to an agent that cannot take it — and the agent never says why, because the
-- request simply fails.
--
-- The column is JSONB because the shape belongs to the client: the server
-- reads the capability names and the host identity, and leaves the rest to the
-- client that wrote it. A client that invents a field does not break the read.
--
-- NULL means the client has never announced. That is not the same as
-- announcing nothing: a client that has never spoken is a client that has not
-- told us what it is, and the two are worth distinguishing.

ALTER TABLE agents
    ADD COLUMN IF NOT EXISTS capabilities JSONB;
