package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
)

// maxListPageSize bounds the LIMIT in ListTasks so a misconfigured caller
// cannot request millions of rows in one round-trip.
const maxListPageSize = 1000

// PostgresStore implements the Store interface backed by a pgx connection
// pool. Rows are stored in the `a2a_tasks` table created by
// migrations/004_a2a_tasks.sql.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore returns a Store backed by the supplied pool. The caller
// retains ownership of the pool and must close it after the Store is no
// longer in use.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// CreateTask inserts a new task. Returns an error wrapping the underlying
// pgx error when the PK already exists.
func (s *PostgresStore) CreateTask(ctx context.Context, task *models.Task) error {
	msg, err := json.Marshal(task.Message)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	arts, err := json.Marshal(task.Artifacts)
	if err != nil {
		return fmt.Errorf("marshal artifacts: %w", err)
	}
	// Result is optional; treat nil as JSON null.
	var resultBytes []byte
	if task.Result != nil {
		resultBytes, err = json.Marshal(task.Result)
		if err != nil {
			return fmt.Errorf("marshal result: %w", err)
		}
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO a2a_tasks (id, status, message, result, artifacts, created_at, updated_at)
		VALUES (@id, @status, @message::jsonb, NULLIF(@result::jsonb, 'null'::jsonb), @artifacts::jsonb, @created_at, @updated_at)
	`, pgx.NamedArgs{
		"id":         task.ID,
		"status":     string(task.Status),
		"message":    msg,
		"result":     resultBytes,
		"artifacts":  arts,
		"created_at": task.CreatedAt,
		"updated_at": task.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

// GetTask returns a task by ID. When the row does not exist the returned
// error wraps ErrTaskNotFound so callers can use errors.Is.
func (s *PostgresStore) GetTask(ctx context.Context, taskID string) (*models.Task, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, status, message, result, artifacts, created_at, updated_at, completed_at
		FROM a2a_tasks
		WHERE id = @id
	`, pgx.NamedArgs{"id": taskID})

	t, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrTaskNotFound, taskID)
		}
		return nil, err
	}
	return t, nil
}

// UpdateTask overwrites the row for task.ID. Returns ErrTaskNotFound (wrapped)
// when the task has been removed between read and write.
func (s *PostgresStore) UpdateTask(ctx context.Context, task *models.Task) error {
	msg, err := json.Marshal(task.Message)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	arts, err := json.Marshal(task.Artifacts)
	if err != nil {
		return fmt.Errorf("marshal artifacts: %w", err)
	}
	var resultBytes []byte
	if task.Result != nil {
		resultBytes, err = json.Marshal(task.Result)
		if err != nil {
			return fmt.Errorf("marshal result: %w", err)
		}
	}

	tag, err := s.pool.Exec(ctx, `
		UPDATE a2a_tasks
		SET status = @status,
		    message = @message::jsonb,
		    result = NULLIF(@result::jsonb, 'null'::jsonb),
		    artifacts = @artifacts::jsonb,
		    updated_at = @updated_at,
		    completed_at = @completed_at
		WHERE id = @id
	`, pgx.NamedArgs{
		"id":           task.ID,
		"status":       string(task.Status),
		"message":      msg,
		"result":       resultBytes,
		"artifacts":    arts,
		"updated_at":   task.UpdatedAt,
		"completed_at": task.CompletedAt,
	})
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %s", ErrTaskNotFound, task.ID)
	}
	return nil
}

// DeleteTask removes a task by ID. Returns ErrTaskNotFound (wrapped) when
// the row does not exist.
func (s *PostgresStore) DeleteTask(ctx context.Context, taskID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM a2a_tasks WHERE id = @id`,
		pgx.NamedArgs{"id": taskID})
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %s", ErrTaskNotFound, taskID)
	}
	return nil
}

// ListTasks returns tasks matching filter. Status and pagination bounds are
// validated before the query is built; LIMIT is capped at maxListPageSize so
// a misconfigured caller cannot request an unbounded scan.
func (s *PostgresStore) ListTasks(ctx context.Context, filter *Filter) ([]models.Task, error) {
	args := pgx.NamedArgs{}
	where := ""
	if filter != nil && filter.Status != "" {
		where = "WHERE status = @status"
		args["status"] = filter.Status
	}

	limit := maxListPageSize
	offset := 0
	if filter != nil {
		if filter.Limit > 0 {
			limit = filter.Limit
			if limit > maxListPageSize {
				limit = maxListPageSize
			}
		}
		if filter.Offset > 0 {
			offset = filter.Offset
		}
	}
	args["limit"] = limit
	args["offset"] = offset

	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, status, message, result, artifacts, created_at, updated_at, completed_at
		FROM a2a_tasks
		%s
		ORDER BY updated_at DESC
		LIMIT @limit OFFSET @offset
	`, where), args)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var out []models.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tasks: %w", err)
	}
	return out, nil
}

// rowScanner is the minimal surface shared by pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanTask reads one row into a Task. result and artifacts are JSONB; nil
// scans are mapped to (nil, nil) so completed_at / result stay optional.
func scanTask(r rowScanner) (*models.Task, error) {
	var (
		task        models.Task
		status      string
		message     []byte
		resultBytes []byte
		arts        []byte
		completedAt *time.Time
	)
	if err := r.Scan(
		&task.ID,
		&status,
		&message,
		&resultBytes,
		&arts,
		&task.CreatedAt,
		&task.UpdatedAt,
		&completedAt,
	); err != nil {
		return nil, err
	}
	task.Status = models.TaskStatus(status)
	task.CompletedAt = completedAt

	if err := json.Unmarshal(message, &task.Message); err != nil {
		return nil, fmt.Errorf("unmarshal message: %w", err)
	}
	if len(arts) > 0 {
		if err := json.Unmarshal(arts, &task.Artifacts); err != nil {
			return nil, fmt.Errorf("unmarshal artifacts: %w", err)
		}
	}
	if len(resultBytes) > 0 {
		var res models.Result
		if err := json.Unmarshal(resultBytes, &res); err != nil {
			return nil, fmt.Errorf("unmarshal result: %w", err)
		}
		task.Result = &res
	}
	return &task, nil
}
