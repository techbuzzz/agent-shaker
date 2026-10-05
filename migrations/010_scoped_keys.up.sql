-- API keys: the credential becomes a reference to a principal instead of a
-- shared secret.
--
-- The old scheme was "does this string match one of N strings in API_KEYS".
-- This one is "which principal does this credential belong to, and what is it
-- allowed to do" — which is what makes attribution, permissions and approval
-- authority expressible at all (see docs/ROADMAP.md, M2).
--
-- Only the hash is stored. The plaintext exists exactly once, in the POST
-- /api/keys response, and never reaches the database, the logs, or the list
-- endpoint.
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- SHA-256 of the presented key, hex-encoded (64 characters, enforced by
    -- construction in Go rather than by the type).
    --
    -- TEXT, not CHAR(64): bpcar semantics pad and ignore trailing spaces,
    -- which would make the UNIQUE constraint treat two different digests as
    -- equal the moment the digest length ever changed. There is nothing to gain
    -- from a fixed-width type here.
    --
    -- A deterministic digest, not bcrypt/argon2, and deliberately so: the
    -- authentication path has to find the row *by* the hash, which a
    -- per-row-salted KDF makes impossible without trying every row. The
    -- trade is sound because generated keys carry 256 bits of entropy
    -- (32 bytes from crypto/rand) — an attacker holding a stolen hash has no
    -- tractable search, so the slow-hash defence buys nothing here and would
    -- cost a full table scan on every request.
    key_hash TEXT NOT NULL,

    -- First 12 characters of the plaintext, shown in the UI so an operator can
    -- tell two keys apart. It is a display aid, not a secret: 12 characters of
    -- a 256-bit random value carry no more authority than the row id.
    key_prefix VARCHAR(16) NOT NULL,

    -- Denormalised from people.kind on purpose. The authentication path reads
    -- only api_keys, and requiring a join to people on every request to learn
    -- whether the caller is a human or a service would put a second table on
    -- the hot path for a value that cannot change: a key is reissued against a
    -- different principal rather than retyped.
    principal_type VARCHAR(16) NOT NULL,

    -- ON DELETE CASCADE: a principal that no longer exists cannot own a
    -- credential, and leaving orphaned keys behind would keep granting access
    -- to somebody who has been deleted from the people list.
    principal_id UUID NOT NULL REFERENCES people(id) ON DELETE CASCADE,

    -- NULL means "not project-scoped" — the key sees everything the principal
    -- can see. A non-NULL value restricts it to one project, which is the
    -- shape an agent belonging to a single team needs.
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,

    -- Empty array = unrestricted. This is what makes the migration safe for an
    -- existing deployment: a key with no scopes behaves exactly like the old
    -- shared secret, and restricting access becomes an explicit act.
    scopes TEXT[] NOT NULL DEFAULT '{}',

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP,
    revoked_at TIMESTAMP,

    CONSTRAINT api_keys_principal_type_check CHECK (principal_type IN ('human', 'service')),

    -- UNIQUE (not a plain index): the digest is the lookup key, and Postgres
    -- gets uniqueness from the index it must build anyway. A second index on
    -- the same column would only cost write time and disk.
    CONSTRAINT api_keys_key_hash_key UNIQUE (key_hash)
);

-- Listing the keys a principal holds.
CREATE INDEX IF NOT EXISTS idx_api_keys_principal ON api_keys (principal_id);

-- Listing the keys scoped to one project. Partial, because an unscoped key is
-- NULL here and never appears in that query's result set.
CREATE INDEX IF NOT EXISTS idx_api_keys_project ON api_keys (project_id) WHERE project_id IS NOT NULL;

-- The management UI lists keys newest-first with a LIMIT, which is an
-- "ORDER BY created_at DESC LIMIT n" over the whole table. Measured on 20k
-- rows without this index: Seq Scan + top-N sort, 8.7 ms per request, and the
-- cost grows with the table rather than with the page size. The index turns it
-- into a bounded Index Scan that reads only the rows the page will show.
CREATE INDEX IF NOT EXISTS idx_api_keys_created_at ON api_keys (created_at DESC);
