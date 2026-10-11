package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// ContextsStore is the typed query layer for the contexts table.
type ContextsStore struct {
	q Querier
}

func NewContextsStore(q Querier) *ContextsStore {
	return &ContextsStore{q: q}
}

// Available reports whether the store has a usable Querier.
func (s *ContextsStore) Available() bool { return s.q != nil }

func (s *ContextsStore) CreateContext(ctx context.Context, c *models.Context) error {
	_, err := s.q.ExecContext(ctx, `
		INSERT INTO contexts (id, project_id, agent_id, task_id, title, content, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, c.ID, c.ProjectID, c.AgentID, c.TaskID, c.Title, c.Content, pq.Array(c.Tags), c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert context: %w", err)
	}
	return nil
}

// ListContexts returns contexts filtered by project (required), with
// optional task and tags filters. The tags filter uses Postgres' `&&`
// operator so any matching tag triggers inclusion.
func (s *ContextsStore) ListContexts(ctx context.Context, projectID uuid.UUID, taskID *uuid.UUID, tags []string) ([]models.Context, error) {
	args := []any{projectID}
	q := `SELECT id, project_id, agent_id, task_id, title, content, tags, created_at, updated_at FROM contexts WHERE project_id = $1`
	if taskID != nil {
		q += fmt.Sprintf(" AND task_id = $%d", len(args)+1)
		args = append(args, *taskID)
	}
	if len(tags) > 0 {
		q += fmt.Sprintf(" AND tags && $%d", len(args)+1)
		args = append(args, pq.Array(tags))
	}
	q += " ORDER BY created_at DESC"

	rows, err := s.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list contexts: %w", err)
	}
	defer rows.Close()

	var out []models.Context
	for rows.Next() {
		var c models.Context
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.AgentID, &c.TaskID, &c.Title, &c.Content, pq.Array(&c.Tags), &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan context: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating contexts: %w", err)
	}
	return out, nil
}

func (s *ContextsStore) GetContext(ctx context.Context, id uuid.UUID) (*models.Context, error) {
	var c models.Context
	err := s.q.QueryRowContext(ctx, `
		SELECT id, project_id, agent_id, task_id, title, content, tags, created_at, updated_at
		FROM contexts WHERE id = $1
	`, id).Scan(&c.ID, &c.ProjectID, &c.AgentID, &c.TaskID, &c.Title, &c.Content, pq.Array(&c.Tags), &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("context %s not found", id)
		}
		return nil, fmt.Errorf("select context: %w", err)
	}
	return &c, nil
}

func (s *ContextsStore) UpdateContext(ctx context.Context, c *models.Context) error {
	_, err := s.q.ExecContext(ctx, `
		UPDATE contexts
		SET task_id = $1, title = $2, content = $3, tags = $4, updated_at = $5
		WHERE id = $6
	`, c.TaskID, c.Title, c.Content, pq.Array(c.Tags), c.UpdatedAt, c.ID)
	if err != nil {
		return fmt.Errorf("update context: %w", err)
	}
	return nil
}

func (s *ContextsStore) DeleteContext(ctx context.Context, id uuid.UUID) error {
	tag, err := s.q.ExecContext(ctx, `DELETE FROM contexts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete context: %w", err)
	}
	rows, err := tag.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("context %s not found", id)
	}
	return nil
}

// GetContextProjectID returns the project_id for a context id (used by
// the delete broadcast).
func (s *ContextsStore) GetContextProjectID(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	var pid uuid.UUID
	err := s.q.QueryRowContext(ctx, `SELECT project_id FROM contexts WHERE id = $1`, id).Scan(&pid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("context %s not found", id)
		}
		return uuid.Nil, fmt.Errorf("select context project: %w", err)
	}
	return pid, nil
}

// Search limits for SearchContext. They are exported so the HTTP layer and the
// MCP tool agree on the same bounds instead of each inventing their own.
const (
	// DefaultSearchLimit is what a caller gets when it does not ask for a size.
	DefaultSearchLimit = 20

	// MaxSearchLimit caps what a caller may ask for. Without a cap, one
	// request can pull an entire project into memory and into an agent's
	// context window, which is the failure this search exists to avoid.
	MaxSearchLimit = 100
)

// SearchContexts finds contexts within a project whose title or body matches
// the query, most relevant first.
//
// The search is scoped to a project rather than global because that is the
// unit of isolation in the product: a context is a project's knowledge, and a
// cross-project hit would leak one team's notes into another team's task.
//
// An empty or whitespace-only query returns no results without touching the
// database. That is a deliberate short-circuit rather than an optimisation:
// websearch_to_tsquery on an empty string yields an empty query, and asking a
// GIN index about an empty query is a way to make "return nothing" cost a
// round trip.
func (s *ContextsStore) SearchContexts(ctx context.Context, projectID uuid.UUID, query string, limit int) ([]models.Context, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return []models.Context{}, nil
	}

	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}

	// websearch_to_tsquery never raises a syntax error, which matters here:
	// the query text comes from an agent, and an agent pasting a stray quote
	// should get fewer results rather than an error.
	rows, err := s.q.QueryContext(ctx, `
		SELECT id, project_id, agent_id, task_id, title, content, tags, created_at, updated_at
		FROM contexts
		WHERE project_id = $1
		  AND search_vector @@ websearch_to_tsquery('simple', $2)
		ORDER BY ts_rank(search_vector, websearch_to_tsquery('simple', $2)) DESC, created_at DESC
		LIMIT $3
	`, projectID, trimmed, limit)
	if err != nil {
		return nil, fmt.Errorf("search contexts: %w", err)
	}
	defer rows.Close()

	var out []models.Context
	for rows.Next() {
		var c models.Context
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.AgentID, &c.TaskID, &c.Title, &c.Content, pq.Array(&c.Tags), &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan context: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating contexts: %w", err)
	}
	if out == nil {
		return []models.Context{}, nil
	}
	return out, nil
}
