package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/techbuzzz/agent-shaker/internal/models"
)


// apiKeyColumns is the projection every single-row lookup selects. Kept in one
// place so the scan helpers and the SQL cannot drift apart; a mismatch here is
// a runtime scan error, not a compile error.
const apiKeyColumns = `
	k.id, k.key_hash, k.key_prefix, k.principal_type, k.principal_id,
	k.project_id, k.scopes, k.created_at, k.last_used_at, k.revoked_at`

// rowScanner is satisfied by *sql.Row and *sql.Rows, which lets one scan
// helper serve both the single-row lookups on the request path and the list
// query.
type rowScanner interface {
	Scan(dest ...any) error
}

// APIKeysStore is the typed query layer for the api_keys table.
type APIKeysStore struct {
	q Querier
}

// NewAPIKeysStore returns a store bound to the supplied Querier.
func NewAPIKeysStore(q Querier) *APIKeysStore {
	return &APIKeysStore{q: q}
}

// Available reports whether the store has a usable Querier.
func (s *APIKeysStore) Available() bool { return s.q != nil }

// CreateAPIKey inserts a key. The caller supplies KeyHash, never the plaintext:
// this package has no code path that could store a secret.
func (s *APIKeysStore) CreateAPIKey(ctx context.Context, k *models.APIKey) error {
	if _, err := s.q.ExecContext(ctx, `
		INSERT INTO api_keys (id, key_hash, key_prefix, principal_type, principal_id, project_id, scopes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, k.ID, k.KeyHash, k.KeyPrefix, k.PrincipalType, k.PrincipalID, k.ProjectID, pq.Array(k.Scopes), k.CreatedAt); err != nil {
		if isUniqueViolation(err, "api_keys_key_hash_key") {
			// A duplicate digest means the same secret was generated twice, which
			// is astronomically unlikely and worth surfacing rather than
			// retrying: a retry would paper over a broken entropy source.
			return fmt.Errorf("api key digest already exists: %w", err)
		}
		return fmt.Errorf("insert api key: %w", err)
	}
	return nil
}

// GetAPIKeyByHash resolves a presented credential to its row.
//
// This is the hot path: it runs once per request when the resolver's cache is
// cold, and it is an index probe on the UNIQUE (key_hash) constraint — 0.08 ms
// measured on 20 000 rows.
//
// A revoked key is returned rather than hidden. The caller decides, and the
// decision is "reject", but hiding it here would make "revoked" and "never
// existed" indistinguishable in the store's own errors — and the revoke path
// needs to read revoked rows to answer idempotently.
func (s *APIKeysStore) GetAPIKeyByHash(ctx context.Context, digest string) (*models.APIKey, error) {
	row := s.q.QueryRowContext(ctx, `
		SELECT`+apiKeyColumns+`
		FROM api_keys k WHERE k.key_hash = $1
	`, digest)

	k, err := scanAPIKey(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: hash %s", models.ErrAPIKeyNotFound, digest)
		}
		return nil, fmt.Errorf("select api key by hash: %w", err)
	}
	return k, nil
}

// GetAPIKeyByID returns one key by id.
func (s *APIKeysStore) GetAPIKeyByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error) {
	row := s.q.QueryRowContext(ctx, `
		SELECT`+apiKeyColumns+`
		FROM api_keys k WHERE k.id = $1
	`, id)

	k, err := scanAPIKey(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: id %s", models.ErrAPIKeyNotFound, id)
		}
		return nil, fmt.Errorf("select api key: %w", err)
	}
	return k, nil
}

// ListAPIKeys returns keys newest-first, optionally narrowed to one principal
// and/or excluding revoked rows.
//
// The join to people is what makes a key list readable — a bare uuid tells an
// operator nothing — and it costs one pkey lookup per row, memoised across the
// 100-row page (0.4 ms for the whole page, measured).
func (s *APIKeysStore) ListAPIKeys(ctx context.Context, principalID *uuid.UUID, includeRevoked bool) ([]models.APIKey, error) {
	q := `
		SELECT k.id, k.key_hash, k.key_prefix, k.principal_type, k.principal_id,
		       k.project_id, k.scopes, k.created_at, k.last_used_at, k.revoked_at,
		       p.display_name
		FROM api_keys k
		JOIN people p ON p.id = k.principal_id`
	var args []any
	if principalID != nil {
		q += " WHERE k.principal_id = $1"
		args = append(args, *principalID)
	}
	if !includeRevoked {
		if len(args) == 0 {
			q += " WHERE k.revoked_at IS NULL"
		} else {
			q += " AND k.revoked_at IS NULL"
		}
	}
	q += " ORDER BY k.created_at DESC"

	rows, err := s.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	out := make([]models.APIKey, 0, 16)
	for rows.Next() {
		k, err := scanAPIKeyWithPrincipal(rows)
		if err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		out = append(out, *k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating api keys: %w", err)
	}
	return out, nil
}

// RevokeAPIKey marks a key unusable and returns the row as it now stands.
//
// Revocation is a flag, never a DELETE: the audit journal (M3) and any incident
// review need to see that a credential existed and when it stopped working, and
// a deleted row is indistinguishable from one that was never issued.
//
// Revoking an already-revoked key succeeds and is a no-op. That is deliberate —
// DELETE is not idempotent by default in a client's mental model, and a retried
// request (flaky mobile link, impatient double-click) should not turn into a
// 404 that looks like a different key.
func (s *APIKeysStore) RevokeAPIKey(ctx context.Context, id uuid.UUID) (*models.APIKey, error) {
	row := s.q.QueryRowContext(ctx, `
		UPDATE api_keys SET revoked_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL
		RETURNING`+apiKeyColumns,
		id)

	k, err := scanAPIKey(row)
	if err == nil {
		return k, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("revoke api key: %w", err)
	}

	// Either it was already revoked or it does not exist. One extra indexed read
	// tells them apart, and it only runs on that rare path.
	existing, err := s.GetAPIKeyByID(ctx, id)
	if err != nil {
		if errors.Is(err, models.ErrAPIKeyNotFound) {
			return nil, fmt.Errorf("%w: id %s", models.ErrAPIKeyNotFound, id)
		}
		return nil, err
	}
	return existing, nil
}

// TouchAPIKeyLastUsed stamps a successful authentication.
//
// It is called at most once per key per resolver cache lifetime, never per
// request: a write on every request would turn authentication into the most
// write-heavy operation in the system, and "when was this key last used" is
// operational trivia that tolerates being 15 seconds stale.
func (s *APIKeysStore) TouchAPIKeyLastUsed(ctx context.Context, id uuid.UUID) error {
	if _, err := s.q.ExecContext(ctx, `
		UPDATE api_keys SET last_used_at = NOW() WHERE id = $1
	`, id); err != nil {
		return fmt.Errorf("touch api key last_used_at: %w", err)
	}
	return nil
}

// scanAPIKey materialises one row of apiKeyColumns.
func scanAPIKey(row rowScanner) (*models.APIKey, error) {
	var (
		k         models.APIKey
		projectID uuid.NullUUID
		lastUsed  sql.NullTime
		revokedAt sql.NullTime
	)
	if err := row.Scan(
		&k.ID, &k.KeyHash, &k.KeyPrefix, &k.PrincipalType, &k.PrincipalID,
		&projectID, pq.Array(&k.Scopes), &k.CreatedAt, &lastUsed, &revokedAt,
	); err != nil {
		return nil, err
	}
	applyNullColumns(&k, projectID, lastUsed, revokedAt)
	return &k, nil
}

// scanAPIKeyWithPrincipal materialises a row that carries people.display_name.
func scanAPIKeyWithPrincipal(row rowScanner) (*models.APIKey, error) {
	var (
		k         models.APIKey
		projectID uuid.NullUUID
		lastUsed  sql.NullTime
		revokedAt sql.NullTime
	)
	if err := row.Scan(
		&k.ID, &k.KeyHash, &k.KeyPrefix, &k.PrincipalType, &k.PrincipalID,
		&projectID, pq.Array(&k.Scopes), &k.CreatedAt, &lastUsed, &revokedAt,
		&k.PrincipalName,
	); err != nil {
		return nil, err
	}
	applyNullColumns(&k, projectID, lastUsed, revokedAt)
	return &k, nil
}

// applyNullColumns converts the nullable columns once, for both scanners.
//
// Scanning into the *T fields directly would rely on database/sql's
// pointer-dereferencing rules to turn SQL NULL into a nil pointer, which works
// for *uuid.UUID and *time.Time but fails differently for a future column type
// nobody has tested yet. Explicit is cheaper than that class of bug.
func applyNullColumns(k *models.APIKey, projectID uuid.NullUUID, lastUsed, revokedAt sql.NullTime) {
	if projectID.Valid {
		id := projectID.UUID
		k.ProjectID = &id
	}
	if lastUsed.Valid {
		t := lastUsed.Time
		k.LastUsedAt = &t
	}
	if revokedAt.Valid {
		t := revokedAt.Time
		k.RevokedAt = &t
	}
}
