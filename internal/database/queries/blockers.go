package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// BlockersStore is the typed query layer for the `blockers` table.
//
// Every mutating method takes an explicit Querier so a handler can compose
// several of them inside one transaction. A blocker that is resolved without
// also releasing its task and agent is the failure mode this shape exists to
// prevent: the human sees "resolved" while the agent stays blocked forever.
type BlockersStore struct {
	q Querier
}

func NewBlockersStore(q Querier) *BlockersStore {
	return &BlockersStore{q: q}
}

// Available reports whether the store has a usable Querier.
func (s *BlockersStore) Available() bool { return s.q != nil }

// ErrBlockerClosed is returned when a mutation targets a blocker that is
// already resolved or dismissed.
//
// It is a sentinel rather than a string so the handler can answer 409 without
// matching on message text — the same thing `isNotFound` does badly in this
// package today.
var ErrBlockerClosed = errors.New("blocker is already closed")

// CreateBlocker inserts a blocker and returns the stored row.
func (s *BlockersStore) CreateBlocker(ctx context.Context, b *models.Blocker) error {
	_, err := s.q.ExecContext(ctx, `
		INSERT INTO blockers (id, task_id, agent_id, project_id, blocker_type, title, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'open', NOW(), NOW())
	`, b.ID, b.TaskID, b.AgentID, b.ProjectID, string(b.Type), b.Title, b.Description)
	if err != nil {
		return fmt.Errorf("insert blocker: %w", err)
	}
	return nil
}

// GetBlocker returns one blocker by id.
func (s *BlockersStore) GetBlocker(ctx context.Context, q Querier, id uuid.UUID) (*models.Blocker, error) {
	if q == nil {
		q = s.q
	}
	var b models.Blocker
	var (
		description sql.NullString
		resolution  sql.NullString
		resolvedBy  uuid.NullUUID
		resolvedAt  sql.NullTime
		createdAt   sql.NullTime
		updatedAt   sql.NullTime
	)
	err := q.QueryRowContext(ctx, `
		SELECT id, task_id, agent_id, project_id, blocker_type, title, description, status,
		       resolution, resolved_by, resolved_at, created_at, updated_at
		FROM blockers WHERE id = $1
	`, id).Scan(
		&b.ID, &b.TaskID, &b.AgentID, &b.ProjectID, &b.Type, &b.Title, &description, &b.Status,
		&resolution, &resolvedBy, &resolvedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("blocker %s not found", id)
		}
		return nil, fmt.Errorf("select blocker: %w", err)
	}
	b.Description = description.String
	if resolution.Valid {
		b.Resolution = &resolution.String
	}
	if resolvedBy.Valid {
		v := resolvedBy.UUID
		b.ResolvedBy = &v
	}
	if resolvedAt.Valid {
		v := resolvedAt.Time
		b.ResolvedAt = &v
	}
	b.CreatedAt = createdAt.Time
	b.UpdatedAt = updatedAt.Time
	return &b, nil
}

// ListBlockers returns a project's blockers, optionally filtered by status.
// An empty status returns every blocker; the list is oldest-first because the
// question a human opens this screen with is "what has been waiting longest".
func (s *BlockersStore) ListBlockers(ctx context.Context, projectID uuid.UUID, status string) ([]models.Blocker, error) {
	q := `
		SELECT id, task_id, agent_id, project_id, blocker_type, title, description, status,
		       resolution, resolved_by, resolved_at, created_at, updated_at
		FROM blockers WHERE project_id = $1`
	args := []any{projectID}
	if status != "" {
		q += " AND status = $2"
		args = append(args, status)
	}
	q += " ORDER BY created_at ASC"

	rows, err := s.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list blockers: %w", err)
	}
	defer rows.Close()

	out := []models.Blocker{}
	for rows.Next() {
		var b models.Blocker
		var (
			description sql.NullString
			resolution  sql.NullString
			resolvedBy  uuid.NullUUID
			resolvedAt  sql.NullTime
			createdAt   sql.NullTime
			updatedAt   sql.NullTime
		)
		if err := rows.Scan(
			&b.ID, &b.TaskID, &b.AgentID, &b.ProjectID, &b.Type, &b.Title, &description, &b.Status,
			&resolution, &resolvedBy, &resolvedAt, &createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan blocker: %w", err)
		}
		b.Description = description.String
		if resolution.Valid {
			b.Resolution = &resolution.String
		}
		if resolvedBy.Valid {
			v := resolvedBy.UUID
			b.ResolvedBy = &v
		}
		if resolvedAt.Valid {
			v := resolvedAt.Time
			b.ResolvedAt = &v
		}
		b.CreatedAt = createdAt.Time
		b.UpdatedAt = updatedAt.Time
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating blockers: %w", err)
	}
	return out, nil
}

// ListBlockersByAgent returns the blockers one agent raised, newest first.
// This is the read side of "get_my_blockers": an agent polls it after
// reconnecting, having missed the WebSocket push while it was down.
func (s *BlockersStore) ListBlockersByAgent(ctx context.Context, agentID uuid.UUID, status string) ([]models.Blocker, error) {
	q := `
		SELECT id, task_id, agent_id, project_id, blocker_type, title, description, status,
		       resolution, resolved_by, resolved_at, created_at, updated_at
		FROM blockers WHERE agent_id = $1`
	args := []any{agentID}
	if status != "" {
		q += " AND status = $2"
		args = append(args, status)
	}
	q += " ORDER BY created_at DESC"

	rows, err := s.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list agent blockers: %w", err)
	}
	defer rows.Close()

	out := []models.Blocker{}
	for rows.Next() {
		var b models.Blocker
		var (
			description sql.NullString
			resolution  sql.NullString
			resolvedBy  uuid.NullUUID
			resolvedAt  sql.NullTime
			createdAt   sql.NullTime
			updatedAt   sql.NullTime
		)
		if err := rows.Scan(
			&b.ID, &b.TaskID, &b.AgentID, &b.ProjectID, &b.Type, &b.Title, &description, &b.Status,
			&resolution, &resolvedBy, &resolvedAt, &createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan blocker: %w", err)
		}
		b.Description = description.String
		if resolution.Valid {
			b.Resolution = &resolution.String
		}
		if resolvedBy.Valid {
			v := resolvedBy.UUID
			b.ResolvedBy = &v
		}
		if resolvedAt.Valid {
			v := resolvedAt.Time
			b.ResolvedAt = &v
		}
		b.CreatedAt = createdAt.Time
		b.UpdatedAt = updatedAt.Time
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating agent blockers: %w", err)
	}
	return out, nil
}

// CloseBlocker moves a blocker to resolved or dismissed.
//
// The `AND status IN ('open','in_review')` guard is the whole point: two humans
// answering the same blocker at once must not produce two resolutions, and the
// loser has to be told rather than silently overwriting the winner.
func (s *BlockersStore) CloseBlocker(ctx context.Context, q Querier, id uuid.UUID, status models.BlockerStatus, resolution string, resolvedBy *uuid.UUID) (closed bool, err error) {
	if q == nil {
		q = s.q
	}
	tag, err := q.ExecContext(ctx, `
		UPDATE blockers
		SET status = $1,
		    resolution = $2,
		    resolved_by = $3,
		    resolved_at = NOW(),
		    updated_at = NOW()
		WHERE id = $4 AND status IN ('open', 'in_review')
	`, string(status), resolution, resolvedBy, id)
	if err != nil {
		return false, fmt.Errorf("close blocker: %w", err)
	}
	rows, err := tag.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return false, ErrBlockerClosed
	}
	return true, nil
}

// CountOpenBlockersForTask counts a task's blockers that still need attention.
// The task and agent release in the resolve flow is gated on this being zero,
// so it must count both 'open' and 'in_review': an in-review blocker has a human
// looking at it and the agent is still waiting.
func (s *BlockersStore) CountOpenBlockersForTask(ctx context.Context, q Querier, taskID uuid.UUID) (int, error) {
	if q == nil {
		q = s.q
	}
	var n int
	err := q.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM blockers WHERE task_id = $1 AND status IN ('open', 'in_review')
	`, taskID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count open blockers: %w", err)
	}
	return n, nil
}

// CountOpenBlockersForAgent is the agent-level release gate: an agent whose
// last blocker was answered goes back to idle, one that still has another
// blocker open does not.
func (s *BlockersStore) CountOpenBlockersForAgent(ctx context.Context, q Querier, agentID uuid.UUID) (int, error) {
	if q == nil {
		q = s.q
	}
	var n int
	err := q.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM blockers WHERE agent_id = $1 AND status IN ('open', 'in_review')
	`, agentID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count open blockers for agent: %w", err)
	}
	return n, nil
}

// BeginTx starts a transaction; see ProjectsStore.BeginTx for the same shape.
func (s *BlockersStore) BeginTx(ctx context.Context) (*sql.Tx, error) {
	tx, ok := s.q.(txBeginner)
	if !ok {
		return nil, errors.New("BlockersStore: underlying Querier does not support transactions; pass *sql.DB")
	}
	return tx.BeginTx(ctx, nil)
}
