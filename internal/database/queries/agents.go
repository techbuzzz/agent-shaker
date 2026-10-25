package queries

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// AgentsStore is the typed query layer for the agents table.
type AgentsStore struct {
	q Querier
}

// NewAgentsStore returns a store bound to the supplied Querier.
func NewAgentsStore(q Querier) *AgentsStore {
	return &AgentsStore{q: q}
}

// Available reports whether the store has a usable Querier.
func (s *AgentsStore) Available() bool { return s.q != nil }

// CreateAgent inserts a new agent row.
func (s *AgentsStore) CreateAgent(ctx context.Context, a *models.Agent) error {
	// tool_type is NOT NULL with a default, but it is still sent explicitly: a
	// caller that forgot to set it would otherwise silently get 'unknown' and
	// a config generated for the wrong client, which is exactly the class of
	// bug the column exists to prevent.
	toolType := a.ToolType
	if toolType == "" {
		toolType = models.ToolTypeUnknown
	}
	_, err := s.q.ExecContext(ctx, `
		INSERT INTO agents (id, project_id, name, role, team, status, last_seen, created_at, tool_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, a.ID, a.ProjectID, a.Name, a.Role, a.Team, a.Status, a.LastSeen, a.CreatedAt, toolType)
	if err != nil {
		return fmt.Errorf("insert agent: %w", err)
	}
	return nil
}

// agentSelectColumns is the column list every agent read shares.
//
// The coordination columns (lease, current task, last report) are part of it
// because a SELECT that omits them does not fail — the JSON still carries the
// keys with null values, which reads exactly like "this agent has never
// heartbeated". That is how the whole lease surface in the UI ended up dead
// without a single error anywhere.
//
// tool_type belongs here for the same reason: a read that omitted it would
// produce an agent the config generator cannot target, and the endpoint would
// fall back to the generic layout rather than complain.
//
// The lateral join carries the newest heartbeat's progress and ETA so the
// dashboard can answer "how far along, and by when" without an extra request
// per agent.
const agentSelectColumns = `
		a.id, a.project_id, a.name, a.role, a.team, a.status, a.last_seen, a.created_at, a.tool_type,
		a.lease_expires_at, a.current_task_id, a.heartbeat_payload, a.capabilities,
		hb.progress_pct, hb.eta`

// ListAgents returns all agents, or agents filtered by project when projectID
// is non-nil.
func (s *AgentsStore) ListAgents(ctx context.Context, projectID *uuid.UUID) ([]models.Agent, error) {
	const fromAgents = `
		FROM agents a
		LEFT JOIN LATERAL (
			SELECT progress_pct, eta FROM agent_heartbeats
			WHERE agent_id = a.id ORDER BY heartbeat_time DESC LIMIT 1
		) hb ON TRUE`

	var rows *sql.Rows
	var err error
	if projectID != nil {
		rows, err = s.q.QueryContext(ctx, `SELECT `+agentSelectColumns+fromAgents+
			` WHERE a.project_id = $1 ORDER BY a.created_at DESC`, *projectID)
	} else {
		rows, err = s.q.QueryContext(ctx, `SELECT `+agentSelectColumns+fromAgents+
			` ORDER BY a.created_at DESC`)
	}
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	var out []models.Agent
	for rows.Next() {
		a, err := scanAgentRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating agents: %w", err)
	}
	return out, nil
}

// GetAgent returns one agent by id.
func (s *AgentsStore) GetAgent(ctx context.Context, id uuid.UUID) (*models.Agent, error) {
	row := s.q.QueryRowContext(ctx, `SELECT `+agentSelectColumns+`
		FROM agents a
		LEFT JOIN LATERAL (
			SELECT progress_pct, eta FROM agent_heartbeats
			WHERE agent_id = a.id ORDER BY heartbeat_time DESC LIMIT 1
		) hb ON TRUE
		WHERE a.id = $1
	`, id)

	a := &models.Agent{}
	var (
		lease        sql.NullTime
		payload      []byte
		capabilities []byte
		progress     sql.NullInt64
		eta          sql.NullTime
	)
	if err := row.Scan(
		&a.ID, &a.ProjectID, &a.Name, &a.Role, &a.Team, &a.Status, &a.LastSeen, &a.CreatedAt, &a.ToolType,
		&lease, &a.CurrentTaskID, &payload, &capabilities, &progress, &eta,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("agent %s not found", id)
		}
		return nil, fmt.Errorf("select agent: %w", err)
	}
	applyAgentCoordination(a, lease, payload, progress, eta)
	applyAgentCapabilities(a, capabilities)
	return a, nil
}

// scanAgentRow reads one agent row into the model.
func scanAgentRow(row rowScanner) (*models.Agent, error) {
	a := &models.Agent{}
	var (
		lease        sql.NullTime
		payload      []byte
		capabilities []byte
		progress     sql.NullInt64
		eta          sql.NullTime
	)
	if err := row.Scan(
		&a.ID, &a.ProjectID, &a.Name, &a.Role, &a.Team, &a.Status, &a.LastSeen, &a.CreatedAt, &a.ToolType,
		&lease, &a.CurrentTaskID, &payload, &capabilities, &progress, &eta,
	); err != nil {
		return nil, fmt.Errorf("scan agent: %w", err)
	}
	applyAgentCoordination(a, lease, payload, progress, eta)
	applyAgentCapabilities(a, capabilities)
	return a, nil
}

// applyAgentCoordination fills the coordination fields, which are all nullable:
// an agent that has never heartbeated has no lease, no payload and no ETA.
func applyAgentCoordination(a *models.Agent, lease sql.NullTime, payload []byte, progress sql.NullInt64, eta sql.NullTime) {
	if lease.Valid {
		v := lease.Time
		a.LeaseExpiresAt = &v
	}
	if len(payload) > 0 {
		var decoded map[string]interface{}
		// A heartbeat payload is arbitrary JSONB written by an agent; a row whose
		// object cannot be decoded must still read back rather than fail the list.
		if err := json.Unmarshal(payload, &decoded); err == nil {
			a.HeartbeatPayload = decoded
		}
	}
	if progress.Valid {
		v := int(progress.Int64)
		a.LastProgressPct = &v
	}
	if eta.Valid {
		v := eta.Time
		a.LastHeartbeatETA = &v
	}
}

// applyAgentCapabilities decodes the announced capabilities.
//
// A row whose object cannot be decoded must still read back rather than fail
// the list: the column belongs to the client that wrote it, and a client that
// invented a field should not take the agent list down with it.
func applyAgentCapabilities(a *models.Agent, capabilities []byte) {
	if len(capabilities) == 0 {
		return
	}

	var decoded models.AgentCapabilities
	if err := json.Unmarshal(capabilities, &decoded); err != nil {
		return
	}
	a.Capabilities = &decoded
}

// SetCapabilities records what an agent's client says it can do.
//
// The whole announcement replaces the previous one: a client that drops a
// capability has stopped supporting it, and keeping the old list would leave
// the server routing work to a client that no longer accepts it.
func (s *AgentsStore) SetCapabilities(ctx context.Context, id uuid.UUID, caps *models.AgentCapabilities) error {
	encoded, err := json.Marshal(caps)
	if err != nil {
		return fmt.Errorf("encode agent capabilities: %w", err)
	}

	_, err = s.q.ExecContext(ctx,
		`UPDATE agents SET capabilities = $2 WHERE id = $1`, id, encoded)
	if err != nil {
		return fmt.Errorf("update agent capabilities: %w", err)
	}
	return nil
}

// UpdateAgentStatus sets status and bumps last_seen.
func (s *AgentsStore) UpdateAgentStatus(ctx context.Context, id uuid.UUID, status string, lastSeen any) error {
	_, err := s.q.ExecContext(ctx, `
		UPDATE agents SET status = $1, last_seen = $2 WHERE id = $3
	`, status, lastSeen, id)
	if err != nil {
		return fmt.Errorf("update agent status: %w", err)
	}
	return nil
}

// UpdateAgentStatusTx is UpdateAgentStatus inside a caller's transaction.
//
// The agent's status is part of the same fact as the task's: an agent reporting
// "blocked" while its task says "in progress" tells an operator two different
// stories about why nothing is moving. Both are written in one transaction.
func (s *AgentsStore) UpdateAgentStatusTx(ctx context.Context, q Querier, id uuid.UUID, status string, lastSeen any) error {
	if q == nil {
		q = s.q
	}
	_, err := q.ExecContext(ctx, `
		UPDATE agents SET status = $1, last_seen = $2 WHERE id = $3
	`, status, lastSeen, id)
	if err != nil {
		return fmt.Errorf("update agent status in tx: %w", err)
	}
	return nil
}

// DeleteAgentCascade removes an agent and its related rows in a single
// transaction. Caller owns the transaction.
func (s *AgentsStore) DeleteAgentCascade(ctx context.Context, tx Querier, id uuid.UUID) (projectID uuid.UUID, err error) {
	if _, err := tx.ExecContext(ctx, `DELETE FROM contexts WHERE task_id IN (SELECT id FROM tasks WHERE agent_id = $1)`, id); err != nil {
		return uuid.Nil, fmt.Errorf("delete related contexts: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE agent_id = $1`, id); err != nil {
		return uuid.Nil, fmt.Errorf("delete related tasks: %w", err)
	}
	tag, err := tx.ExecContext(ctx, `DELETE FROM agents WHERE id = $1`, id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("delete agent: %w", err)
	}
	rows, err := tag.RowsAffected()
	if err != nil {
		return uuid.Nil, fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return uuid.Nil, fmt.Errorf("agent %s not found", id)
	}
	return
}

// BeginTx starts a transaction; see ProjectsStore.BeginTx for the same shape.
func (s *AgentsStore) BeginTx(ctx context.Context) (*sql.Tx, error) {
	tx, ok := s.q.(txBeginner)
	if !ok {
		return nil, errors.New("AgentsStore: underlying Querier does not support transactions; pass *sql.DB")
	}
	return tx.BeginTx(ctx, nil)
}

// GetAgentProjectID returns just the project_id for an agent. Used by the
// delete flow to look up the broadcast target after the delete.
func (s *AgentsStore) GetAgentProjectID(ctx context.Context, q Querier, id uuid.UUID) (uuid.UUID, error) {
	var pid uuid.UUID
	err := q.QueryRowContext(ctx, `SELECT project_id FROM agents WHERE id = $1`, id).Scan(&pid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("agent %s not found", id)
		}
		return uuid.Nil, fmt.Errorf("select agent project: %w", err)
	}
	return pid, nil
}
