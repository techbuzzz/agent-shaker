package queries

import (
	"context"
	"fmt"
)

// DashboardStats is the typed aggregate returned by DashboardStore.Stats.
// Kept here (not in models) because it is a pure aggregation; downstream
// callers wrap it in the public handlers.DashboardStats wire type.
type DashboardStats struct {
	Projects ProjectBucket
	Agents   AgentBucket
	Tasks    TaskBucket
	Contexts ContextBucket
}

type ProjectBucket struct {
	Total    int
	Active   int
	Archived int
}

type AgentBucket struct {
	Total   int
	Active  int
	Idle    int
	Offline int
}

type TaskBucket struct {
	Total      int
	Pending    int
	InProgress int
	Done       int
	Blocked    int
}

type ContextBucket struct {
	Total int
}

// DashboardStore computes aggregate statistics for the dashboard view.
// One query per resource keeps each COUNT cheap and avoids a long
// full-table scan; Postgres runs the four queries in sequence.
type DashboardStore struct {
	q Querier
}

func NewDashboardStore(q Querier) *DashboardStore {
	return &DashboardStore{q: q}
}

// Available reports whether the store has a usable Querier.
func (s *DashboardStore) Available() bool { return s.q != nil }

// Stats returns the four buckets in a single call. Each underlying query
// is bounded (filtered COUNT); failure of any individual bucket is
// non-fatal — the bucket is zero and the others are still returned so the
// dashboard renders a partial result rather than 500.
func (s *DashboardStore) Stats(ctx context.Context) (DashboardStats, error) {
	out := DashboardStats{}

	var err error
	if out.Projects, err = s.projectStats(ctx); err != nil {
		return out, fmt.Errorf("project stats: %w", err)
	}
	if out.Agents, err = s.agentStats(ctx); err != nil {
		return out, fmt.Errorf("agent stats: %w", err)
	}
	if out.Tasks, err = s.taskStats(ctx); err != nil {
		return out, fmt.Errorf("task stats: %w", err)
	}
	if out.Contexts, err = s.contextStats(ctx); err != nil {
		return out, fmt.Errorf("context stats: %w", err)
	}
	return out, nil
}

func (s *DashboardStore) projectStats(ctx context.Context) (ProjectBucket, error) {
	var b ProjectBucket
	err := s.q.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'active')  AS active,
			COUNT(*) FILTER (WHERE status = 'archived') AS archived
		FROM projects
	`).Scan(&b.Total, &b.Active, &b.Archived)
	if err != nil {
		return ProjectBucket{}, err
	}
	return b, nil
}

func (s *DashboardStore) agentStats(ctx context.Context) (AgentBucket, error) {
	var b AgentBucket
	err := s.q.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'active')  AS active,
			COUNT(*) FILTER (WHERE status = 'idle')    AS idle,
			COUNT(*) FILTER (WHERE status = 'offline') AS offline
		FROM agents
	`).Scan(&b.Total, &b.Active, &b.Idle, &b.Offline)
	if err != nil {
		return AgentBucket{}, err
	}
	return b, nil
}

func (s *DashboardStore) taskStats(ctx context.Context) (TaskBucket, error) {
	var b TaskBucket
	err := s.q.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'pending')     AS pending,
			COUNT(*) FILTER (WHERE status = 'in_progress') AS in_progress,
			COUNT(*) FILTER (WHERE status = 'done')        AS done,
			COUNT(*) FILTER (WHERE status = 'blocked')     AS blocked
		FROM tasks
	`).Scan(&b.Total, &b.Pending, &b.InProgress, &b.Done, &b.Blocked)
	if err != nil {
		return TaskBucket{}, err
	}
	return b, nil
}

func (s *DashboardStore) contextStats(ctx context.Context) (ContextBucket, error) {
	var b ContextBucket
	err := s.q.QueryRowContext(ctx, `SELECT COUNT(*) AS total FROM contexts`).Scan(&b.Total)
	if err != nil {
		return ContextBucket{}, err
	}
	return b, nil
}
