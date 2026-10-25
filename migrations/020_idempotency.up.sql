-- Idempotency keys for the mutating MCP tools.
--
-- Why this is here and not in the MCP rewrite (roadmap milestone M5): the
-- 2026-07-28 revision of the specification makes retries ordinary traffic. A
-- broken response stream loses the in-flight request, and the client is
-- required to re-issue it with a new request id; a multi-round-trip request
-- retries the original request. Both are normal, documented behaviours of a
-- conforming client, not error paths.
--
-- A coordination plane whose tools mutate shared state has to be idempotent
-- by construction before that happens, because the failure it produces is not
-- a failed request. It is a task claimed twice, or a task completed twice, or
-- a handoff that creates a second successor — a state the server then reports
-- as truth. The client cannot tell the difference, so the server must.
--
-- The key is supplied by the client and is scoped to (tool, agent). An agent
-- is known on every mutating call because the credential names it
-- (api_keys.agent_id, migration 017), so the scope is never ambiguous and
-- two agents cannot collide on the same key.
--
-- The stored response is the first result, replayed for any repeat of the
-- same key. Replaying the first result is what makes the operation
-- idempotent: the second request sees the outcome of the first rather than
-- computing a second one.

CREATE TABLE IF NOT EXISTS idempotency_keys (
    tool       VARCHAR(50) NOT NULL,
    agent_id   UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    key        VARCHAR(200) NOT NULL,
    -- The hash of the request the key was used for.
    --
    -- It is part of the primary key on purpose. The key alone says "this
    -- client means one logical operation"; the request hash says which one. A
    -- client that reuses a key for a different request gets a different row
    -- and a real execution, rather than a replay of the first answer — being
    -- told "claimed" for a task that was never claimed is worse than the
    -- duplicate the key was meant to prevent.
    request_hash VARCHAR(64) NOT NULL,
    -- The first result, replayed for a repeat of the same key and the same
    -- request.
    --
    -- TEXT rather than JSONB on purpose. JSONB normalises what it stores — it
    -- drops insignificant whitespace and reorders keys — so a replay would
    -- return an answer that parses the same but is not the same bytes as the
    -- one the client first received. A client that compares its retry against
    -- its first answer would see a difference. The row exists to replay one
    -- exact answer, not to be queried.
    response     TEXT NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tool, agent_id, key, request_hash)
);

-- The keys are only useful while a client might still retry. A key older
-- than the retention window can no longer be replayed, and the row is dead
-- weight — but pruning is a separate concern from correctness, so the index
-- makes the prune cheap and the policy is documented here rather than
-- implemented in a migration.
--
-- Retention policy: 24 hours. That is longer than any client retry window
-- and short enough that the table stays proportional to the fleet rather
-- than to the project's lifetime.
CREATE INDEX IF NOT EXISTS idx_idempotency_created_at ON idempotency_keys(created_at);
