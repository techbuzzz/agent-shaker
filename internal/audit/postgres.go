package audit

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/database"
)

// PostgresRecorder writes the journal, chained.
//
// The chain is per project: one project's traffic is one chain. A project's
// head hash is published outside this database — to a file the operator
// names, or through the API for them to pin somewhere — so that whoever
// rewrites the trail cannot rewrite the anchor with it.
//
// The chain is what makes "nobody can quietly rewrote it" checkable rather
// than merely intended: editing or removing any row breaks every hash after
// it, and the head is the only thing an attacker cannot recompute quietly.
type PostgresRecorder struct {
	db database.Querier

	// anchorDir is where the head hash is written, when the operator named a
	// directory. Empty means the head is only readable through the API.
	anchorDir string
}

// NewPostgresRecorder returns a recorder over the given database.
//
// The anchor directory is optional: with none, the chain is still written and
// still verifiable, but the head has to be pinned by hand.
func NewPostgresRecorder(db database.Querier, anchorDir string) *PostgresRecorder {
	return &PostgresRecorder{db: db, anchorDir: anchorDir}
}

// Record writes one event.
//
// A failure is returned, not swallowed: the decision to ignore it belongs to
// the caller, and the caller is the broker subscriber, which knows that a
// journal write must not fail the operation that caused it.
func (r *PostgresRecorder) Record(ctx context.Context, ev Event) error {
	// The action is checked first, before the database: an event with no
	// action is a programming error, and it should read as one whether or not
	// there is a database to write it to.
	if ev.Action == "" {
		return errors.New("an audit event needs an action")
	}
	if r == nil || r.db == nil {
		return nil
	}

	occurredAt := ev.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin audit transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// One writer per chain at a time. Without it, two writers read the same
	// head and produce two rows claiming the same predecessor — a fork, which
	// is exactly what the chain exists to prevent.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, lockKey(ev.ProjectID)); err != nil {
		return fmt.Errorf("lock audit chain: %w", err)
	}

	prevHash, err := headHash(ctx, tx, ev.ProjectID)
	if err != nil {
		return err
	}

	hash, err := chainHash(prevHash, ev, occurredAt)
	if err != nil {
		return err
	}

	before, err := jsonColumn(ev.Before)
	if err != nil {
		return fmt.Errorf("encode audit before: %w", err)
	}
	after, err := jsonColumn(ev.After)
	if err != nil {
		return fmt.Errorf("encode audit after: %w", err)
	}

	var projectID any
	if ev.ProjectID != uuid.Nil {
		projectID = ev.ProjectID
	}
	var entityID any
	if ev.EntityID != uuid.Nil {
		entityID = ev.EntityID
	}
	var actorID any
	if ev.ActorID != uuid.Nil {
		actorID = ev.ActorID
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO audit_events (
			occurred_at, actor_type, actor_id, project_id, entity_type, entity_id,
			action, before, after, trace_id, request_id, prev_hash, hash
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`,
		occurredAt, nullString(ev.ActorType), actorID, projectID, nullString(ev.EntityType), entityID,
		ev.Action, before, after, nullString(ev.TraceID), nullString(ev.RequestID), nullString(prevHash), hash,
	); err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit audit event: %w", err)
	}

	r.writeAnchor(ev.ProjectID, hash)
	return nil
}

// headHash returns the hash of the newest row in a chain, or the empty string
// when the chain has no rows yet.
func headHash(ctx context.Context, tx *sql.Tx, projectID uuid.UUID) (string, error) {
	var hash sql.NullString

	query := `SELECT hash FROM audit_events WHERE project_id = $1 AND hash IS NOT NULL ORDER BY id DESC LIMIT 1`
	args := []any{projectID}
	if projectID == uuid.Nil {
		query = `SELECT hash FROM audit_events WHERE project_id IS NULL AND hash IS NOT NULL ORDER BY id DESC LIMIT 1`
		args = nil
	}

	err := tx.QueryRowContext(ctx, query, args...).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read audit chain head: %w", err)
	}
	return hash.String, nil
}

// chainHash computes a row's hash from its predecessor and its own content.
//
// The canonical encoding is JSON with the keys in a fixed order, so two
// writers that read the same row compute the same hash. Go's encoder sorts map
// keys, which is what makes the snapshot fields deterministic.
func chainHash(prevHash string, ev Event, occurredAt time.Time) (string, error) {
	payload := struct {
		PrevHash   string         `json:"prev_hash"`
		Action     string         `json:"action"`
		ProjectID  string         `json:"project_id"`
		EntityType string         `json:"entity_type"`
		EntityID   string         `json:"entity_id"`
		ActorType  string         `json:"actor_type"`
		ActorID    string         `json:"actor_id"`
		Before     map[string]any `json:"before"`
		After      map[string]any `json:"after"`
		TraceID    string         `json:"trace_id"`
		RequestID  string         `json:"request_id"`
		OccurredAt string         `json:"occurred_at"`
	}{
		PrevHash:   prevHash,
		Action:     ev.Action,
		ProjectID:  uuidString(ev.ProjectID),
		EntityType: ev.EntityType,
		EntityID:   uuidString(ev.EntityID),
		ActorType:  ev.ActorType,
		ActorID:    uuidString(ev.ActorID),
		Before:     ev.Before,
		After:      ev.After,
		TraceID:    ev.TraceID,
		RequestID:  ev.RequestID,
		OccurredAt: occurredAt.UTC().Format(time.RFC3339Nano),
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode audit event: %w", err)
	}

	// The separator keeps "no predecessor" and "a predecessor" from hashing
	// to the same thing.
	sum := sha256.Sum256(append([]byte(prevHash+"\n"), raw...))
	return hex.EncodeToString(sum[:]), nil
}

// writeAnchor publishes the head of a chain.
//
// It is best effort and it is logged when it fails: an anchor that could not
// be written is a gap in the operator's copy of the trail, and hiding that
// would be worse than the gap.
func (r *PostgresRecorder) writeAnchor(projectID uuid.UUID, hash string) {
	if r.anchorDir == "" || hash == "" {
		return
	}

	path := filepath.Join(r.anchorDir, projectID.String()+".anchor")
	content := fmt.Sprintf("%s  %s\n", hash, time.Now().UTC().Format(time.RFC3339))

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		slog.Error("could not write the audit anchor",
			"error", err,
			"path", path,
		)
	}
}

// lockKey turns a project id into an advisory lock key.
//
// The first eight bytes of the UUID, read as a big-endian integer. Taken from
// the UUID rather than hashed in the database, so the lock does not depend on
// an internal Postgres function.
func lockKey(projectID uuid.UUID) int64 {
	if projectID == uuid.Nil {
		return 0
	}
	return int64(binary.BigEndian.Uint64(projectID[:8]))
}

// jsonColumn encodes an optional snapshot.
//
// A nil map becomes SQL NULL rather than the JSON literal "null", so "no
// before" and "an empty before" stay different things.
func jsonColumn(value map[string]any) (any, error) {
	if value == nil {
		return nil, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// nullString turns an empty string into SQL NULL.
func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// uuidString renders a uuid, with the zero uuid as the empty string.
func uuidString(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}
