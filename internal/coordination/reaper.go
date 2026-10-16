// Package coordination holds the background logic that keeps the fleet honest:
// the reaper that takes away expired leases, and the resolver that moves tasks
// to ready when their predecessors finish.
//
// It is a package of its own rather than part of handlers or queries because
// neither owns it. The reaper is a goroutine with no HTTP surface; the resolver
// is called from both the MCP surface and the REST surface and must not know
// which one called it.
package coordination

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

// Config holds the tunables for the coordination background workers.
//
// Defaults are chosen so that the common case works with no configuration at
// all: a lease twice the reaper interval means one missed tick is not enough to
// declare an agent dead.
type Config struct {
	// LeaseTTL is how long an agent's heartbeat keeps it alive.
	LeaseTTL time.Duration
	// ReaperInterval is how often the reaper looks for expired leases.
	ReaperInterval time.Duration
}

// DefaultConfig returns the documented defaults: 60s lease, 30s reaper.
func DefaultConfig() Config {
	return Config{LeaseTTL: 60 * time.Second, ReaperInterval: 30 * time.Second}
}

// Tx is the transaction surface the reaper needs from a store: the querier the
// dependency writes take, plus commit and rollback.
//
// It is an interface rather than *sql.Tx so the reaper can be tested without a
// database. The behaviour worth testing here is what happens when an agent
// heartbeats between the sweep's SELECT and its UPDATE, and that needs a seam
// — with *sql.Tx in the signature there is no way to construct one.
type Tx interface {
	queries.Querier
	Commit() error
	Rollback() error
}

// StaleAgentHandler is the subset of the agents store the reaper needs.
type StaleAgentHandler interface {
	ListStaleAgents(ctx context.Context, limit int) ([]models.StaleAgent, error)
	MarkAgentOffline(ctx context.Context, q queries.Querier, agentID uuid.UUID) (bool, error)
	ReleaseAgentTasks(ctx context.Context, q queries.Querier, agentID uuid.UUID) ([]uuid.UUID, error)
	BeginTx(ctx context.Context) (Tx, error)
}

// StaleAgentReaper expires leases nobody renewed and returns the abandoned work
// to the backlog.
//
// The name is deliberate: it reaps agents, not leases. A lease is just the
// mechanism; what the operator cares about is an agent that stopped working and
// the tasks it was holding.
type StaleAgentReaper struct {
	store  StaleAgentHandler
	hub    *websocket.Hub
	config Config
	logger *slog.Logger
}

// NewStaleAgentReaper returns a reaper bound to the supplied store and hub.
// A nil hub disables broadcasting without disabling the reaper — the server can
// run without WebSocket clients, and silently skipping the expiry in that case
// would leave work stuck forever.
func NewStaleAgentReaper(store StaleAgentHandler, hub *websocket.Hub, config Config) *StaleAgentReaper {
	logger := slog.Default().With("component", "stale_agent_reaper")
	return &StaleAgentReaper{store: store, hub: hub, config: config, logger: logger}
}

// Timing lives in the scheduler now (see scheduler.go): the reaper is a task,
// not a goroutine with its own ticker. Keeping Run around would mean two
// places that can schedule a sweep, and the one nobody remembers is the one
// that runs twice.
//
// SweepOnce takes the context so a cancelled job aborts instead of finishing a
// sweep nobody will read.
// SweepOnce runs one pass and returns how many agents it took offline.
//
// Exported so it can be called directly from a test or an admin endpoint rather
// than only on the ticker — a reaper that can only be exercised by waiting is a
// reaper nobody tests.
func (r *StaleAgentReaper) SweepOnce(ctx context.Context) int {
	stale, err := r.store.ListStaleAgents(ctx, 100)
	if err != nil {
		// One failed sweep must not kill the loop: the next tick tries again.
		r.logger.Error("listing stale agents failed", "error", err)
		return 0
	}

	reaped := 0
	for _, agent := range stale {
		if err := r.reapAgent(ctx, agent); err != nil {
			r.logger.Error("reaping agent failed", "agent_id", agent.ID, "error", err)
			continue
		}
		reaped++
	}
	return reaped
}

// reapAgent expires one agent's lease and releases its work, atomically.
//
// The transaction is the point. Marking an agent offline without releasing its
// tasks leaves work that nobody owns and nothing that will ever assign it; the
// two statements have to succeed or fail together.
func (r *StaleAgentReaper) reapAgent(ctx context.Context, agent models.StaleAgent) error {
	tx, err := r.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	changed, err := r.store.MarkAgentOffline(ctx, tx, agent.ID)
	if err != nil {
		return err
	}
	if !changed {
		// The agent heartbeated between the SELECT and this UPDATE. Its lease is
		// still valid and its task is still its own — leaving it alone is the
		// correct outcome, not an error.
		return tx.Commit()
	}

	released, err := r.store.ReleaseAgentTasks(ctx, tx, agent.ID)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	r.broadcast(agent, released)
	return nil
}

// broadcast tells the UI what happened. The payload carries project_id because
// the hub routes by it — see websocket.Hub.broadcastMessage.
func (r *StaleAgentReaper) broadcast(agent models.StaleAgent, released []uuid.UUID) {
	if r.hub == nil {
		return
	}
	taskIDs := make([]string, 0, len(released))
	for _, id := range released {
		taskIDs = append(taskIDs, id.String())
	}
	r.hub.BroadcastToProject(agent.ProjectID, "agent_gone_stale", map[string]any{
		"agent_id":        agent.ID,
		"agent_name":      agent.Name,
		"project_id":      agent.ProjectID,
		"reassigned_task": taskIDs,
	})
}
