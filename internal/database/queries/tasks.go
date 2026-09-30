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

// TasksStore is the typed query layer for the project `tasks` table.
type TasksStore struct {
	q Querier
}

func NewTasksStore(q Querier) *TasksStore {
	return &TasksStore{q: q}
}

// Available reports whether the store has a usable Querier.
func (s *TasksStore) Available() bool { return s.q != nil }

func (s *TasksStore) CreateTask(ctx context.Context, t *models.Task) error {
	_, err := s.q.ExecContext(ctx, `
		INSERT INTO tasks (id, project_id, title, description, status, priority, created_by, assigned_to, output, milestone_id, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, t.ID, t.ProjectID, t.Title, t.Description, string(t.Status), t.Priority, t.CreatedBy, t.AssignedTo, t.Output, t.MilestoneID, pq.Array(t.Tags), t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

func (s *TasksStore) ListTasks(ctx context.Context, projectID *uuid.UUID, status string) ([]models.Task, error) {
	args := []any{}
	q := `SELECT id, project_id, title, description, status, priority, created_by, assigned_to, output, milestone_id, tags, created_at, updated_at FROM tasks WHERE 1=1`
	if projectID != nil {
		q += " AND project_id = $1"
		args = append(args, *projectID)
	}
	if status != "" {
		q += fmt.Sprintf(" AND status = $%d", len(args)+1)
		args = append(args, status)
	}
	q += " ORDER BY created_at DESC"

	rows, err := s.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var out []models.Task
	for rows.Next() {
		var t models.Task
		var statusStr string
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &statusStr, &t.Priority, &t.CreatedBy, &t.AssignedTo, &t.Output, &t.MilestoneID, pq.Array(&t.Tags), &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		t.Status = models.TaskStatus(statusStr)
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tasks: %w", err)
	}
	return out, nil
}

func (s *TasksStore) GetTask(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	var t models.Task
	var statusStr string
	err := s.q.QueryRowContext(ctx, `
		SELECT id, project_id, title, description, status, priority, created_by, assigned_to, output, milestone_id, tags, created_at, updated_at
		FROM tasks WHERE id = $1
	`, id).Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &statusStr, &t.Priority, &t.CreatedBy, &t.AssignedTo, &t.Output, &t.MilestoneID, pq.Array(&t.Tags), &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("task %s not found", id)
		}
		return nil, fmt.Errorf("select task: %w", err)
	}
	t.Status = models.TaskStatus(statusStr)
	return &t, nil
}

func (s *TasksStore) UpdateTask(ctx context.Context, t *models.Task) error {
	_, err := s.q.ExecContext(ctx, `
		UPDATE tasks
		SET title = $1, description = $2, status = $3, priority = $4, assigned_to = $5, output = $6, updated_at = $7
		WHERE id = $8
	`, t.Title, t.Description, string(t.Status), t.Priority, t.AssignedTo, t.Output, t.UpdatedAt, t.ID)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

func (s *TasksStore) UpdateTaskStatus(ctx context.Context, id uuid.UUID, status string) error {
	tag, err := s.q.ExecContext(ctx, `
		UPDATE tasks SET status = $1, updated_at = NOW() WHERE id = $2
	`, status, id)
	if err != nil {
		return fmt.Errorf("update task status: %w", err)
	}
	rows, err := tag.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("task %s not found", id)
	}
	return nil
}

func (s *TasksStore) ReassignTask(ctx context.Context, id uuid.UUID, assignedTo uuid.UUID) error {
	tag, err := s.q.ExecContext(ctx, `
		UPDATE tasks SET assigned_to = $1, updated_at = NOW() WHERE id = $2
	`, assignedTo, id)
	if err != nil {
		return fmt.Errorf("reassign task: %w", err)
	}
	rows, err := tag.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("task %s not found", id)
	}
	return nil
}

func (s *TasksStore) DeleteTask(ctx context.Context, id uuid.UUID) error {
	tag, err := s.q.ExecContext(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	rows, err := tag.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("task %s not found", id)
	}
	return nil
}

// ClaimTask atomically assigns the task to agentID only if the task is
// currently unassigned OR already assigned to agentID. Returns
// (claimed=true) on success, (claimed=false) when another agent owns the
// task. Used by the MCP `claim_task` tool to avoid two agents stealing the
// same work item.
//
// On success the task is also moved to `in_progress` so the dashboard
// reflects the claim immediately.
func (s *TasksStore) ClaimTask(ctx context.Context, taskID, agentID uuid.UUID) (claimed bool, err error) {
	tag, err := s.q.ExecContext(ctx, `
		UPDATE tasks
		SET assigned_to = $1,
		    status = CASE WHEN status = 'pending' THEN 'in_progress' ELSE status END,
		    updated_at = NOW()
		WHERE id = $2 AND (assigned_to IS NULL OR assigned_to = $1)
	`, agentID, taskID)
	if err != nil {
		return false, fmt.Errorf("claim task: %w", err)
	}
	rows, err := tag.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return false, nil // owned by someone else
	}
	return true, nil
}

// CountMilestoneOpenTasks returns the number of tasks linked to a
// milestone that are not yet in a terminal status. Used by the
// "milestone close" guard.
func (s *TasksStore) CountMilestoneOpenTasks(ctx context.Context, milestoneID uuid.UUID) (int, error) {
	var n int
	err := s.q.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM tasks
		WHERE milestone_id = $1
		  AND status NOT IN ('done', 'cancelled')
	`, milestoneID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count milestone open tasks: %w", err)
	}
	return n, nil
}

// CountMilestoneTasks returns total tasks linked to a milestone (for the
// "done / total" progress meter on MilestoneCard).
func (s *TasksStore) CountMilestoneTasks(ctx context.Context, milestoneID uuid.UUID) (int, error) {
	var n int
	err := s.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks WHERE milestone_id = $1`, milestoneID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count milestone tasks: %w", err)
	}
	return n, nil
}
