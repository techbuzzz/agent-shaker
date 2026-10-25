package coordination

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/events"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

// DependencyResolver promotes tasks to ready once everything they wait for is
// done.
//
// It is called after every task completion, from whichever surface completed it
// (REST or MCP), and it is deliberately the only place that decides a task is
// ready. Two independent implementations of that decision would disagree about
// the status vocabulary eventually, and a task stuck in blocked forever is
// invisible in every list except the one you are looking at.
type DependencyResolver struct {
	store  DependencyStore
	tasks  ReadyTaskStore
	hub    *websocket.Hub
	broker *events.Broker
	logger *slog.Logger
}

// DependencyStore is the subset of the dependency store the resolver needs.
//
// The q parameter is threaded through rather than closed over so the resolver
// can run inside a caller's transaction when the release is part of a larger
// operation (resolving a blocker writes to blockers, tasks and agents at once).
type DependencyStore interface {
	DependentTaskIDs(ctx context.Context, q queries.Querier, completedTaskID uuid.UUID) ([]uuid.UUID, error)
	CountUnfinishedDependencies(ctx context.Context, q queries.Querier, taskID uuid.UUID) (int, error)
	MarkTaskReady(ctx context.Context, q queries.Querier, taskID uuid.UUID) (bool, error)
	MarkTaskBlocked(ctx context.Context, q queries.Querier, taskID uuid.UUID) error
}

// ReadyTaskStore reads the fields a task_ready event has to carry.
type ReadyTaskStore interface {
	GetTask(ctx context.Context, id uuid.UUID) (*models.Task, error)
}

// NewDependencyResolver returns a resolver bound to the supplied stores.
func NewDependencyResolver(store DependencyStore, tasks ReadyTaskStore, hub *websocket.Hub, broker *events.Broker) *DependencyResolver {
	return &DependencyResolver{
		store:  store,
		tasks:  tasks,
		hub:    hub,
		broker: broker,
		logger: slog.Default().With("component", "dependency_resolver"),
	}
}

// ResolveAfterCompletion promotes every task that was waiting on completedTask.
//
// Returns the ids it actually moved, not the ones it considered. A caller that
// broadcasts on the difference between "checked" and "changed" sends a task_ready
// event for a task that was already ready, and the UI announces work that
// nothing just unblocked.
func (r *DependencyResolver) ResolveAfterCompletion(ctx context.Context, completedTaskID uuid.UUID) ([]uuid.UUID, error) {
	dependents, err := r.store.DependentTaskIDs(ctx, nil, completedTaskID)
	if err != nil {
		return nil, err
	}
	if len(dependents) == 0 {
		return nil, nil
	}

	promoted := make([]uuid.UUID, 0, len(dependents))
	for _, taskID := range dependents {
		ready, err := r.maybePromote(ctx, taskID)
		if err != nil {
			// One stuck dependency must not stop the others from being unblocked.
			r.logger.Error("resolving dependent task failed", "task_id", taskID, "error", err)
			continue
		}
		if ready {
			promoted = append(promoted, taskID)
		}
	}
	return promoted, nil
}

// maybePromote checks one task's predecessors and moves it to ready if they are
// all done.
func (r *DependencyResolver) maybePromote(ctx context.Context, taskID uuid.UUID) (bool, error) {
	unfinished, err := r.store.CountUnfinishedDependencies(ctx, nil, taskID)
	if err != nil {
		return false, err
	}
	if unfinished > 0 {
		return false, nil
	}

	promoted, err := r.store.MarkTaskReady(ctx, nil, taskID)
	if err != nil || !promoted {
		return false, err
	}

	// Read after the write so the event carries the title and assignee the UI
	// shows, rather than making every client fetch the task it was just told
	// about.
	//
	// The nil check is not defensive padding: the promotion is the important
	// half and it has already happened, so a missing hub must not turn a
	// successful unlock into a panic that rolls the caller's transaction back.
	task, err := r.tasks.GetTask(ctx, taskID)
	if err != nil {
		// The promotion itself succeeded; only the notification failed. Say
		// so rather than reporting an error the caller would retry against.
		r.logger.Warn("task promoted but could not be read for broadcast",
			"task_id", taskID, "error", err)
		return true, nil
	}

	if r.hub != nil {
		r.hub.BroadcastToProject(task.ProjectID, "task_ready", map[string]any{
			"task_id":     task.ID,
			"project_id":  task.ProjectID,
			"title":       task.Title,
			"status":      string(models.StatusTaskReady),
			"assigned_to": task.AssignedTo,
		})
	}

	events.Send(ctx, r.broker, events.New(events.TaskReady, task.ProjectID, events.Options{
		EntityType: events.EntityTask,
		EntityID:   task.ID,
		ActorType:  events.ActorService,
		Payload: map[string]any{
			events.PayloadTitle:  task.Title,
			events.PayloadStatus: string(models.StatusTaskReady),
		},
	}))

	return true, nil
}

// ReleaseAfterBlockerResolution decides what a task becomes once its blockers
// are answered.
//
// A task goes back to in_progress when it was already being worked on, and to
// ready when its dependencies are satisfied. Returning it to pending in the
// second case would lose the information that it is claimable right now.
func (r *DependencyResolver) ReleaseAfterBlockerResolution(ctx context.Context, q queries.Querier, taskID uuid.UUID, wasInProgress bool) error {
	unfinished, err := r.store.CountUnfinishedDependencies(ctx, q, taskID)
	if err != nil {
		return err
	}

	if unfinished > 0 {
		// Still waiting on a dependency: blocked is the honest state, and the
		// dependency resolver will promote it later.
		return r.store.MarkTaskBlocked(ctx, q, taskID)
	}
	if !wasInProgress {
		// It was not in progress before the blocker, so there is no work to
		// resume — ready is the correct landing spot.
		_, err := r.store.MarkTaskReady(ctx, q, taskID)
		return err
	}
	// In progress before the blocker and still in progress after: nothing to do.
	return nil
}
