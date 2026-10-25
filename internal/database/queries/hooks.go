package queries

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// HooksStore is the typed query layer for the hooks and hook_runs tables.
type HooksStore struct {
	q Querier
}

// NewHooksStore returns a store over the given Querier.
func NewHooksStore(q Querier) *HooksStore {
	return &HooksStore{q: q}
}

// Available reports whether the store has a usable Querier.
func (s *HooksStore) Available() bool { return s.q != nil }

const hookColumns = `id, project_id, event, matcher, command, env, stdin, post,
	timeout_seconds, enabled, created_by, created_at, updated_at`

// CreateHook inserts a hook.
func (s *HooksStore) CreateHook(ctx context.Context, h *models.Hook) error {
	matcher, err := marshalNullableJSON(h.Matcher)
	if err != nil {
		return fmt.Errorf("encode hook matcher: %w", err)
	}
	post, err := marshalNullableJSON(h.Post)
	if err != nil {
		return fmt.Errorf("encode hook post: %w", err)
	}
	env, err := json.Marshal(h.Env)
	if err != nil {
		return fmt.Errorf("encode hook env: %w", err)
	}
	stdin, err := json.Marshal(h.Stdin)
	if err != nil {
		return fmt.Errorf("encode hook stdin: %w", err)
	}

	_, err = s.q.ExecContext(ctx, `
		INSERT INTO hooks (id, project_id, event, matcher, command, env, stdin, post,
			timeout_seconds, enabled, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`,
		h.ID, h.ProjectID, h.Event, matcher, pq.Array(h.Command), env, stdin, post,
		h.TimeoutSeconds, h.Enabled, h.CreatedBy, h.CreatedAt, h.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert hook: %w", err)
	}
	return nil
}

// UpdateHook writes the mutable fields of a hook.
func (s *HooksStore) UpdateHook(ctx context.Context, h *models.Hook) error {
	matcher, err := marshalNullableJSON(h.Matcher)
	if err != nil {
		return fmt.Errorf("encode hook matcher: %w", err)
	}
	post, err := marshalNullableJSON(h.Post)
	if err != nil {
		return fmt.Errorf("encode hook post: %w", err)
	}
	env, err := json.Marshal(h.Env)
	if err != nil {
		return fmt.Errorf("encode hook env: %w", err)
	}
	stdin, err := json.Marshal(h.Stdin)
	if err != nil {
		return fmt.Errorf("encode hook stdin: %w", err)
	}

	_, err = s.q.ExecContext(ctx, `
		UPDATE hooks
		SET event = $2, matcher = $3, command = $4, env = $5, stdin = $6, post = $7,
			timeout_seconds = $8, enabled = $9, updated_at = $10
		WHERE id = $1
	`,
		h.ID, h.Event, matcher, pq.Array(h.Command), env, stdin, post,
		h.TimeoutSeconds, h.Enabled, h.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update hook: %w", err)
	}
	return nil
}

// DeleteHook removes a hook. Its runs survive: hook_runs.hook_id is set to
// NULL on delete, because "this hook ran twice and was then deleted" is
// exactly the row an operator needs.
func (s *HooksStore) DeleteHook(ctx context.Context, id uuid.UUID) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM hooks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete hook: %w", err)
	}
	return nil
}

// GetHook returns one hook by id.
func (s *HooksStore) GetHook(ctx context.Context, id uuid.UUID) (*models.Hook, error) {
	row := s.q.QueryRowContext(ctx,
		`SELECT `+hookColumns+` FROM hooks WHERE id = $1`, id)

	h, err := scanHook(row)
	if err != nil {
		return nil, err
	}
	return h, nil
}

// ListHooks returns every hook for a project, newest first.
func (s *HooksStore) ListHooks(ctx context.Context, projectID uuid.UUID) ([]models.Hook, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+hookColumns+` FROM hooks WHERE project_id = $1 ORDER BY created_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list hooks: %w", err)
	}
	defer rows.Close()

	out := make([]models.Hook, 0)
	for rows.Next() {
		h, err := scanHook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating hooks: %w", err)
	}
	return out, nil
}

// ListEnabledHooks returns the enabled hooks bound to one event type in a
// project. This is the dispatcher's query, and the partial index on
// (project_id, event) WHERE enabled is built for exactly it.
func (s *HooksStore) ListEnabledHooks(ctx context.Context, projectID uuid.UUID, eventType string) ([]models.Hook, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+hookColumns+` FROM hooks
		 WHERE project_id = $1 AND event = $2 AND enabled
		 ORDER BY created_at`,
		projectID, eventType,
	)
	if err != nil {
		return nil, fmt.Errorf("list enabled hooks: %w", err)
	}
	defer rows.Close()

	out := make([]models.Hook, 0)
	for rows.Next() {
		h, err := scanHook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating enabled hooks: %w", err)
	}
	return out, nil
}

// RecordRun writes one hook run.
func (s *HooksStore) RecordRun(ctx context.Context, run *models.HookRun) error {
	_, err := s.q.ExecContext(ctx, `
		INSERT INTO hook_runs (id, hook_id, delivery_id, project_id, event,
			entity_type, entity_id, status, exit_code, stdout, stderr,
			duration_ms, started_at, finished_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`,
		run.ID, run.HookID, run.DeliveryID, run.ProjectID, run.Event,
		run.EntityType, run.EntityID, run.Status, run.ExitCode, run.Stdout,
		run.Stderr, run.DurationMS, run.StartedAt, run.FinishedAt,
	)
	if err != nil {
		return fmt.Errorf("insert hook run: %w", err)
	}
	return nil
}

// ListRuns returns the most recent runs for a project.
func (s *HooksStore) ListRuns(ctx context.Context, projectID uuid.UUID, limit int) ([]models.HookRun, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.q.QueryContext(ctx, `
		SELECT id, hook_id, delivery_id, project_id, event, entity_type,
			entity_id, status, exit_code, stdout, stderr, duration_ms,
			started_at, finished_at
		FROM hook_runs
		WHERE project_id = $1
		ORDER BY started_at DESC
		LIMIT $2
	`, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("list hook runs: %w", err)
	}
	defer rows.Close()

	out := make([]models.HookRun, 0)
	for rows.Next() {
		var r models.HookRun
		if err := rows.Scan(
			&r.ID, &r.HookID, &r.DeliveryID, &r.ProjectID, &r.Event,
			&r.EntityType, &r.EntityID, &r.Status, &r.ExitCode, &r.Stdout,
			&r.Stderr, &r.DurationMS, &r.StartedAt, &r.FinishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan hook run: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating hook runs: %w", err)
	}
	return out, nil
}

// SetHooksEnabled turns the hook layer on or off for one project.
func (s *HooksStore) SetHooksEnabled(ctx context.Context, projectID uuid.UUID, enabled bool) error {
	_, err := s.q.ExecContext(ctx,
		`UPDATE projects SET hooks_enabled = $2 WHERE id = $1`, projectID, enabled)
	if err != nil {
		return fmt.Errorf("set hooks_enabled: %w", err)
	}
	return nil
}

// HooksEnabled reports whether the hook layer is enabled for one project.
func (s *HooksStore) HooksEnabled(ctx context.Context, projectID uuid.UUID) (bool, error) {
	var enabled bool
	err := s.q.QueryRowContext(ctx,
		`SELECT hooks_enabled FROM projects WHERE id = $1`, projectID).Scan(&enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read hooks_enabled: %w", err)
	}
	return enabled, nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanHook(row scanner) (*models.Hook, error) {
	var (
		h       models.Hook
		matcher []byte
		env     []byte
		stdin   []byte
		post    []byte
		command pq.StringArray
	)
	if err := row.Scan(
		&h.ID, &h.ProjectID, &h.Event, &matcher, &command, &env, &stdin, &post,
		&h.TimeoutSeconds, &h.Enabled, &h.CreatedBy, &h.CreatedAt, &h.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan hook: %w", err)
	}

	h.Command = []string(command)
	h.Env = models.HookEnv{}
	if len(env) > 0 {
		if err := json.Unmarshal(env, &h.Env); err != nil {
			return nil, fmt.Errorf("decode hook env: %w", err)
		}
	}
	if err := json.Unmarshal(stdin, &h.Stdin); err != nil {
		return nil, fmt.Errorf("decode hook stdin: %w", err)
	}
	if err := unmarshalNullableJSON(matcher, &h.Matcher); err != nil {
		return nil, fmt.Errorf("decode hook matcher: %w", err)
	}
	if err := unmarshalNullableJSON(post, &h.Post); err != nil {
		return nil, fmt.Errorf("decode hook post: %w", err)
	}
	return &h, nil
}

// marshalNullableJSON encodes an optional value. A nil pointer becomes SQL
// NULL rather than the JSON literal "null", so the column can carry "no
// matcher" and "an empty matcher" as different things.
//
// Generic over the pointer type rather than taking `any`: a typed nil pointer
// stored in an interface is not equal to nil, and the check that catches it
// would have to be a type switch over every optional column in the schema.
func marshalNullableJSON[T any](v *T) (any, error) {
	if v == nil {
		return nil, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func unmarshalNullableJSON(raw []byte, target any) error {
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return err
	}
	return nil
}
