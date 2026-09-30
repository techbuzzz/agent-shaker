package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// MilestonesStore is the typed query layer for the milestones table.
type MilestonesStore struct {
	q Querier
}

func NewMilestonesStore(q Querier) *MilestonesStore {
	return &MilestonesStore{q: q}
}

// Create inserts a new milestone row.
func (s *MilestonesStore) Create(ctx context.Context, m *models.Milestone) error {
	_, err := s.q.ExecContext(ctx, `
		INSERT INTO milestones (id, project_id, title, description, status, target_date, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, m.ID, m.ProjectID, m.Title, m.Description, string(m.Status), m.TargetDate, m.CreatedBy, m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert milestone: %w", err)
	}
	return nil
}

// ListByProject returns all milestones for a project, newest first.
func (s *MilestonesStore) ListByProject(ctx context.Context, projectID uuid.UUID) ([]models.Milestone, error) {
	rows, err := s.q.QueryContext(ctx, `
		SELECT id, project_id, title, description, status, target_date, created_by, created_at, updated_at
		FROM milestones WHERE project_id = $1 ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list milestones: %w", err)
	}
	defer rows.Close()

	var out []models.Milestone
	for rows.Next() {
		var m models.Milestone
		var statusStr string
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.Title, &m.Description, &statusStr, &m.TargetDate, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan milestone: %w", err)
		}
		m.Status = models.MilestoneStatus(statusStr)
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating milestones: %w", err)
	}
	return out, nil
}

// Get returns one milestone by id.
func (s *MilestonesStore) Get(ctx context.Context, id uuid.UUID) (*models.Milestone, error) {
	var m models.Milestone
	var statusStr string
	err := s.q.QueryRowContext(ctx, `
		SELECT id, project_id, title, description, status, target_date, created_by, created_at, updated_at
		FROM milestones WHERE id = $1
	`, id).Scan(&m.ID, &m.ProjectID, &m.Title, &m.Description, &statusStr, &m.TargetDate, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("milestone %s not found", id)
		}
		return nil, fmt.Errorf("select milestone: %w", err)
	}
	m.Status = models.MilestoneStatus(statusStr)
	return &m, nil
}

// UpdateStatus changes a milestone's status. The "all tasks must be done"
// rule is enforced in the handler layer (with a count query + a row lock)
// rather than the DB, so we keep this method side-effect-light.
func (s *MilestonesStore) UpdateStatus(ctx context.Context, id uuid.UUID, status models.MilestoneStatus, updatedAt any) error {
	tag, err := s.q.ExecContext(ctx, `
		UPDATE milestones SET status = $1, updated_at = $2 WHERE id = $3
	`, string(status), updatedAt, id)
	if err != nil {
		return fmt.Errorf("update milestone status: %w", err)
	}
	rows, err := tag.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("milestone %s not found", id)
	}
	return nil
}

// Update edits mutable fields (description / status). Title is intentionally
// immutable after creation so audit trails stay stable.
func (s *MilestonesStore) Update(ctx context.Context, m *models.Milestone) error {
	_, err := s.q.ExecContext(ctx, `
		UPDATE milestones
		SET description = $1, status = $2, target_date = $3, updated_at = $4
		WHERE id = $5
	`, m.Description, string(m.Status), m.TargetDate, m.UpdatedAt, m.ID)
	if err != nil {
		return fmt.Errorf("update milestone: %w", err)
	}
	return nil
}

// Delete removes a milestone. Tasks linked to it are unlinked via the
// ON DELETE SET NULL foreign key on tasks.milestone_id.
func (s *MilestonesStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.q.ExecContext(ctx, `DELETE FROM milestones WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete milestone: %w", err)
	}
	rows, err := tag.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("milestone %s not found", id)
	}
	return nil
}
