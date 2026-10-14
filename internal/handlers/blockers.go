package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/coordination"
	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/httpx"
	"github.com/techbuzzz/agent-shaker/internal/middleware"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

// BlockerHandler is the human-facing surface of the Coordination Plane: the
// screen where someone reads what an agent is stuck on and answers it.
//
// Every mutating method here closes the loop the query layer was shaped for —
// the blocker row, the task status and the agent status all move together, so a
// human never sees "resolved" next to an agent that is still blocked.
type BlockerHandler struct {
	store    *queries.BlockersStore
	deps     *queries.TaskDependenciesStore
	agents   *queries.AgentsStore
	tasks    *queries.TasksStore
	resolver *coordination.DependencyResolver
	hub      *websocket.Hub
}

// NewBlockerHandler returns a handler backed by the supplied stores. The
// resolver is injected rather than constructed here because it is shared with
// the task-completion path, and two resolvers would mean two answers to the
// question of whether a task is ready.
func NewBlockerHandler(
	store *queries.BlockersStore,
	deps *queries.TaskDependenciesStore,
	agents *queries.AgentsStore,
	tasks *queries.TasksStore,
	resolver *coordination.DependencyResolver,
	hub *websocket.Hub,
) *BlockerHandler {
	return &BlockerHandler{
		store:    store,
		deps:     deps,
		agents:   agents,
		tasks:    tasks,
		resolver: resolver,
		hub:      hub,
	}
}

// projectPathID reads the project id from the path.
//
// muxVars predates the snake_case route names and only bridges id, taskId and
// projectId, so the newer route variables are read through PathValue directly
// with muxVars as the fallback — one call here rather than every handler
// repeating the guess.
func projectPathID(r *http.Request) string {
	if v := r.PathValue("project_id"); v != "" {
		return v
	}
	return muxVars(r)["projectId"]
}

// blockerPathID reads the blocker id from the path, falling back to the generic
// {id} variable so the handler works under either registration.
func blockerPathID(r *http.Request) string {
	if v := r.PathValue("blocker_id"); v != "" {
		return v
	}
	return muxVars(r)["id"]
}

// available reports whether the handler can serve a request at all. Both stores
// are checked because every method reads tasks as well as blockers; a partial
// wiring would otherwise panic inside the recovery middleware.
func (h *BlockerHandler) available() bool {
	return h.store.Available() && h.tasks.Available()
}

// ListBlockers returns a project's blockers, oldest first, optionally filtered
// by status. The empty slice matters: the UI does `.map(...)` on the result and
// a JSON null turns a list into a crash on an empty project.
func (h *BlockerHandler) ListBlockers(w http.ResponseWriter, r *http.Request) {
	if !h.available() {
		handleNoStore(w, r)
		return
	}
	projectID, err := uuid.Parse(projectPathID(r))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid project_id: %v", httpx.ErrBadRequest, err))
		return
	}
	status := r.URL.Query().Get("status")
	if status != "" && !models.IsValidBlockerStatus(status) {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid status %q; must be one of: %s",
			httpx.ErrBadRequest, status, models.JoinValues(models.AllBlockerStatuses)))
		return
	}

	blockers, err := h.store.ListBlockers(r.Context(), projectID, status)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if blockers == nil {
		blockers = []models.Blocker{}
	}
	httpx.WriteJSON(w, http.StatusOK, blockers)
}

// GetBlocker returns one blocker.
//
// The project check is not decoration: the URL is project-scoped, so serving a
// row that belongs to a different project would hand a caller another team's
// blocker content through their own valid path. Missing and out-of-project both
// answer 404 for that reason — a 403 would confirm the id exists.
func (h *BlockerHandler) GetBlocker(w http.ResponseWriter, r *http.Request) {
	if !h.available() {
		handleNoStore(w, r)
		return
	}
	projectID, err := uuid.Parse(projectPathID(r))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid project_id: %v", httpx.ErrBadRequest, err))
		return
	}
	blockerID, err := uuid.Parse(blockerPathID(r))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid blocker id: %v", httpx.ErrBadRequest, err))
		return
	}

	blocker, err := h.store.GetBlocker(r.Context(), nil, blockerID)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("blocker not found: %w", httpx.ErrNotFound))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	if blocker.ProjectID != projectID {
		httpx.WriteError(w, r, fmt.Errorf("blocker not found: %w", httpx.ErrNotFound))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, blocker)
}

// RaiseBlocker records an agent's request for help and blocks the work it was
// doing.
//
// The task and agent reads are validation, not decoration: a blocker raised
// against another project's task would put that task's status and the other
// project's agent dashboard into a state nobody in this project can see.
func (h *BlockerHandler) RaiseBlocker(w http.ResponseWriter, r *http.Request) {
	if !h.available() {
		handleNoStore(w, r)
		return
	}
	projectID, err := uuid.Parse(projectPathID(r))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid project_id: %v", httpx.ErrBadRequest, err))
		return
	}
	var req models.CreateBlockerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w: %w", httpx.ErrBadRequest, err))
		return
	}
	if !models.IsValidBlockerType(string(req.BlockerType)) {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid blocker_type %q; must be one of: %s",
			httpx.ErrBadRequest, req.BlockerType, models.JoinValues(models.AllBlockerTypes)))
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		httpx.WriteError(w, r, fmt.Errorf("%w: title is required", httpx.ErrBadRequest))
		return
	}
	if len([]rune(title)) > 500 {
		// blockers.title is VARCHAR(500): a longer title is truncated by
		// PostgreSQL or rejected outright, and either way the human reads a
		// different string than the agent sent.
		httpx.WriteError(w, r, fmt.Errorf("%w: title must be at most 500 characters", httpx.ErrBadRequest))
		return
	}

	task, err := h.tasks.GetTask(r.Context(), req.TaskID)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("task not found: %w", httpx.ErrNotFound))
			return
		}
		httpx.WriteError(w, r, fmt.Errorf("get task: %w", err))
		return
	}
	if task.ProjectID != projectID {
		httpx.WriteError(w, r, fmt.Errorf("task not found: %w", httpx.ErrNotFound))
		return
	}
	if models.IsTerminalTaskStatus(string(task.Status)) {
		// A blocker on a done task has nobody to release: the resolve flow would
		// try to move it to ready and resurrect finished work.
		httpx.WriteError(w, r, fmt.Errorf("%w: cannot raise a blocker on a task with status %q",
			httpx.ErrBadRequest, task.Status))
		return
	}
	if task.AssignedTo == nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: task %s has no assigned agent to block",
			httpx.ErrBadRequest, task.ID))
		return
	}
	agentID := *task.AssignedTo

	agent, err := h.agents.GetAgent(r.Context(), agentID)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("agent not found: %w", httpx.ErrNotFound))
			return
		}
		httpx.WriteError(w, r, fmt.Errorf("get agent: %w", err))
		return
	}
	if agent.ProjectID != projectID {
		httpx.WriteError(w, r, fmt.Errorf("agent not found: %w", httpx.ErrNotFound))
		return
	}

	blocker := models.Blocker{
		ID:          uuid.New(),
		TaskID:      task.ID,
		AgentID:     agentID,
		ProjectID:   projectID,
		Type:        req.BlockerType,
		Title:       title,
		Description: req.Description,
		Status:      models.BlockerOpen,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tx, err := h.store.BeginTx(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	if err := h.store.CreateBlocker(r.Context(), &blocker); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if task.Status != models.StatusTaskBlocked {
		// A task that is in progress stops being in progress when the agent
		// raises a blocker on it: the agent has just said it cannot proceed, and
		// leaving the task in "in progress" makes the dashboard report work that
		// is not happening. MarkTaskBlocked covers pending and ready (a task
		// waiting on a dependency is already blocked); in_progress needs the
		// explicit status write.
		if task.Status == models.StatusTaskInProgress {
			if err := h.tasks.UpdateTaskStatusTx(r.Context(), tx, task.ID, string(models.StatusTaskBlocked)); err != nil {
				httpx.WriteError(w, r, err)
				return
			}
		} else if err := h.deps.MarkTaskBlocked(r.Context(), tx, task.ID); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	// The agent status write is not in the transaction: AgentsStore's status
	// helpers bind to the pool, not to a caller-supplied Querier. It runs after
	// the commit so a rolled-back blocker cannot leave an agent blocked with
	// nothing on record.
	if err := h.agents.UpdateAgentStatus(r.Context(), agentID, string(models.StatusAgentBlocked), time.Now()); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	h.hub.BroadcastToProject(projectID, "blocker_raised", map[string]any{
		"blocker_id":   blocker.ID,
		"task_id":      task.ID,
		"agent_id":     agentID,
		"project_id":   projectID,
		"agent_name":   agent.Name,
		"blocker_type": string(blocker.Type),
		"title":        blocker.Title,
		"description":  blocker.Description,
	})
	httpx.WriteJSON(w, http.StatusCreated, blocker)
}

// ResolveBlocker answers a blocker with something the agent can act on.
//
// The resolution is required: a blocker closed with an empty string leaves the
// agent in exactly the state it was in before it asked, minus the reason it
// stopped.
func (h *BlockerHandler) ResolveBlocker(w http.ResponseWriter, r *http.Request) {
	var req models.ResolveBlockerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w: %w", httpx.ErrBadRequest, err))
		return
	}
	h.closeBlocker(w, r, models.BlockerResolved, req.Resolution, req.ResolvedBy, "blocker_resolved")
}

// DismissBlocker closes a blocker that was not worth answering.
//
// A reason is mandatory even though there is no answer: "why was this dropped"
// is the only thing that stops the same agent re-raising it tomorrow.
func (h *BlockerHandler) DismissBlocker(w http.ResponseWriter, r *http.Request) {
	var req models.DismissBlockerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w: %w", httpx.ErrBadRequest, err))
		return
	}
	h.closeBlocker(w, r, models.BlockerDismissed, req.Reason, req.ResolvedBy, "blocker_resolved")
}

// closeBlocker is the shared body of the resolve and dismiss flows. They differ
// only in the terminal status, the field the human's text arrives in, and the
// fact that dismiss requires it — everything after the close is identical, and
// two copies of the release logic would be two things to keep in step.
func (h *BlockerHandler) closeBlocker(
	w http.ResponseWriter,
	r *http.Request,
	status models.BlockerStatus,
	resolution string,
	resolvedBy uuid.UUID,
	event string,
) {
	if !h.available() {
		handleNoStore(w, r)
		return
	}
	projectID, err := uuid.Parse(projectPathID(r))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid project_id: %v", httpx.ErrBadRequest, err))
		return
	}
	blockerID, err := uuid.Parse(blockerPathID(r))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid blocker id: %v", httpx.ErrBadRequest, err))
		return
	}
	resolution = strings.TrimSpace(resolution)
	if resolution == "" {
		httpx.WriteError(w, r, fmt.Errorf("%w: resolution is required; a blocker closed without an answer leaves the agent blocked",
			httpx.ErrBadRequest))
		return
	}

	tx, err := h.store.BeginTx(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	blocker, err := h.store.GetBlocker(r.Context(), tx, blockerID)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("blocker not found: %w", httpx.ErrNotFound))
			return
		}
		httpx.WriteError(w, r, fmt.Errorf("get blocker: %w", err))
		return
	}
	if blocker.ProjectID != projectID {
		httpx.WriteError(w, r, fmt.Errorf("blocker not found: %w", httpx.ErrNotFound))
		return
	}

	// Read inside the transaction for the status only; the task store's GetTask
	// reads through the pool, so the value is a consistent-enough snapshot for
	// "was this being worked on when the blocker landed".
	task, err := h.tasks.GetTask(r.Context(), blocker.TaskID)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("task not found: %w", httpx.ErrNotFound))
			return
		}
		httpx.WriteError(w, r, fmt.Errorf("get task: %w", err))
		return
	}
	wasInProgress := task.Status == models.StatusTaskInProgress

	var resolvedByPtr *uuid.UUID
	if resolvedBy != uuid.Nil {
		resolvedByPtr = &resolvedBy
	}
	if _, err := h.store.CloseBlocker(r.Context(), tx, blockerID, status, resolution, resolvedByPtr); err != nil {
		if errors.Is(err, queries.ErrBlockerClosed) {
			writeBlockerConflict(w, r, "blocker was already resolved or dismissed")
			return
		}
		httpx.WriteError(w, r, fmt.Errorf("close blocker: %w", err))
		return
	}

	agentFreed, err := h.releaseAfterClose(r.Context(), tx, blocker, task, wasInProgress)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := tx.Commit(); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	// Outside the transaction on purpose — see the note in RaiseBlocker: the
	// agents store binds to the pool, and releasing an agent after the blocker
	// state is durable is what keeps a failed commit from freeing work that is
	// still blocked.
	if blocker.AgentID != uuid.Nil && agentFreed {
		if err := h.agents.UpdateAgentStatus(r.Context(), blocker.AgentID, string(models.StatusAgentIdle), time.Now()); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
	}

	closed, err := h.store.GetBlocker(r.Context(), nil, blockerID)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("get blocker: %w", err))
		return
	}

	// The terminal status travels with the event. Both resolve and dismiss emit
	// blocker_resolved, and a client that cannot tell them apart files a
	// dismissal under "Answer" — the two are deliberately different facts: one
	// is an answer the agent acts on, the other is a reason it stops asking.
	h.hub.BroadcastToProject(projectID, event, map[string]any{
		"blocker_id": blockerID,
		"task_id":    blocker.TaskID,
		"agent_id":   blocker.AgentID,
		"project_id": projectID,
		"status":     string(status),
		"resolution": resolution,
	})
	httpx.WriteJSON(w, http.StatusOK, closed)
}

// releaseAfterClose decides what the task and the agent become once a blocker
// is answered. It runs inside the caller's transaction.
//
// Task: only when the task has no other open blocker. If it was not in progress
// and has no unfinished dependencies it becomes ready — claimable right now; if
// it still waits on a dependency it goes back to blocked and the dependency
// resolver promotes it later. A task that was in progress keeps its work.
//
// Agent: only when this was the agent's last open blocker. One that still has
// another blocker waiting on a human stays blocked.
func (h *BlockerHandler) releaseAfterClose(
	ctx context.Context,
	tx *sql.Tx,
	blocker *models.Blocker,
	task *models.Task,
	wasInProgress bool,
) (agentFreed bool, err error) {
	openTaskBlockers, err := h.store.CountOpenBlockersForTask(ctx, tx, blocker.TaskID)
	if err != nil {
		return false, fmt.Errorf("count open blockers for task: %w", err)
	}
	if openTaskBlockers == 0 {
		if err := h.resolver.ReleaseAfterBlockerResolution(ctx, tx, blocker.TaskID, wasInProgress); err != nil {
			return false, fmt.Errorf("release task after blocker resolution: %w", err)
		}
	}

	openAgentBlockers, err := h.store.CountOpenBlockersForAgent(ctx, tx, blocker.AgentID)
	if err != nil {
		return false, fmt.Errorf("count open blockers for agent: %w", err)
	}
	return openAgentBlockers == 0, nil
}

// writeBlockerConflict answers 409 for an already-closed blocker.
//
// httpx.WriteError derives its status from the error's sentinel and has no
// conflict case, and ErrBlockerClosed is not one of its mapped sentinels. The
// envelope is built here by hand rather than routing a loser of a resolve race
// through as a 500.
func writeBlockerConflict(w http.ResponseWriter, r *http.Request, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	_ = json.NewEncoder(w).Encode(httpx.ErrorEnvelope{
		Error: httpx.ErrorBody{
			Code:      "blocker_closed",
			Message:   message,
			RequestID: middleware.RequestIDFromContext(r.Context()),
		},
	})
}
