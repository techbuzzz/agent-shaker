package queries

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// AuditStore is the typed query layer for the audit_events table.
//
// There is no update and no delete here, and there will not be one. The
// journal is append-only; a method that rewrote a row would be a way to
// rewrite history.
type AuditStore struct {
	q Querier
}

// NewAuditStore returns a store bound to the supplied Querier.
func NewAuditStore(q Querier) *AuditStore {
	return &AuditStore{q: q}
}

// Available reports whether the store has a usable Querier.
func (s *AuditStore) Available() bool { return s.q != nil }

// AuditFilter narrows a journal read.
type AuditFilter struct {
	ProjectID  uuid.UUID
	EntityType string
	EntityID   *uuid.UUID
	Action     string
	Limit      int
}

// ListAuditEvents returns the newest journal rows, newest first.
//
// The project is required. A journal spans every project in the deployment,
// and a reader with no project scope would be reading all of them.
func (s *AuditStore) ListAuditEvents(ctx context.Context, filter AuditFilter) ([]models.AuditEvent, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	rows, err := s.q.QueryContext(ctx, `
		SELECT id, occurred_at, actor_type, actor_id, project_id, entity_type,
			entity_id, action, before, after, trace_id, request_id, prev_hash, hash
		FROM audit_events
		WHERE project_id = $1
		  AND ($2 = '' OR entity_type = $2)
		  AND ($3::uuid IS NULL OR entity_id = $3)
		  AND ($4 = '' OR action = $4)
		ORDER BY id DESC
		LIMIT $5
	`,
		filter.ProjectID, filter.EntityType, filter.EntityID, filter.Action, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()

	out := make([]models.AuditEvent, 0)
	for rows.Next() {
		var (
			event      models.AuditEvent
			before     []byte
			after      []byte
			actorType  sql.NullString
			entityType sql.NullString
			traceID    sql.NullString
			requestID  sql.NullString
			prevHash   sql.NullString
			hash       sql.NullString
		)
		if err := rows.Scan(
			&event.ID, &event.OccurredAt, &actorType, &event.ActorID,
			&event.ProjectID, &entityType, &event.EntityID, &event.Action,
			&before, &after, &traceID, &requestID, &prevHash, &hash,
		); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		event.ActorType = actorType.String
		event.EntityType = entityType.String
		event.TraceID = traceID.String
		event.RequestID = requestID.String
		event.PrevHash = prevHash.String
		event.Hash = hash.String
		event.Before = decodeSnapshot(before)
		event.After = decodeSnapshot(after)
		out = append(out, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating audit events: %w", err)
	}
	return out, nil
}

// AuditChainHead returns the head of a project's chain.
//
// An empty result means the project has no journalled events yet, which is
// not an error: a project whose coordination plane has never fired has no
// trail to publish.
func (s *AuditStore) AuditChainHead(ctx context.Context, projectID uuid.UUID) (*models.AuditChainHead, error) {
	head := &models.AuditChainHead{ProjectID: projectID}

	var rowID int64
	err := s.q.QueryRowContext(ctx, `
		SELECT id, hash, occurred_at FROM audit_events
		WHERE project_id = $1 AND hash IS NOT NULL
		ORDER BY id DESC LIMIT 1
	`, projectID).Scan(&rowID, &head.HeadHash, &head.RecordedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("read audit chain head: %w", err)
	}

	head.RowID = rowID
	return head, nil
}

// decodeSnapshot turns a stored JSONB column into a map.
//
// A column that does not decode yields an empty map rather than an error: the
// journal is a record of what somebody else wrote, and refusing to read it
// would make the whole trail unavailable over one bad row.
func decodeSnapshot(raw []byte) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return map[string]any{}
	}
	return decoded
}
