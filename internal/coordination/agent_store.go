package coordination

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/models"
)

// AgentStore adapts *queries.AgentsStore to StaleAgentHandler.
//
// The adapter exists for one reason: Go has no covariant returns, so a store
// whose BeginTx returns *sql.Tx does not satisfy an interface that returns the
// narrower Tx. Without this, either the interface would have to be *sql.Tx —
// which would make the reaper untestable without a database — or every call site
// would need its own wrapper. One wrapper, here, keeps both properties.
type AgentStore struct {
	Store *queries.AgentsStore
}

func (a AgentStore) ListStaleAgents(ctx context.Context, limit int) ([]models.StaleAgent, error) {
	return a.Store.ListStaleAgents(ctx, limit)
}

func (a AgentStore) MarkAgentOffline(ctx context.Context, q queries.Querier, id uuid.UUID) (bool, error) {
	return a.Store.MarkAgentOffline(ctx, q, id)
}

func (a AgentStore) ReleaseAgentTasks(ctx context.Context, q queries.Querier, id uuid.UUID) ([]uuid.UUID, error) {
	return a.Store.ReleaseAgentTasks(ctx, q, id)
}

func (a AgentStore) BeginTx(ctx context.Context) (Tx, error) {
	var (
		tx  *sql.Tx
		err error
	)
	if tx, err = a.Store.BeginTx(ctx); err != nil {
		return nil, err
	}
	return tx, nil
}
