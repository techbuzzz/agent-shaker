package queries

import (
	"context"
	"database/sql"
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
	_, err := s.q.ExecContext(ctx, `
		INSERT INTO agents (id, project_id, name, role, team, status, last_seen, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, a.ID, a.ProjectID, a.Name, a.Role, a.Team, a.Status, a.LastSeen, a.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert agent: %w", err)
	}
	return nil
}

// ListAgents returns all agents, or agents filtered by project when
// projectID is non-nil.
func (s *AgentsStore) ListAgents(ctx context.Context, projectID *uuid.UUID) ([]models.Agent, error) {
	var rows *sql.Rows
	var err error
	if projectID != nil {
		rows, err = s.q.QueryContext(ctx, `
			SELECT id, project_id, name, role, team, status, last_seen, created_at
			FROM agents WHERE project_id = $1 ORDER BY created_at DESC
		`, *projectID)
	} else {
		rows, err = s.q.QueryContext(ctx, `
			SELECT id, project_id, name, role, team, status, last_seen, created_at
			FROM agents ORDER BY created_at DESC
		`)
	}
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	var out []models.Agent
	for rows.Next() {
		var a models.Agent
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.Name, &a.Role, &a.Team, &a.Status, &a.LastSeen, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating agents: %w", err)
	}
	return out, nil
}

// GetAgent returns one agent by id.
func (s *AgentsStore) GetAgent(ctx context.Context, id uuid.UUID) (*models.Agent, error) {
	var a models.Agent
	err := s.q.QueryRowContext(ctx, `
		SELECT id, project_id, name, role, team, status, last_seen, created_at
		FROM agents WHERE id = $1
	`, id).Scan(&a.ID, &a.ProjectID, &a.Name, &a.Role, &a.Team, &a.Status, &a.LastSeen, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("agent %s not found", id)
		}
		return nil, fmt.Errorf("select agent: %w", err)
	}
	return &a, nil
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
