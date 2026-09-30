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

// GlobalContextsStore is the typed query layer for the global_contexts table.
type GlobalContextsStore struct {
	q Querier
}

func NewGlobalContextsStore(q Querier) *GlobalContextsStore {
	return &GlobalContextsStore{q: q}
}

// Create inserts a new global_contexts row. The DB enforces uniqueness on
// (title) when scope='global' and (project_id, title) when scope='project';
// a duplicate returns a Postgres unique-violation wrapped in the error.
func (s *GlobalContextsStore) Create(ctx context.Context, g *models.GlobalContext) error {
	_, err := s.q.ExecContext(ctx, `
		INSERT INTO global_contexts (id, scope, project_id, agent_id, title, content, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, g.ID, string(g.Scope), g.ProjectID, g.AgentID, g.Title, g.Content, pq.Array(g.Tags), g.CreatedAt, g.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert global context: %w", err)
	}
	return nil
}

// List returns docs filtered by scope + optional project_id + optional tags.
//
//   - scope="global"  → returns all server-wide docs (project_id ignored).
//   - scope="project" → returns project-scoped docs (project_id required).
//   - scope=""        → returns both, project_id narrows the project slice.
//
// tagPrefix (when non-empty) restricts to docs whose `tags` array contains
// any tag starting with that prefix; this is what powers the
// "feature:auth" grouping without adding a dedicated features table.
func (s *GlobalContextsStore) List(ctx context.Context, scope models.GlobalContextScope, projectID *uuid.UUID, tagPrefix string) ([]models.GlobalContext, error) {
	args := []any{}
	q := `SELECT id, scope, project_id, agent_id, title, content, tags, created_at, updated_at FROM global_contexts WHERE 1=1`
	if scope != "" {
		q += fmt.Sprintf(" AND scope = $%d", len(args)+1)
		args = append(args, string(scope))
	}
	if projectID != nil {
		q += fmt.Sprintf(" AND project_id = $%d", len(args)+1)
		args = append(args, *projectID)
	}
	if tagPrefix != "" {
		// EXISTS subquery: any tag starts with the prefix.
		q += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM unnest(tags) tag WHERE tag LIKE $%d)", len(args)+1)
		args = append(args, tagPrefix+"%")
	}
	q += " ORDER BY updated_at DESC"

	rows, err := s.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list global contexts: %w", err)
	}
	defer rows.Close()

	var out []models.GlobalContext
	for rows.Next() {
		var g models.GlobalContext
		var scopeStr string
		if err := rows.Scan(&g.ID, &scopeStr, &g.ProjectID, &g.AgentID, &g.Title, &g.Content, pq.Array(&g.Tags), &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan global context: %w", err)
		}
		g.Scope = models.GlobalContextScope(scopeStr)
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating global contexts: %w", err)
	}
	return out, nil
}

// Get returns one doc by id.
func (s *GlobalContextsStore) Get(ctx context.Context, id uuid.UUID) (*models.GlobalContext, error) {
	var g models.GlobalContext
	var scopeStr string
	err := s.q.QueryRowContext(ctx, `
		SELECT id, scope, project_id, agent_id, title, content, tags, created_at, updated_at
		FROM global_contexts WHERE id = $1
	`, id).Scan(&g.ID, &scopeStr, &g.ProjectID, &g.AgentID, &g.Title, &g.Content, pq.Array(&g.Tags), &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("global context %s not found", id)
		}
		return nil, fmt.Errorf("select global context: %w", err)
	}
	g.Scope = models.GlobalContextScope(scopeStr)
	return &g, nil
}

// GetByGlobalTitle returns a scope='global' doc by title. Used by the MCP
// `read_resource("global://<title>")` path.
func (s *GlobalContextsStore) GetByGlobalTitle(ctx context.Context, title string) (*models.GlobalContext, error) {
	var g models.GlobalContext
	var scopeStr string
	err := s.q.QueryRowContext(ctx, `
		SELECT id, scope, project_id, agent_id, title, content, tags, created_at, updated_at
		FROM global_contexts WHERE scope = 'global' AND title = $1
	`, title).Scan(&g.ID, &scopeStr, &g.ProjectID, &g.AgentID, &g.Title, &g.Content, pq.Array(&g.Tags), &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("global context %q not found", title)
		}
		return nil, fmt.Errorf("select global context by title: %w", err)
	}
	g.Scope = models.GlobalContextScope(scopeStr)
	return &g, nil
}

// Update edits mutable fields (content, tags).
func (s *GlobalContextsStore) Update(ctx context.Context, g *models.GlobalContext) error {
	_, err := s.q.ExecContext(ctx, `
		UPDATE global_contexts SET content = $1, tags = $2, updated_at = $3 WHERE id = $4
	`, g.Content, pq.Array(g.Tags), g.UpdatedAt, g.ID)
	if err != nil {
		return fmt.Errorf("update global context: %w", err)
	}
	return nil
}

// Delete removes a doc.
func (s *GlobalContextsStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.q.ExecContext(ctx, `DELETE FROM global_contexts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete global context: %w", err)
	}
	rows, err := tag.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("global context %s not found", id)
	}
	return nil
}
