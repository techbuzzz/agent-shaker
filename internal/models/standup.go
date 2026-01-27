package models

import (
	"time"

	"github.com/google/uuid"
)

// DailyStandup represents a daily standup entry from an agent
type DailyStandup struct {
	ID          uuid.UUID `json:"id" db:"id"`
	AgentID     uuid.UUID `json:"agent_id" db:"agent_id"`
	ProjectID   uuid.UUID `json:"project_id" db:"project_id"`
	StandupDate time.Time `json:"standup_date" db:"standup_date"`
	Did         string    `json:"did" db:"did"`               // What I did yesterday
	Doing       string    `json:"doing" db:"doing"`           // What I'm doing today
	Done        string    `json:"done" db:"done"`             // What I plan to complete
	Blockers    string    `json:"blockers" db:"blockers"`     // Any blockers
	Challenges  string    `json:"challenges" db:"challenges"` // Current challenges
	References  string    `json:"references" db:"references"` // Links, docs, etc.
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CreateStandupRequest represents a request to create a daily standup
type CreateStandupRequest struct {
	AgentID     uuid.UUID `json:"agent_id"`
	ProjectID   uuid.UUID `json:"project_id"`
	StandupDate string    `json:"standup_date"` // YYYY-MM-DD format
	Did         string    `json:"did"`
	Doing       string    `json:"doing"`
	Done        string    `json:"done"`
	Blockers    string    `json:"blockers"`
	Challenges  string    `json:"challenges"`
	References  string    `json:"references"`
}

// UpdateStandupRequest represents a request to update a daily standup
type UpdateStandupRequest struct {
	Did        string `json:"did"`
	Doing      string `json:"doing"`
	Done       string `json:"done"`
	Blockers   string `json:"blockers"`
	Challenges string `json:"challenges"`
	References string `json:"references"`
}

// StandupWithAgent extends DailyStandup with agent information
type StandupWithAgent struct {
	DailyStandup
	AgentName string `json:"agent_name" db:"agent_name"`
	AgentRole string `json:"agent_role" db:"agent_role"`
	AgentTeam string `json:"agent_team" db:"agent_team"`
}

// AgentHeartbeat represents a heartbeat signal from an agent
type AgentHeartbeat struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	AgentID       uuid.UUID              `json:"agent_id" db:"agent_id"`
	HeartbeatTime time.Time              `json:"heartbeat_time" db:"heartbeat_time"`
	Status        string                 `json:"status" db:"status"`
	Metadata      map[string]interface{} `json:"metadata" db:"metadata"`
}

// CreateHeartbeatRequest represents a request to record a heartbeat
type CreateHeartbeatRequest struct {
	AgentID  uuid.UUID              `json:"agent_id"`
	Status   string                 `json:"status"`
	Metadata map[string]interface{} `json:"metadata"`
}
