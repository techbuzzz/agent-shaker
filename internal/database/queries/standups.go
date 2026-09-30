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

// StandupsStore is the typed query layer for the standups + agent_heartbeats
// tables. The model has more than CRUD (heartbeat tracking) so this store
// is wider than the others.
type StandupsStore struct {
	q Querier
}

func NewStandupsStore(q Querier) *StandupsStore {
	return &StandupsStore{q: q}
}

// Available reports whether the store has a usable Querier.
func (s *StandupsStore) Available() bool { return s.q != nil }

func (s *StandupsStore) CreateStandup(ctx context.Context, st *models.DailyStandup) error {
	_, err := s.q.ExecContext(ctx, `
		INSERT INTO standups (id, agent_id, project_id, standup_date, did, doing, done, blockers, challenges, reference_links, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, st.ID, st.AgentID, st.ProjectID, st.StandupDate, st.Did, st.Doing, st.Done, st.Blockers, st.Challenges, st.ReferenceLinks, st.CreatedAt, st.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert standup: %w", err)
	}
	return nil
}

func (s *StandupsStore) ListStandups(ctx context.Context, projectID *uuid.UUID) ([]models.DailyStandup, error) {
	args := []any{}
	q := `SELECT id, agent_id, project_id, standup_date, did, doing, done, blockers, challenges, reference_links, created_at, updated_at FROM standups`
	if projectID != nil {
		q += " WHERE project_id = $1"
		args = append(args, *projectID)
	}
	q += " ORDER BY standup_date DESC"
	rows, err := s.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list standups: %w", err)
	}
	defer rows.Close()
	var out []models.DailyStandup
	for rows.Next() {
		var st models.DailyStandup
		if err := rows.Scan(&st.ID, &st.AgentID, &st.ProjectID, &st.StandupDate, &st.Did, &st.Doing, &st.Done, &st.Blockers, &st.Challenges, &st.ReferenceLinks, &st.CreatedAt, &st.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan standup: %w", err)
		}
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating standups: %w", err)
	}
	return out, nil
}

func (s *StandupsStore) GetStandup(ctx context.Context, id uuid.UUID) (*models.DailyStandup, error) {
	var st models.DailyStandup
	err := s.q.QueryRowContext(ctx, `
		SELECT id, agent_id, project_id, standup_date, did, doing, done, blockers, challenges, reference_links, created_at, updated_at
		FROM standups WHERE id = $1
	`, id).Scan(&st.ID, &st.AgentID, &st.ProjectID, &st.StandupDate, &st.Did, &st.Doing, &st.Done, &st.Blockers, &st.Challenges, &st.ReferenceLinks, &st.CreatedAt, &st.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("standup %s not found", id)
		}
		return nil, fmt.Errorf("select standup: %w", err)
	}
	return &st, nil
}

func (s *StandupsStore) UpdateStandup(ctx context.Context, st *models.DailyStandup) error {
	_, err := s.q.ExecContext(ctx, `
		UPDATE standups
		SET did = $1, doing = $2, done = $3, blockers = $4, challenges = $5, reference_links = $6, updated_at = $7
		WHERE id = $8
	`, st.Did, st.Doing, st.Done, st.Blockers, st.Challenges, st.ReferenceLinks, st.UpdatedAt, st.ID)
	if err != nil {
		return fmt.Errorf("update standup: %w", err)
	}
	return nil
}

func (s *StandupsStore) DeleteStandup(ctx context.Context, id uuid.UUID) error {
	tag, err := s.q.ExecContext(ctx, `DELETE FROM standups WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete standup: %w", err)
	}
	rows, err := tag.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("standup %s not found", id)
	}
	return nil
}

// RecordHeartbeat inserts (or upserts) a heartbeat for an agent.
func (s *StandupsStore) RecordHeartbeat(ctx context.Context, hb *models.AgentHeartbeat) error {
	meta, err := json.Marshal(hb.Metadata)
	if err != nil {
		return fmt.Errorf("marshal heartbeat metadata: %w", err)
	}
	_, err = s.q.ExecContext(ctx, `
		INSERT INTO agent_heartbeats (id, agent_id, heartbeat_time, status, metadata)
		VALUES ($1, $2, $3, $4, $5::jsonb)
	`, hb.ID, hb.AgentID, hb.HeartbeatTime, hb.Status, meta)
	if err != nil {
		return fmt.Errorf("insert heartbeat: %w", err)
	}
	// Also bump agents.last_seen so dashboards see liveness without a join.
	_, err = s.q.ExecContext(ctx, `UPDATE agents SET last_seen = $1 WHERE id = $2`,
		hb.HeartbeatTime, hb.AgentID)
	if err != nil {
		return fmt.Errorf("update last_seen: %w", err)
	}
	return nil
}

// GetAgentHeartbeats returns recent heartbeats for an agent, newest first.
func (s *StandupsStore) GetAgentHeartbeats(ctx context.Context, agentID uuid.UUID, since time.Time, limit int) ([]models.AgentHeartbeat, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.q.QueryContext(ctx, `
		SELECT id, agent_id, heartbeat_time, status, metadata
		FROM agent_heartbeats
		WHERE agent_id = $1 AND heartbeat_time >= $2
		ORDER BY heartbeat_time DESC LIMIT $3
	`, agentID, since, limit)
	if err != nil {
		return nil, fmt.Errorf("list heartbeats: %w", err)
	}
	defer rows.Close()

	var out []models.AgentHeartbeat
	for rows.Next() {
		var hb models.AgentHeartbeat
		var metaBytes []byte
		if err := rows.Scan(&hb.ID, &hb.AgentID, &hb.HeartbeatTime, &hb.Status, &metaBytes); err != nil {
			return nil, fmt.Errorf("scan heartbeat: %w", err)
		}
		if len(metaBytes) > 0 {
			if err := json.Unmarshal(metaBytes, &hb.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal heartbeat metadata: %w", err)
			}
		}
		out = append(out, hb)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating heartbeats: %w", err)
	}
	return out, nil
}
