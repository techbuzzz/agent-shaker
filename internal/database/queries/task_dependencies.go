package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// TaskDependenciesStore is the typed query layer for the `task_dependencies`
// table — the edges of the task DAG.
type TaskDependenciesStore struct {
	q Querier
}

func NewTaskDependenciesStore(q Querier) *TaskDependenciesStore {
	return &TaskDependenciesStore{q: q}
}

// Available reports whether the store has a usable Querier.
func (s *TaskDependenciesStore) Available() bool { return s.q != nil }

// ErrCyclicDependency is returned when adding an edge would close a cycle.
//
// A cycle means two tasks each wait for the other: neither can ever start, and
// the resolver that would promote them to ready spins forever waiting for a
// predecessor that is itself waiting. It is reported as a sentinel so the
// handler answers 409 with a message an agent can act on.
var ErrCyclicDependency = errors.New("cyclic dependency detected")

// AddDependency records that taskID waits for dependsOn.
//
// The cycle check runs inside the same statement as the insert so two agents
// declaring A→B and B→A at the same moment cannot both pass a check-then-insert
// pair. PostgreSQL evaluates the data-modifying CTE against the same snapshot,
// which is what makes the guard atomic here.
func (s *TaskDependenciesStore) AddDependency(ctx context.Context, taskID, dependsOn uuid.UUID) error {
	return s.AddDependencyTx(ctx, nil, taskID, dependsOn)
}

// AddDependencyTx is AddDependency inside a caller's transaction.
//
// The handoff path needs this: completing a task, creating its successor and
// recording the edge between them are one fact, and an edge written outside the
// transaction can survive a rollback that discarded the task it pointed at.
func (s *TaskDependenciesStore) AddDependencyTx(ctx context.Context, q Querier, taskID, dependsOn uuid.UUID) error {
	if q == nil {
		q = s.q
	}
	if taskID == dependsOn {
		return ErrCyclicDependency
	}

	// Walk forward from dependsOn: if we can reach taskID, adding this edge
	// closes a loop.
	var reachable bool
	err := q.QueryRowContext(ctx, `
		WITH RECURSIVE dep_chain AS (
			SELECT depends_on_task_id FROM task_dependencies WHERE task_id = $1
			UNION
			SELECT td.depends_on_task_id
			FROM task_dependencies td
			JOIN dep_chain dc ON td.task_id = dc.depends_on_task_id
		)
		SELECT EXISTS (SELECT 1 FROM dep_chain WHERE depends_on_task_id = $2)
	`, dependsOn, taskID).Scan(&reachable)
	if err != nil {
		return fmt.Errorf("check cyclic dependency: %w", err)
	}
	if reachable {
		return ErrCyclicDependency
	}

	// ON CONFLICT DO NOTHING: two agents declaring the same dependency is a
	// race, not an error, and the second one already has what it asked for.
	_, err = q.ExecContext(ctx, `
		INSERT INTO task_dependencies (task_id, depends_on_task_id)
		VALUES ($1, $2)
		ON CONFLICT (task_id, depends_on_task_id) DO NOTHING
	`, taskID, dependsOn)
	if err != nil {
		return fmt.Errorf("insert task dependency: %w", err)
	}
	return nil
}

// ListDependencies returns a task's edges, both directions: what it waits for
// and what waits for it.
//
// Both halves are returned because the agent asking is usually trying to decide
// what to work on next, and "you are blocked by X" without "you unblock Y" is
// half the answer.
func (s *TaskDependenciesStore) ListDependencies(ctx context.Context, taskID uuid.UUID) (dependsOn, dependsFor []models.TaskDependency, err error) {
	rows, err := s.q.QueryContext(ctx, `
		SELECT id, task_id, depends_on_task_id, created_at
		FROM task_dependencies WHERE task_id = $1 ORDER BY created_at
	`, taskID)
	if err != nil {
		return nil, nil, fmt.Errorf("list dependencies: %w", err)
	}
	defer rows.Close()
	dependsOn, err = scanDependencyRows(rows)
	if err != nil {
		return nil, nil, err
	}

	rows2, err := s.q.QueryContext(ctx, `
		SELECT id, task_id, depends_on_task_id, created_at
		FROM task_dependencies WHERE depends_on_task_id = $1 ORDER BY created_at
	`, taskID)
	if err != nil {
		return nil, nil, fmt.Errorf("list dependents: %w", err)
	}
	defer rows2.Close()
	dependsFor, err = scanDependencyRows(rows2)
	if err != nil {
		return nil, nil, err
	}
	return dependsOn, dependsFor, nil
}

func scanDependencyRows(rows *sql.Rows) ([]models.TaskDependency, error) {
	out := []models.TaskDependency{}
	for rows.Next() {
		var d models.TaskDependency
		var createdAt sql.NullTime
		if err := rows.Scan(&d.ID, &d.TaskID, &d.DependsOnTaskID, &createdAt); err != nil {
			return nil, fmt.Errorf("scan task dependency: %w", err)
		}
		d.CreatedAt = createdAt.Time
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating task dependencies: %w", err)
	}
	return out, nil
}

// CountUnfinishedDependencies returns how many of a task's predecessors are not
// yet done.
//
// This is the readiness predicate. It deliberately counts rather than exists:
// the task becomes ready when this reaches zero, and a task with two unfinished
// predecessors must not be promoted when only one finishes.
func (s *TaskDependenciesStore) CountUnfinishedDependencies(ctx context.Context, q Querier, taskID uuid.UUID) (int, error) {
	if q == nil {
		q = s.q
	}
	var n int
	err := q.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM task_dependencies td
		JOIN tasks t ON t.id = td.depends_on_task_id
		WHERE td.task_id = $1 AND t.status <> 'done'
	`, taskID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count unfinished dependencies: %w", err)
	}
	return n, nil
}

// DependentTaskIDs returns the tasks that were waiting on the completed task.
// Called on every completion, so it is the query the reverse index exists for.
func (s *TaskDependenciesStore) DependentTaskIDs(ctx context.Context, q Querier, completedTaskID uuid.UUID) ([]uuid.UUID, error) {
	if q == nil {
		q = s.q
	}
	rows, err := q.QueryContext(ctx, `
		SELECT DISTINCT task_id FROM task_dependencies WHERE depends_on_task_id = $1
	`, completedTaskID)
	if err != nil {
		return nil, fmt.Errorf("list dependent task ids: %w", err)
	}
	defer rows.Close()

	out := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan dependent task id: %w", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating dependent task ids: %w", err)
	}
	return out, nil
}

// MarkTaskReady promotes a blocked task to ready.
//
// The `AND status = 'blocked'` guard keeps this idempotent under concurrency: two
// completions resolving the same dependent both run this, one moves it, the
// other affects zero rows and reports false instead of double-broadcasting a
// task_ready event.
func (s *TaskDependenciesStore) MarkTaskReady(ctx context.Context, q Querier, taskID uuid.UUID) (promoted bool, err error) {
	if q == nil {
		q = s.q
	}
	tag, err := q.ExecContext(ctx, `
		UPDATE tasks SET status = 'ready', updated_at = NOW()
		WHERE id = $1 AND status = 'blocked'
	`, taskID)
	if err != nil {
		return false, fmt.Errorf("mark task ready: %w", err)
	}
	rows, err := tag.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	return rows > 0, nil
}

// MarkTaskBlocked puts a task back into blocked when a new dependency is added
// to something it does not yet satisfy.
func (s *TaskDependenciesStore) MarkTaskBlocked(ctx context.Context, q Querier, taskID uuid.UUID) error {
	if q == nil {
		q = s.q
	}
	_, err := q.ExecContext(ctx, `
		UPDATE tasks SET status = 'blocked', updated_at = NOW()
		WHERE id = $1 AND status IN ('pending', 'ready')
	`, taskID)
	if err != nil {
		return fmt.Errorf("mark task blocked: %w", err)
	}
	return nil
}

// BeginTx starts a transaction; see ProjectsStore.BeginTx for the same shape.
func (s *TaskDependenciesStore) BeginTx(ctx context.Context) (*sql.Tx, error) {
	tx, ok := s.q.(txBeginner)
	if !ok {
		return nil, errors.New("TaskDependenciesStore: underlying Querier does not support transactions; pass *sql.DB")
	}
	return tx.BeginTx(ctx, nil)
}
