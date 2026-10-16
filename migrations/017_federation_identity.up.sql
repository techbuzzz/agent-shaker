-- Federation Phase 2: agent identity binding and tool targeting.
--
-- Two additions, both about making a generated MCP configuration trustworthy:
--
--   1. agents.tool_type  — which client this agent runs in. The config
--      generator picks the file layout from it, so "the file we told you to
--      write" and "what this agent actually is" stop being two guesses.
--
--   2. api_keys.agent_id — one key, one agent. Until now a key was a
--      principal plus an optional project, and the MCP surface took agent_id
--      from the URL query string without authenticating it: any holder of a
--      project key could act as any agent in that project. Binding the agent
--      to the row makes the identity a property of the credential.
--
-- Forward-only, per docs/MIGRATIONS.md: there is no .down file.

-- === Tool type ============================================================
--
-- A CHECK rather than an enum, for the reason migration 009 gives: an enum
-- needs ALTER TYPE to add a value and cannot be dropped in older Postgres,
-- while a CHECK is a cheaper long-term cost for a vocabulary this size.
-- internal/models.ToolType is the single source of truth for the values; this
-- constraint only makes the database agree with it.
--
-- DEFAULT 'unknown' rather than NOT NULL with no default: agents created
-- before this migration, and agents created by an MCP runtime that has not
-- been upgraded, both have to keep working. 'unknown' is the honest answer
-- for them, and the config endpoint renders the generic format for it.
ALTER TABLE agents ADD COLUMN IF NOT EXISTS tool_type VARCHAR(50) NOT NULL DEFAULT 'unknown'
    CHECK (tool_type IN ('claude-code','codex','opencode','copilot-cli','copilot-vs',
                         'cursor','windsurf','a2a','generic-mcp','unknown'));

-- Filtering the roster by tool ("which agents run under Claude Code") is a
-- fleet-view question the dashboard and the config generator both ask.
CREATE INDEX IF NOT EXISTS idx_agents_tool_type ON agents(tool_type);

-- === Key-to-agent binding =================================================
--
-- NULLABLE and with no default, deliberately. A NULL agent_id means "this key
-- is not bound to an agent", which is what every key issued before this
-- migration is, and what a legacy API_KEYS secret resolves to. Those must keep
-- working exactly as they did: the MCP handler treats a NULL binding as "trust
-- the requested agent_id, as before". Only keys minted through the agent
-- connection flow get a value here, and for those the binding is authoritative.
--
-- ON DELETE CASCADE, not SET NULL: a credential that survives its agent is a
-- live key that resolves to an agent row which no longer exists — the next
-- heartbeat from it would create work nobody is tracking. Deleting an agent
-- revoking its credentials is the correct behaviour, and the UI lists them
-- first so the operator is not surprised.
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS agent_id UUID REFERENCES agents(id) ON DELETE CASCADE;

-- The key inventory filters "keys of this agent" and the UI shows one agent's
-- credentials. Partial, because the overwhelmingly common case is a NULL
-- binding that this index can never serve.
CREATE INDEX IF NOT EXISTS idx_api_keys_agent ON api_keys(agent_id) WHERE agent_id IS NOT NULL;