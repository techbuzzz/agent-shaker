package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/coordination"
	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/models"
)

// Coordination Plane tools.
//
// These live beside the older MCP tools rather than inside handler.go because
// they are the only ones that need the typed stores (blockers, dependencies,
// heartbeats) and the dependency resolver: everything else in this package
// talks to h.db directly. The conventions are the same as handler.go —
// JSON in, JSON string out, an error payload with isError=true, `false` on
// success — so an agent cannot tell which file a tool was written in.

// missingCoordinationStore names the first store a coordination tool needs and
// that was not wired by NewMCPHandlerWithCoordination.
//
// Answering with the store's name is deliberate: a handler built by the plain
// NewMCPHandler is a legitimate configuration (the REST surface serves the same
// data), so "tool unavailable" with a reason is more useful than a nil-pointer
// panic in the Recovery middleware.
func (h *MCPHandler) missingCoordinationStore() string {
	switch {
	case h.blockers == nil:
		return "blockers"
	case h.deps == nil:
		return "task_dependencies"
	case h.agents == nil:
		return "agents"
	case h.tasks == nil:
		return "tasks"
	case h.heartbeats == nil:
		return "heartbeats"
	}
	return ""
}

// optionalString reads a string argument that is allowed to be absent.
func optionalString(args map[string]interface{}, key string) string {
	s, _ := args[key].(string)
	return strings.TrimSpace(s)
}

// optionalUUID parses an optional uuid argument. An empty value means absent,
// which is not the same as a malformed one: the second is an error the agent
// can fix, the first is a legitimate "not specified".
func optionalUUID(args map[string]interface{}, key string) (*uuid.UUID, error) {
	raw := optionalString(args, key)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%s must be a uuid: %w", key, err)
	}
	return &id, nil
}

// optionalProgress reads progress_pct, which is bounded by a database CHECK
// constraint. Rejecting out-of-range values here turns a 500 into a message the
// agent can act on.
func optionalProgress(args map[string]interface{}) (*int, error) {
	raw, present := args["progress_pct"]
	if !present || raw == nil {
		return nil, nil
	}
	// JSON numbers decode as float64, but an integer-valued literal is still a
	// float64 on this path; accept both rather than rejecting a valid 0.
	var value float64
	switch v := raw.(type) {
	case float64:
		value = v
	case int:
		value = float64(v)
	case json.Number:
		parsed, err := v.Float64()
		if err != nil {
			return nil, errors.New("progress_pct must be a number between 0 and 100")
		}
		value = parsed
	default:
		return nil, errors.New("progress_pct must be a number between 0 and 100")
	}
	if value != float64(int(value)) {
		return nil, errors.New("progress_pct must be a whole number between 0 and 100")
	}
	pct := int(value)
	if pct < 0 || pct > 100 {
		return nil, errors.New("progress_pct must be between 0 and 100")
	}
	return &pct, nil
}

// executeSendHeartbeat records liveness and renews the caller's lease.
//
// The agent is taken from the connection, never from the arguments: a heartbeat
// is a statement about the caller, and accepting an agent_id would let any
// connected agent keep another one alive.
func (h *MCPHandler) executeSendHeartbeat(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if ctx.AgentID == "" {
		return `{"error": "No agent_id configured in MCP connection URL. Add ?agent_id=UUID to the URL."}`, true
	}
	if missing := h.missingCoordinationStore(); missing != "" {
		return fmt.Sprintf(`{"error": "heartbeats are not available on this server: %s store not configured"}`, missing), true
	}

	agentID, err := uuid.Parse(ctx.AgentID)
	if err != nil {
		return fmt.Sprintf(`{"error": "agent_id from the MCP connection is not a uuid: %s"}`, err.Error()), true
	}

	status := optionalString(args, "status")
	if status == "" {
		return fmt.Sprintf(`{"error": "status is required (%s)"}`,
			models.JoinValues(models.AllHeartbeatStatuses)), true
	}
	// The heartbeat vocabulary is deliberately not the agent-status one: an
	// agent reporting what it is doing says "working", and the stored agent
	// status has no such value. See models.HeartbeatStatus.
	if !models.IsValidHeartbeatStatus(status) {
		return fmt.Sprintf(`{"error": "invalid status, must be one of: %s"}`,
			models.JoinValues(models.AllHeartbeatStatuses)), true
	}

	taskID, err := optionalUUID(args, "task_id")
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	progress, err := optionalProgress(args)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	var eta *time.Time
	if raw := optionalString(args, "eta"); raw != "" {
		parsed, parseErr := time.Parse(time.RFC3339, raw)
		if parseErr != nil {
			return `{"error": "eta must be an RFC3339 timestamp, e.g. 2026-01-31T15:04:05Z"}`, true
		}
		utc := parsed.UTC()
		eta = &utc
	}

	metadata, _ := args["metadata"].(map[string]interface{})

	qctx := ctx.requestContext()
	ttl := coordination.DefaultConfig().LeaseTTL

	// The payload is the agent's own report and is stored verbatim as JSONB: the
	// server reads two known keys out of it and leaves the rest alone.
	payload := map[string]interface{}{}
	if taskID != nil {
		payload["task_id"] = taskID.String()
	}
	if progress != nil {
		payload["progress_pct"] = *progress
	}
	if eta != nil {
		payload["eta"] = eta.Format(time.RFC3339)
	}
	if len(metadata) > 0 {
		payload["metadata"] = metadata
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf(`{"error": "Failed to encode heartbeat payload: %s"}`, err.Error()), true
	}

	// The agent row stores the fleet's view; the reported activity stays in the
	// heartbeat history and in this response.
	agent, err := h.agents.RecordHeartbeat(qctx, agentID, models.HeartbeatStatus(status).ToAgentStatus(), taskID, progress, payloadJSON, ttl)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	hb := &models.AgentHeartbeat{
		AgentID:     agentID,
		Status:      status,
		ProjectID:   &agent.ProjectID,
		TaskID:      taskID,
		ProgressPct: progress,
	}
	if err := h.heartbeats.RecordHeartbeat(qctx, nil, hb); err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	leaseExpiresAt := time.Now().Add(ttl)
	if h.hub != nil {
		h.hub.BroadcastToProject(agent.ProjectID, "agent_heartbeat", map[string]interface{}{
			"agent_id":         agent.ID,
			"project_id":       agent.ProjectID,
			"name":             agent.Name,
			"status":           agent.Status,
			"task_id":          taskID,
			"progress_pct":     progress,
			"lease_expires_at": leaseExpiresAt,
		})
	}

	resultJSON, _ := json.MarshalIndent(map[string]interface{}{
		"success":               true,
		"agent_id":              agentID,
		"status":                agent.Status,
		"lease_ttl_seconds":     int(ttl.Seconds()),
		"next_heartbeat_before": leaseExpiresAt,
		"broadcast":             h.hub != nil,
	}, "", "  ")
	return string(resultJSON), false
}

// executeCompleteTaskWithHandoff closes a task and opens its successor.
//
// The two halves are one operation on purpose: completing the task and then
// crashing leaves finished work with no next step, which is exactly the state a
// team cannot recover from on its own.
func (h *MCPHandler) executeCompleteTaskWithHandoff(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if ctx.AgentID == "" {
		return `{"error": "No agent_id configured in MCP connection URL. Add ?agent_id=UUID to the URL."}`, true
	}
	if missing := h.missingCoordinationStore(); missing != "" {
		return fmt.Sprintf(`{"error": "handoffs are not available on this server: %s store not configured"}`, missing), true
	}

	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return `{"error": "task_id is required"}`, true
	}
	completedID, err := uuid.Parse(taskID)
	if err != nil {
		return fmt.Sprintf(`{"error": "task_id must be a uuid: %s"}`, err.Error()), true
	}

	successorTitle := optionalString(args, "successor_title")
	if successorTitle == "" {
		return `{"error": "successor_title is required"}`, true
	}
	handoffContext, hasContext := args["handoff_context"].(string)
	if !hasContext || strings.TrimSpace(handoffContext) == "" {
		return `{"error": "handoff_context is required: it is what the next agent needs and cannot re-derive"}`, true
	}
	output := optionalString(args, "output")
	successorDescription, _ := args["successor_description"].(string)

	successorAssignedTo, err := optionalUUID(args, "successor_assigned_to")
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	agentID, err := uuid.Parse(ctx.AgentID)
	if err != nil {
		return fmt.Sprintf(`{"error": "agent_id from the MCP connection is not a uuid: %s"}`, err.Error()), true
	}

	qctx := ctx.requestContext()

	task, err := h.tasks.GetTask(qctx, completedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Sprintf(`{"error": "Task not found: %s"}`, taskID), true
		}
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	// Ownership is checked before anything is written: an agent completing
	// another agent's task would both close the wrong work and open a successor
	// under its own name.
	if task.AssignedTo != nil && *task.AssignedTo != agentID {
		return fmt.Sprintf(
			`{"error": "Task is assigned to a different agent", "task_id": %q, "assigned_to": %q}`,
			taskID, task.AssignedTo.String()), true
	}
	if models.IsTerminalTaskStatus(string(task.Status)) {
		return fmt.Sprintf(
			`{"error": "Task is already %s and cannot be completed", "task_id": %q, "status": %q}`,
			task.Status, taskID, task.Status), true
	}

	successorID := uuid.New()

	tx, err := h.db.BeginTx(qctx, nil)
	if err != nil {
		return fmt.Sprintf(`{"error": "Failed to start transaction: %s"}`, err.Error()), true
	}
	// Rollback is a no-op once the transaction has been committed, so deferring
	// it unconditionally is safe and covers every early return below.
	defer func() { _ = tx.Rollback() }()

	// Same ownership guard and terminal-state refusal as completeTaskSQL in
	// handler.go, so the invariant is enforced identically on the handoff path.
	var completedTitle string
	err = tx.QueryRowContext(qctx, `
		UPDATE tasks
		   SET status = 'done', output = $2, updated_at = NOW()
		 WHERE id = $1
		   AND (assigned_to IS NULL OR assigned_to = $3)
		   AND status NOT IN ('done', 'cancelled')
		RETURNING title
	`, completedID, output, agentID).Scan(&completedTitle)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Sprintf(
			`{"error": "Task could not be completed: it was finished, cancelled or reassigned in the meantime", "task_id": %q}`,
			taskID), true
	case err != nil:
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	// The successor is inserted with predecessor_task_id and handoff_context set:
	// "this task exists because that one finished, and here is what it learned"
	// is history, and it has to survive a dependency being added later.
	_, err = tx.ExecContext(qctx, `
		INSERT INTO tasks (id, project_id, title, description, status, priority,
		                   created_by, assigned_to, predecessor_task_id, handoff_context,
		                   created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'ready', 'medium', $5, $6, $7, $8, NOW(), NOW())
	`, successorID, task.ProjectID, successorTitle, successorDescription, agentID,
		successorAssignedTo, completedID, handoffContext)
	if err != nil {
		return fmt.Sprintf(`{"error": "Failed to create successor task: %s"}`, err.Error()), true
	}

	// The edge is written inside the same transaction as the completion and the
	// successor. A task that exists with no record of what it waits for is
	// claimable when it should not be, and a cycle guard that runs against
	// another connection cannot see this transaction's own writes.
	if err := h.deps.AddDependencyTx(qctx, tx, successorID, completedID); err != nil {
		if errors.Is(err, queries.ErrCyclicDependency) {
			return fmt.Sprintf(
				`{"error": "cyclic dependency: task %s cannot depend on task %s", "task_id": %q, "depends_on_task_id": %q}`,
				successorID, completedID, successorID.String(), taskID), true
		}
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	if err := tx.Commit(); err != nil {
		return fmt.Sprintf(`{"error": "Failed to commit handoff: %s"}`, err.Error()), true
	}

	// ready when nothing is outstanding, blocked otherwise. Counted rather than
	// assumed: an agent that pre-declared an edge on a task id it guessed must
	// not get a successor that claims to be startable.
	successorStatus := models.StatusTaskReady
	unfinished, err := h.deps.CountUnfinishedDependencies(qctx, nil, successorID)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}
	if unfinished > 0 {
		successorStatus = models.StatusTaskBlocked
		if err := h.tasks.UpdateTaskStatus(qctx, successorID, string(models.StatusTaskBlocked)); err != nil {
			return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
		}
	}

	if h.hub != nil {
		h.hub.BroadcastToProject(task.ProjectID, "task_handoff", map[string]interface{}{
			"from_task_id":    completedID,
			"to_task_id":      successorID,
			"from_agent_id":   agentID,
			"to_agent_id":     successorAssignedTo,
			"handoff_context": handoffContext,
			"project_id":      task.ProjectID,
		})
	}

	// Anything else waiting on the completed task is promoted too, not just the
	// successor: the resolver is the only place that decides readiness.
	if h.resolver != nil {
		if _, err := h.resolver.ResolveAfterCompletion(qctx, completedID); err != nil {
			return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
		}
	}

	resultJSON, _ := json.MarshalIndent(map[string]interface{}{
		"success":           true,
		"task_id":           completedID,
		"title":             completedTitle,
		"status":            string(models.StatusTaskDone),
		"successor_task_id": successorID,
		"successor_title":   successorTitle,
		"successor_status":  string(successorStatus),
		"project_id":        task.ProjectID,
		"broadcast":         h.hub != nil,
	}, "", "  ")
	return string(resultJSON), false
}

// executeAddTaskDependency records that one task waits for another, and blocks
// the waiter until the predecessor is done.
func (h *MCPHandler) executeAddTaskDependency(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if missing := h.missingCoordinationStore(); missing != "" {
		return fmt.Sprintf(`{"error": "task dependencies are not available on this server: %s store not configured"}`, missing), true
	}

	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return `{"error": "task_id is required"}`, true
	}
	dependsOnID, ok := args["depends_on_task_id"].(string)
	if !ok || dependsOnID == "" {
		return `{"error": "depends_on_task_id is required"}`, true
	}
	taskUUID, err := uuid.Parse(taskID)
	if err != nil {
		return fmt.Sprintf(`{"error": "task_id must be a uuid: %s"}`, err.Error()), true
	}
	dependsOnUUID, err := uuid.Parse(dependsOnID)
	if err != nil {
		return fmt.Sprintf(`{"error": "depends_on_task_id must be a uuid: %s"}`, err.Error()), true
	}

	qctx := ctx.requestContext()
	task, err := h.tasks.GetTask(qctx, taskUUID)
	if err != nil {
		return fmt.Sprintf(`{"error": "Task not found: %s"}`, taskID), true
	}

	if err := h.deps.AddDependency(qctx, taskUUID, dependsOnUUID); err != nil {
		// A cycle is not a database failure but a statement about the plan, so
		// it is answered with both ids: the agent has to know which two tasks
		// are the ones that cannot both happen.
		if errors.Is(err, queries.ErrCyclicDependency) {
			return fmt.Sprintf(
				`{"error": "cyclic dependency: task %s cannot depend on task %s because the reverse edge already exists", "task_id": %q, "depends_on_task_id": %q}`,
				taskID, dependsOnID, taskID, dependsOnID), true
		}
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	status := string(task.Status)
	unfinished, err := h.deps.CountUnfinishedDependencies(qctx, nil, taskUUID)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}
	if unfinished > 0 && !models.IsTerminalTaskStatus(status) {
		if err := h.deps.MarkTaskBlocked(qctx, nil, taskUUID); err != nil {
			return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
		}
		status = string(models.StatusTaskBlocked)
	}

	if h.hub != nil {
		h.hub.BroadcastToProject(task.ProjectID, "task_updated", map[string]interface{}{
			"task_id":         taskUUID,
			"project_id":      task.ProjectID,
			"title":           task.Title,
			"status":          status,
			"unfinished_deps": unfinished,
		})
	}

	resultJSON, _ := json.MarshalIndent(map[string]interface{}{
		"success":                 true,
		"task_id":                 taskUUID,
		"depends_on_task_id":      dependsOnUUID,
		"unfinished_dependencies": unfinished,
		"status":                  status,
		"broadcast":               h.hub != nil,
	}, "", "  ")
	return string(resultJSON), false
}

// executeGetTaskDependencies returns both directions of a task's edges.
func (h *MCPHandler) executeGetTaskDependencies(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if missing := h.missingCoordinationStore(); missing != "" {
		return fmt.Sprintf(`{"error": "task dependencies are not available on this server: %s store not configured"}`, missing), true
	}

	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return `{"error": "task_id is required"}`, true
	}
	taskUUID, err := uuid.Parse(taskID)
	if err != nil {
		return fmt.Sprintf(`{"error": "task_id must be a uuid: %s"}`, err.Error()), true
	}

	qctx := ctx.requestContext()
	dependsOn, dependsFor, err := h.deps.ListDependencies(qctx, taskUUID)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	resultJSON, _ := json.MarshalIndent(map[string]interface{}{
		"task_id":     taskUUID,
		"depends_on":  h.dependencySummaries(qctx, dependsOn, func(d models.TaskDependency) uuid.UUID { return d.DependsOnTaskID }),
		"depends_for": h.dependencySummaries(qctx, dependsFor, func(d models.TaskDependency) uuid.UUID { return d.TaskID }),
	}, "", "  ")
	return string(resultJSON), false
}

// dependencySummaries renders one direction of the edge list as task id, title
// and status.
//
// Both arrays are always emitted, empty rather than null: an agent reading this
// is asking "what is between me and starting" and "what do I unblock when I
// finish", and a null in either answer is a case it has to special-case.
//
// A task that has since been deleted still appears, carrying its id alone —
// dropping it would hide the fact that an edge points at something that is gone.
func (h *MCPHandler) dependencySummaries(ctx context.Context, deps []models.TaskDependency, related func(models.TaskDependency) uuid.UUID) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(deps))
	for _, d := range deps {
		relatedID := related(d)
		entry := map[string]interface{}{"task_id": relatedID}
		if task, err := h.tasks.GetTask(ctx, relatedID); err == nil {
			entry["title"] = task.Title
			entry["status"] = string(task.Status)
		}
		out = append(out, entry)
	}
	return out
}

// executeRaiseBlocker records a structured request for help only a human can
// give, and puts both the task and the agent into the blocked state.
func (h *MCPHandler) executeRaiseBlocker(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if ctx.AgentID == "" {
		return `{"error": "No agent_id configured in MCP connection URL. Add ?agent_id=UUID to the URL."}`, true
	}
	if missing := h.missingCoordinationStore(); missing != "" {
		return fmt.Sprintf(`{"error": "blockers are not available on this server: %s store not configured"}`, missing), true
	}

	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return `{"error": "task_id is required"}`, true
	}
	taskUUID, err := uuid.Parse(taskID)
	if err != nil {
		return fmt.Sprintf(`{"error": "task_id must be a uuid: %s"}`, err.Error()), true
	}

	blockerType := optionalString(args, "blocker_type")
	if blockerType == "" {
		return fmt.Sprintf(`{"error": "blocker_type is required (%s)"}`,
			models.JoinValues(models.AllBlockerTypes)), true
	}
	if !models.IsValidBlockerType(blockerType) {
		return fmt.Sprintf(`{"error": "invalid blocker_type, must be one of: %s"}`,
			models.JoinValues(models.AllBlockerTypes)), true
	}

	title := optionalString(args, "title")
	if title == "" {
		return `{"error": "title is required"}`, true
	}
	if len(title) > 500 {
		return `{"error": "title must be 500 characters or fewer"}`, true
	}
	description, _ := args["description"].(string)

	agentID, err := uuid.Parse(ctx.AgentID)
	if err != nil {
		return fmt.Sprintf(`{"error": "agent_id from the MCP connection is not a uuid: %s"}`, err.Error()), true
	}

	qctx := ctx.requestContext()

	task, err := h.tasks.GetTask(qctx, taskUUID)
	if err != nil {
		return fmt.Sprintf(`{"error": "Task not found: %s"}`, taskID), true
	}

	blocker := &models.Blocker{
		ID:          uuid.New(),
		TaskID:      taskUUID,
		AgentID:     agentID,
		ProjectID:   task.ProjectID,
		Type:        models.BlockerType(blockerType),
		Title:       title,
		Description: description,
		Status:      models.BlockerOpen,
	}
	// Two blockers on one task are legitimate — an agent can be waiting on a
	// decision and a credential at the same time — so this is never deduped.
	// Deduping would lose one of the two asks and answer neither.
	if err := h.blockers.CreateBlocker(qctx, blocker); err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	// MarkTaskBlocked only moves tasks that are pending or ready: a task already
	// in progress stays in progress and simply stops advancing, which is what
	// "blocked" means for work an agent has already started.
	if err := h.deps.MarkTaskBlocked(qctx, nil, taskUUID); err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}
	if err := h.agents.UpdateAgentStatus(qctx, agentID, string(models.StatusAgentBlocked), time.Now()); err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	if h.hub != nil {
		h.hub.BroadcastToProject(task.ProjectID, "blocker_raised", map[string]interface{}{
			"blocker_id":   blocker.ID,
			"task_id":      taskUUID,
			"agent_id":     agentID,
			"project_id":   task.ProjectID,
			"blocker_type": blocker.Type,
			"title":        blocker.Title,
			"description":  blocker.Description,
			"status":       blocker.Status,
		})
	}

	resultJSON, _ := json.MarshalIndent(map[string]interface{}{
		"success":      true,
		"blocker_id":   blocker.ID,
		"task_id":      taskUUID,
		"agent_id":     agentID,
		"blocker_type": blocker.Type,
		"title":        blocker.Title,
		"status":       blocker.Status,
		"broadcast":    h.hub != nil,
	}, "", "  ")
	return string(resultJSON), false
}

// executeCheckBlockerStatus reads one blocker back.
func (h *MCPHandler) executeCheckBlockerStatus(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if missing := h.missingCoordinationStore(); missing != "" {
		return fmt.Sprintf(`{"error": "blockers are not available on this server: %s store not configured"}`, missing), true
	}

	blockerID, ok := args["blocker_id"].(string)
	if !ok || blockerID == "" {
		return `{"error": "blocker_id is required"}`, true
	}
	blockerUUID, err := uuid.Parse(blockerID)
	if err != nil {
		return fmt.Sprintf(`{"error": "blocker_id must be a uuid: %s"}`, err.Error()), true
	}

	qctx := ctx.requestContext()
	blocker, err := h.blockers.GetBlocker(qctx, nil, blockerUUID)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	// A blocker is another agent's request for help; its description carries
	// whatever that agent put in it. Only the raising agent may read it back, and
	// only when the connection actually identifies one.
	if ctx.AgentID != "" && blocker.AgentID.String() != ctx.AgentID {
		return fmt.Sprintf(
			`{"error": "blocker belongs to a different agent", "blocker_id": %q}`,
			blockerID), true
	}

	resultJSON, _ := json.MarshalIndent(blocker, "", "  ")
	return string(resultJSON), false
}

// executeGetMyBlockers lists the caller's own blockers.
func (h *MCPHandler) executeGetMyBlockers(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if ctx.AgentID == "" {
		return `{"error": "No agent_id configured in MCP connection URL. Add ?agent_id=UUID to the URL."}`, true
	}
	if missing := h.missingCoordinationStore(); missing != "" {
		return fmt.Sprintf(`{"error": "blockers are not available on this server: %s store not configured"}`, missing), true
	}

	agentID, err := uuid.Parse(ctx.AgentID)
	if err != nil {
		return fmt.Sprintf(`{"error": "agent_id from the MCP connection is not a uuid: %s"}`, err.Error()), true
	}

	status := optionalString(args, "status")
	if status != "" && !models.IsValidBlockerStatus(status) {
		return fmt.Sprintf(`{"error": "invalid status, must be one of: %s"}`,
			models.JoinValues(models.AllBlockerStatuses)), true
	}

	qctx := ctx.requestContext()
	blockers, err := h.blockers.ListBlockersByAgent(qctx, agentID, status)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}
	if blockers == nil {
		blockers = []models.Blocker{}
	}

	resultJSON, _ := json.MarshalIndent(map[string]interface{}{
		"agent_id": agentID,
		"status":   status,
		"count":    len(blockers),
		"blockers": blockers,
	}, "", "  ")
	return string(resultJSON), false
}

// resolveAfterCompletion promotes whatever was waiting on a completed task.
//
// A failure here is logged rather than returned: the task itself is already
// done, and reporting the whole call as failed would invite an agent to retry a
// completion that already happened. The dependent tasks stay blocked and the next
// completion picks them up.
func (h *MCPHandler) resolveAfterCompletion(ctx context.Context, completedTaskID string) {
	if h.resolver == nil {
		return
	}
	id, err := uuid.Parse(completedTaskID)
	if err != nil {
		// Unreachable in practice: the guarded UPDATE above compares the id
		// against a uuid column, so a malformed id already failed there.
		return
	}
	if _, err := h.resolver.ResolveAfterCompletion(ctx, id); err != nil {
		slog.ErrorContext(ctx, "mcp: resolving dependents of a completed task failed",
			"task_id", completedTaskID, "error", err)
	}
}
