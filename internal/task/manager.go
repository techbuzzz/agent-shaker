package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
)

// Sentinel errors. Callers should use errors.Is to map these to HTTP status
// codes (ErrTaskNotFound → 404, ErrTaskTerminal → 409).
var (
	// ErrTaskNotFound is returned when a task ID does not exist in the store.
	ErrTaskNotFound = errors.New("task not found")

	// ErrTaskTerminal is returned when a caller tries to cancel a task that
	// has already reached a terminal state (completed, failed or canceled).
	ErrTaskTerminal = errors.New("task already in terminal state")

	// ErrNoExecutor is returned when a task is submitted to a Manager built
	// without a TaskExecutor. Agent Shaker does not run agents, so there is
	// nothing to execute an inbound A2A message against. Reporting this as a
	// failure is deliberate: the alternative — the previous behaviour of echoing
	// the message back and reporting "completed" — told a remote agent that work
	// had been done when none had.
	ErrNoExecutor = errors.New("no task executor configured: this deployment does not execute A2A tasks")
)

// TaskUpdate represents an update event for a task
type TaskUpdate struct {
	Event   string `json:"event"`
	Data    any    `json:"data"`
	IsFinal bool   `json:"is_final"`
}

// TaskExecutor defines the interface for task execution logic
type TaskExecutor interface {
	Execute(ctx context.Context, task *models.Task) (*models.Result, error)
}

// Manager handles A2A task lifecycle management
type Manager struct {
	store       Store
	executor    TaskExecutor
	subscribers map[string][]chan TaskUpdate
	mu          sync.RWMutex
	baseURL     string
}

// NewManager creates a new task manager
func NewManager(store Store, executor TaskExecutor, baseURL string) *Manager {
	return &Manager{
		store:       store,
		executor:    executor,
		subscribers: make(map[string][]chan TaskUpdate),
		baseURL:     baseURL,
	}
}

// CreateTask creates a new task and triggers async execution
func (m *Manager) CreateTask(ctx context.Context, req *models.SendMessageRequest) (*models.Task, error) {
	now := time.Now()
	task := &models.Task{
		ID:        uuid.New().String(),
		Status:    models.TaskStatusPending,
		Message:   req.Message,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := m.store.CreateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Execute out of band: the request context is cancelled as soon as the
	// handler returns, so cancellation must not follow the goroutine, while
	// values (trace and span ids, request-scoped config) must.
	go m.executeTask(context.WithoutCancel(ctx), task.ID)

	return task, nil
}

// GetTask retrieves a task by ID
func (m *Manager) GetTask(ctx context.Context, taskID string) (*models.Task, error) {
	return m.store.GetTask(ctx, taskID)
}

// ListTasks returns tasks matching the filter criteria
func (m *Manager) ListTasks(ctx context.Context, filter *Filter) ([]models.Task, error) {
	return m.store.ListTasks(ctx, filter)
}

// CancelTask attempts to cancel a running task. Returns ErrTaskNotFound when
// the task ID does not exist and ErrTaskTerminal when the task has already
// completed, failed or been cancelled. Callers should use errors.Is to map
// these to status codes (404 / 409 respectively).
func (m *Manager) CancelTask(ctx context.Context, taskID string) error {
	task, err := m.store.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	if task.Status.IsTerminal() {
		return fmt.Errorf("%w: status=%s", ErrTaskTerminal, task.Status)
	}

	task.Status = models.TaskStatusCanceled
	now := time.Now()
	task.CompletedAt = &now
	task.UpdatedAt = now
	task.Result = &models.Result{
		Content: "Task was cancelled",
		Format:  "text",
	}

	if err := m.store.UpdateTask(ctx, task); err != nil {
		return err
	}

	m.notifySubscribers(ctx, taskID, TaskUpdate{
		Event:   "cancelled",
		Data:    task,
		IsFinal: true,
	})

	return nil
}

// SubscribeToTask creates a channel for receiving task updates
func (m *Manager) SubscribeToTask(taskID string) <-chan TaskUpdate {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan TaskUpdate, 10)
	m.subscribers[taskID] = append(m.subscribers[taskID], ch)

	return ch
}

// UnsubscribeFromTask removes a subscription channel
func (m *Manager) UnsubscribeFromTask(taskID string, ch <-chan TaskUpdate) {
	m.mu.Lock()
	defer m.mu.Unlock()

	subs := m.subscribers[taskID]
	for i, sub := range subs {
		if sub == ch {
			m.subscribers[taskID] = append(subs[:i], subs[i+1:]...)
			close(sub)
			break
		}
	}

	// Clean up empty subscriber lists
	if len(m.subscribers[taskID]) == 0 {
		delete(m.subscribers, taskID)
	}
}

// notifySubscribers sends an update to all task subscribers.
//
// ctx is threaded through so a dropped-update warning still carries the trace
// of the operation that produced it: that is precisely the line someone needs
// when asking why a subscriber went quiet, and it is the one case where the
// channel is full and no downstream handler will report anything.
func (m *Manager) notifySubscribers(ctx context.Context, taskID string, update TaskUpdate) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, ch := range m.subscribers[taskID] {
		select {
		case ch <- update:
		default:
			// Channel full, skip this update
			slog.WarnContext(ctx, "subscriber channel full, dropping update", "task_id", taskID)
		}
	}
}

// executeTask runs the task execution logic asynchronously.
//
// ctx must already be detached from the request that created the task: the
// goroutine outlives that request, but it still carries its values so spans
// and trace context survive the hand-off.
func (m *Manager) executeTask(ctx context.Context, taskID string) {
	// Get the task
	task, err := m.store.GetTask(ctx, taskID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get task for execution", "task_id", taskID, "error", err)
		return
	}

	// Update status to running
	task.Status = models.TaskStatusRunning
	task.UpdatedAt = time.Now()

	if err := m.store.UpdateTask(ctx, task); err != nil {
		slog.ErrorContext(ctx, "failed to mark task running", "task_id", taskID, "error", err)
		return
	}

	m.notifySubscribers(ctx, taskID, TaskUpdate{
		Event: "status",
		Data: map[string]any{
			"task_id": taskID,
			"status":  string(models.TaskStatusRunning),
		},
		IsFinal: false,
	})

	// Execute the task. With no executor there is nothing that could have run,
	// so the task fails with a diagnosable reason instead of reporting a result
	// that was never produced.
	var result *models.Result
	var execErr error

	if m.executor == nil {
		execErr = ErrNoExecutor
	} else {
		result, execErr = m.executor.Execute(ctx, task)
	}

	// Execution may outlast a cancellation, so re-read before writing: a
	// CancelTask that arrived while the executor was running has already
	// reached a terminal state and must not be overwritten with a result the
	// caller has asked to discard.
	current, err := m.store.GetTask(ctx, taskID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to re-read task before finalising", "task_id", taskID, "error", err)
		return
	}
	if current.Status.IsTerminal() {
		slog.InfoContext(ctx, "task already terminal, discarding execution result",
			"task_id", taskID, "status", string(current.Status))
		return
	}
	task = current

	// Update task with result
	now := time.Now()
	task.CompletedAt = &now
	task.UpdatedAt = now

	if execErr != nil {
		task.Status = models.TaskStatusFailed
		task.Result = &models.Result{
			Content: execErr.Error(),
			Format:  "text",
		}
	} else {
		task.Status = models.TaskStatusCompleted
		task.Result = result
	}

	if err := m.store.UpdateTask(ctx, task); err != nil {
		slog.ErrorContext(ctx, "failed to update task with result", "task_id", taskID, "error", err)
		return
	}

	// Notify subscribers of the terminal state. The event is named after the
	// status that was actually reached, so a streaming client cannot read a
	// "completed" event for a task that failed.
	m.notifySubscribers(ctx, taskID, TaskUpdate{
		Event:   string(task.Status),
		Data:    task,
		IsFinal: true,
	})

	slog.InfoContext(ctx, "task reached terminal state", "task_id", taskID, "status", string(task.Status))
}

// GetStore returns the underlying store (for testing or advanced usage)
func (m *Manager) GetStore() Store {
	return m.store
}
