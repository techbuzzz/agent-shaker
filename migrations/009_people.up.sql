-- People: the subject the rest of the system attributes action to.
--
-- Until now the only identity in this database was an API key, and every key
-- meant "an anonymous holder of a shared secret". Nothing could answer "who
-- approved this" or "who owns this standup", which is exactly what approvals
-- (M4) and the audit journal (M3) need to exist. This table is that missing
-- subject; no existing table references it yet, so the migration is additive
-- and every historical row stays valid.
--
-- kind is stored rather than inferred, because a service identity (a CI job, an
-- MCP client, the legacy shared secret) and a human authenticate the same way
-- — with a key — but only a human can hold approval authority. Inferring that
-- from a convention would make the rule uncheckable.
CREATE TABLE IF NOT EXISTS people (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind VARCHAR(16) NOT NULL DEFAULT 'human',
    display_name VARCHAR(255) NOT NULL,
    email VARCHAR(320),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- A CHECK rather than an enum: an enum needs ALTER TYPE to add a value and
    -- cannot be dropped in older Postgres. The vocabulary is two words wide and
    -- unlikely to grow, so a constraint is the cheaper long-term cost.
    CONSTRAINT people_kind_check CHECK (kind IN ('human', 'service'))
);

-- One row per real address, case-insensitively: "Ann@Example.com" and
-- "ann@example.com" are the same person, and the API answers 409 rather than
-- silently creating a duplicate identity.
--
-- Partial on IS NOT NULL so the many service identities, which have no email,
-- do not each occupy a NULL entry in the index. (Postgres does index NULLs in
-- a btree, and a unique index over all of them would collide.)
CREATE UNIQUE INDEX IF NOT EXISTS idx_people_email_unique
    ON people (lower(email)) WHERE email IS NOT NULL;

-- Listing people filtered by kind is a real query in the management UI, and
-- the key-management path always joins on principal_id (covered by the
-- api_keys index added in 010).
CREATE INDEX IF NOT EXISTS idx_people_kind ON people (kind);
