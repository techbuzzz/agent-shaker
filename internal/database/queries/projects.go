// Package queries contains hand-rolled typed query helpers backed by a
// database/sql-compatible Querier (database.Querier in internal/database).
// The driver is pgx/v5/stdlib so pgx is the actual Postgres driver while
// handlers continue to consume the familiar database/sql API.
//
// pgxpool.Pool is used directly by callers that want native pgx semantics
// (e.g. internal/task/store_postgres.go).
package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// Querier is the slice of database/sql the application layer depends on.
// The interface matches database.Querier; callers should pass either a
// *sql.DB or *sql.Tx.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// ProjectsStore is the typed query layer for the projects table.
type ProjectsStore struct {
	q Querier
}

// NewProjectsStore returns a store bound to the supplied Querier.
func NewProjectsStore(q Querier) *ProjectsStore {
	return &ProjectsStore{q: q}
}

// Available reports whether the store has a usable Querier. Used by
// handler methods to short-circuit with 503 when the server is running
// without a database (degraded mode).
func (s *ProjectsStore) Available() bool { return s.q != nil }

// CreateProject inserts a new project row.
func (s *ProjectsStore) CreateProject(ctx context.Context, p *models.Project) error {
	_, err := s.q.ExecContext(ctx, `
		INSERT INTO projects (id, name, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, p.ID, p.Name, p.Description, p.Status, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert project: %w", err)
	}
	return nil
}

// GetProject returns one project by id.
func (s *ProjectsStore) GetProject(ctx context.Context, id uuid.UUID) (*models.Project, error) {
	var p models.Project
	err := s.q.QueryRowContext(ctx, `
		SELECT id, name, description, status, created_at, updated_at
		FROM projects WHERE id = $1
	`, id).Scan(&p.ID, &p.Name, &p.Description, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("project %s not found", id)
		}
		return nil, fmt.Errorf("select project: %w", err)
	}
	return &p, nil
}

// ListProjects returns every project, newest first.
func (s *ProjectsStore) ListProjects(ctx context.Context) ([]models.Project, error) {
	rows, err := s.q.QueryContext(ctx, `
		SELECT id, name, description, status, created_at, updated_at
		FROM projects ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var out []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating projects: %w", err)
	}
	return out, nil
}

// UpdateProject replaces the row for p.ID.
func (s *ProjectsStore) UpdateProject(ctx context.Context, p *models.Project) error {
	_, err := s.q.ExecContext(ctx, `
		UPDATE projects
		SET name = $1, description = $2, status = $3, updated_at = $4
		WHERE id = $5
	`, p.Name, p.Description, p.Status, p.UpdatedAt, p.ID)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	return nil
}

// UpdateProjectStatus sets only the status column.
func (s *ProjectsStore) UpdateProjectStatus(ctx context.Context, id uuid.UUID, status string) error {
	tag, err := s.q.ExecContext(ctx, `
		UPDATE projects SET status = $1, updated_at = NOW() WHERE id = $2
	`, status, id)
	if err != nil {
		return fmt.Errorf("update project status: %w", err)
	}
	rows, err := tag.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("project %s not found", id)
	}
	return nil
}

// DeleteProject removes a row by id.
func (s *ProjectsStore) DeleteProject(ctx context.Context, id uuid.UUID) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}

// DeleteProjectCascade removes the project and all rows that reference it
// (contexts, tasks, agents). It must be called inside a transaction the
// caller owns, so a failure rolls back atomically. The supplied Querier is
// expected to be a *sql.Tx.
func (s *ProjectsStore) DeleteProjectCascade(ctx context.Context, tx Querier, id uuid.UUID) error {
	// ON DELETE CASCADE on tasks.project_id would handle tasks and contexts,
	// but the existing schema uses explicit deletes; we keep the existing
	// behaviour here for back-compat.
	if _, err := tx.ExecContext(ctx, `DELETE FROM contexts WHERE task_id IN (SELECT id FROM tasks WHERE project_id = $1)`, id); err != nil {
		return fmt.Errorf("delete related contexts: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE project_id = $1`, id); err != nil {
		return fmt.Errorf("delete related tasks: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM agents WHERE project_id = $1`, id); err != nil {
		return fmt.Errorf("delete related agents: %w", err)
	}
	tag, err := tx.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	rows, err := tag.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("project %s not found", id)
	}
	return nil
}

// BeginTx starts a transaction on the underlying database/sql connection.
// Callers must Commit or Rollback the returned *sql.Tx.
func (s *ProjectsStore) BeginTx(ctx context.Context) (*sql.Tx, error) {
	tx, ok := s.q.(txBeginner)
	if !ok {
		return nil, errors.New("ProjectsStore: underlying Querier does not support transactions; pass *sql.DB")
	}
	return tx.BeginTx(ctx, nil)
}

// txBeginner is satisfied by *sql.DB and *sql.Tx (BeginTx on Tx is a no-op).
type txBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}
