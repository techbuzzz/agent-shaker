package queries

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// HeartbeatsStore is the typed query layer for the `agent_heartbeats` table.
//
// It is separate from AgentsStore because it holds history, not current state:
// the agent's live lease lives on the agents row and is written by
// AgentsStore.RecordHeartbeat. Keeping the two apart is what lets the reaper's
// hot query touch a small, current-state table while history grows underneath it.
type HeartbeatsStore struct {
	q Querier
}

func NewHeartbeatsStore(q Querier) *HeartbeatsStore {
	return &HeartbeatsStore{q: q}
}

// Available reports whether the store has a usable Querier.
func (s *HeartbeatsStore) Available() bool { return s.q != nil }

// RecordHeartbeat appends one row to the heartbeat history.
func (s *HeartbeatsStore) RecordHeartbeat(ctx context.Context, q Querier, hb *models.AgentHeartbeat) error {
	if q == nil {
		q = s.q
	}
	_, err := q.ExecContext(ctx, `
		INSERT INTO agent_heartbeats (agent_id, heartbeat_time, status, metadata, project_id, task_id, progress_pct, eta)
		VALUES ($1, NOW(), $2, $3, $4, $5, $6, $7)
	`, hb.AgentID, hb.Status, hb.Metadata, hb.ProjectID, hb.TaskID, hb.ProgressPct, hb.ETA)
	if err != nil {
		return fmt.Errorf("insert heartbeat: %w", err)
	}
	return nil
}

// ListHeartbeats returns an agent's recent heartbeats, newest first.
func (s *HeartbeatsStore) ListHeartbeats(ctx context.Context, agentID uuid.UUID, limit int) ([]models.AgentHeartbeat, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := s.q.QueryContext(ctx, `
		SELECT id, agent_id, heartbeat_time, status, metadata, project_id, task_id, progress_pct, eta
		FROM agent_heartbeats
		WHERE agent_id = $1
		ORDER BY heartbeat_time DESC
		LIMIT $2
	`, agentID, limit)
	if err != nil {
		return nil, fmt.Errorf("list heartbeats: %w", err)
	}
	defer rows.Close()

	out := []models.AgentHeartbeat{}
	for rows.Next() {
		var hb models.AgentHeartbeat
		var (
			meta        []byte
			heartbeatAt sql.NullTime
			projectID   uuid.NullUUID
			taskID      uuid.NullUUID
			progress    sql.NullInt64
			eta         sql.NullTime
		)
		if err := rows.Scan(
			&hb.ID, &hb.AgentID, &heartbeatAt, &hb.Status, &meta, &projectID, &taskID, &progress, &eta,
		); err != nil {
			return nil, fmt.Errorf("scan heartbeat: %w", err)
		}
		// The existing store shape carries Metadata as a decoded map because
		// the standup writer always built one. A heartbeat written by an agent
		// is arbitrary JSONB, so decode defensively: a row whose metadata is
		// not an object must still be readable rather than failing the scan.
		if len(meta) > 0 {
			var decoded map[string]interface{}
			if err := json.Unmarshal(meta, &decoded); err == nil {
				hb.Metadata = decoded
			}
		}
		if heartbeatAt.Valid {
			hb.HeartbeatTime = heartbeatAt.Time
		}
		if projectID.Valid {
			v := projectID.UUID
			hb.ProjectID = &v
		}
		if taskID.Valid {
			v := taskID.UUID
			hb.TaskID = &v
		}
		if progress.Valid {
			v := int(progress.Int64)
			hb.ProgressPct = &v
		}
		if eta.Valid {
			v := eta.Time
			hb.ETA = &v
		}
		out = append(out, hb)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating heartbeats: %w", err)
	}
	return out, nil
}

// RecordLease renews an agent's lease and records what it reported, in one
// statement.
//
// One statement, not two: if the history write failed after the lease was
// renewed, the agent would look alive to the reaper with nothing on record of
// what it is doing, and the dashboard would show a heartbeat that never
// happened.
//
// `status` is the fleet-level state (active/idle/blocked) — see
// models.HeartbeatStatus for why the reported activity is not stored here.
// DeleteOlderThan removes heartbeat rows older than the cutoff and reports how
// many went.
//
// agent_heartbeats is the fastest-growing table in the schema: one row per agent
// every 30 seconds is roughly 2 880 rows per agent per day, and nothing reads
// beyond "recent". This is the retention the design calls for.
func (s *HeartbeatsStore) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	tag, err := s.q.ExecContext(ctx, `DELETE FROM agent_heartbeats WHERE heartbeat_time < $1`, before)
	if err != nil {
		return 0, fmt.Errorf("delete stale heartbeats: %w", err)
	}
	removed, err := tag.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	return removed, nil
}

func (s *AgentsStore) RecordHeartbeat(ctx context.Context, agentID uuid.UUID, status string, taskID *uuid.UUID, progressPct *int, payload []byte, ttl time.Duration) (*models.Agent, error) {
	var (
		projectID uuid.UUID
		name      string
		role      string
		team      string
		lastSeen  time.Time
		createdAt time.Time
	)
	err := s.q.QueryRowContext(ctx, `
		UPDATE agents
		SET last_seen = NOW(),
		    lease_expires_at = NOW() + $2::interval,
		    status = CASE WHEN status = 'offline' THEN 'active' ELSE $1 END,
		    current_task_id = $3,
		    heartbeat_payload = $4
		WHERE id = $5
		RETURNING id, project_id, name, role, team, status, last_seen, created_at
	`, status, intervalSeconds(ttl), taskID, payload, agentID).
		Scan(&agentID, &projectID, &name, &role, &team, &status, &lastSeen, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("agent %s not found", agentID)
		}
		return nil, fmt.Errorf("renew agent lease: %w", err)
	}
	return &models.Agent{
		ID:        agentID,
		ProjectID: projectID,
		Name:      name,
		Role:      models.AgentRole(role),
		Team:      team,
		Status:    status,
		LastSeen:  lastSeen,
		CreatedAt: createdAt,
	}, nil
}

// intervalSeconds renders a duration for a `$n::interval` parameter.
//
// `$2::interval` alone is not enough: lib/pq sends the parameter as text, and an
// integer number of seconds would be read as that many *seconds* only because
// PostgreSQL's interval input accepts a bare number — passing a Go time.Duration
// would send something like "1m0s", which the same cast would reject.
func intervalSeconds(d time.Duration) string {
	return fmt.Sprintf("%d seconds", int(d.Seconds()))
}

// ListStaleAgents returns agents whose lease has expired and who are not
// already offline.
//
// The predicate mirrors idx_agents_lease exactly, so this is an index scan
// rather than a filter over the whole fleet — the reaper runs it every 30
// seconds forever, and a full scan there would be a background load nobody
// chose.
func (s *AgentsStore) ListStaleAgents(ctx context.Context, limit int) ([]models.StaleAgent, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.q.QueryContext(ctx, `
		SELECT id, project_id, name
		FROM agents
		WHERE lease_expires_at IS NOT NULL
		  AND lease_expires_at < NOW()
		  AND status <> 'offline'
		ORDER BY lease_expires_at
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list stale agents: %w", err)
	}
	defer rows.Close()

	out := []models.StaleAgent{}
	for rows.Next() {
		var a models.StaleAgent
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.Name); err != nil {
			return nil, fmt.Errorf("scan stale agent: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating stale agents: %w", err)
	}
	return out, nil
}

// MarkAgentOffline takes an expired lease, if the agent is still stale.
//
// The `lease_expires_at < NOW()` predicate is the optimistic lock: an agent that
// heartbeated between the reaper's SELECT and this UPDATE keeps its lease, and
// the reaper moves on. Without it a slow agent loses a lease it is still
// holding and its task is taken back while it is mid-work.
func (s *AgentsStore) MarkAgentOffline(ctx context.Context, q Querier, agentID uuid.UUID) (changed bool, err error) {
	if q == nil {
		q = s.q
	}
	tag, err := q.ExecContext(ctx, `
		UPDATE agents
		SET status = 'offline',
		    lease_expires_at = NULL,
		    current_task_id = NULL
		WHERE id = $1 AND status <> 'offline' AND lease_expires_at < NOW()
	`, agentID)
	if err != nil {
		return false, fmt.Errorf("mark agent offline: %w", err)
	}
	rows, err := tag.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	return rows > 0, nil
}

// ReleaseAgentTasks returns an agent's in-progress tasks to the backlog and
// returns their ids so the reaper can tell the UI what was taken back.
//
// `ready` is included on purpose: a task promoted to ready while this agent was
// its intended owner has nobody to pick it up otherwise, because ready tasks are
// offered to whoever asks, not to the previous assignee.
func (s *AgentsStore) ReleaseAgentTasks(ctx context.Context, q Querier, agentID uuid.UUID) ([]uuid.UUID, error) {
	if q == nil {
		q = s.q
	}
	rows, err := q.QueryContext(ctx, `
		UPDATE tasks
		SET assigned_to = NULL,
		    status = CASE WHEN status = 'in_progress' THEN 'pending' ELSE status END,
		    updated_at = NOW()
		WHERE assigned_to = $1 AND status IN ('in_progress', 'ready')
		RETURNING id
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("release agent tasks: %w", err)
	}
	defer rows.Close()

	out := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan released task id: %w", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating released tasks: %w", err)
	}
	return out, nil
}
