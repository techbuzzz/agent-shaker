package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// ProjectReposStore is the typed query layer for the project_repos table.
type ProjectReposStore struct {
	q Querier
}

func NewProjectReposStore(q Querier) *ProjectReposStore {
	return &ProjectReposStore{q: q}
}

func (s *ProjectReposStore) Create(ctx context.Context, r *models.ProjectRepo) error {
	_, err := s.q.ExecContext(ctx, `
		INSERT INTO project_repos (id, project_id, url, branch, role, agent_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, r.ID, r.ProjectID, r.URL, r.Branch, r.Role, r.AgentID, r.CreatedAt, r.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert project repo: %w", err)
	}
	return nil
}

// ListByProject returns all repos for a project, newest first.
func (s *ProjectReposStore) ListByProject(ctx context.Context, projectID uuid.UUID) ([]models.ProjectRepo, error) {
	rows, err := s.q.QueryContext(ctx, `
		SELECT id, project_id, url, branch, role, agent_id, created_at, updated_at
		FROM project_repos WHERE project_id = $1 ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project repos: %w", err)
	}
	defer rows.Close()

	var out []models.ProjectRepo
	for rows.Next() {
		var r models.ProjectRepo
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.URL, &r.Branch, &r.Role, &r.AgentID, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan project repo: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating project repos: %w", err)
	}
	return out, nil
}

// Get returns one repo by id.
func (s *ProjectReposStore) Get(ctx context.Context, id uuid.UUID) (*models.ProjectRepo, error) {
	var r models.ProjectRepo
	err := s.q.QueryRowContext(ctx, `
		SELECT id, project_id, url, branch, role, agent_id, created_at, updated_at
		FROM project_repos WHERE id = $1
	`, id).Scan(&r.ID, &r.ProjectID, &r.URL, &r.Branch, &r.Role, &r.AgentID, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("project repo %s not found", id)
		}
		return nil, fmt.Errorf("select project repo: %w", err)
	}
	return &r, nil
}

// Update edits the repo URL, branch, role, and (optionally) agent_id.
// agent_id can be cleared by passing a UUID-zero value.
func (s *ProjectReposStore) Update(ctx context.Context, r *models.ProjectRepo) error {
	_, err := s.q.ExecContext(ctx, `
		UPDATE project_repos
		SET url = $1, branch = $2, role = $3, agent_id = $4, updated_at = $5
		WHERE id = $6
	`, r.URL, r.Branch, r.Role, r.AgentID, r.UpdatedAt, r.ID)
	if err != nil {
		return fmt.Errorf("update project repo: %w", err)
	}
	return nil
}

// Delete removes a repo. The owning agent row is NOT touched.
func (s *ProjectReposStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.q.ExecContext(ctx, `DELETE FROM project_repos WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete project repo: %w", err)
	}
	rows, err := tag.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("project repo %s not found", id)
	}
	return nil
}

// GetProjectID is used by handlers to find the broadcast target after a
// repo mutation. Mirrors the GetContextProjectID helper on contexts.
func (s *ProjectReposStore) GetProjectID(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	var pid uuid.UUID
	err := s.q.QueryRowContext(ctx, `SELECT project_id FROM project_repos WHERE id = $1`, id).Scan(&pid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("project repo %s not found", id)
		}
		return uuid.Nil, fmt.Errorf("select project repo project: %w", err)
	}
	return pid, nil
}
