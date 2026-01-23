package task

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
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

	// Trigger async execution
	go m.executeTask(task.ID)

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

// CancelTask attempts to cancel a running task
func (m *Manager) CancelTask(ctx context.Context, taskID string) error {
	task, err := m.store.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	if task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusFailed {
		return fmt.Errorf("cannot cancel task with status %s", task.Status)
	}

	task.Status = models.TaskStatusFailed
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

	m.notifySubscribers(taskID, TaskUpdate{
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

// notifySubscribers sends an update to all task subscribers
func (m *Manager) notifySubscribers(taskID string, update TaskUpdate) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, ch := range m.subscribers[taskID] {
		select {
		case ch <- update:
		default:
			// Channel full, skip this update
			log.Printf("Warning: subscriber channel full for task %s", taskID)
		}
	}
}

// executeTask runs the task execution logic asynchronously
func (m *Manager) executeTask(taskID string) {
	ctx := context.Background()

	// Get the task
	task, err := m.store.GetTask(ctx, taskID)
	if err != nil {
		log.Printf("Failed to get task %s for execution: %v", taskID, err)
		return
	}

	// Update status to running
	task.Status = models.TaskStatusRunning
	task.UpdatedAt = time.Now()

	if err := m.store.UpdateTask(ctx, task); err != nil {
		log.Printf("Failed to update task %s status to running: %v", taskID, err)
		return
	}

	m.notifySubscribers(taskID, TaskUpdate{
		Event: "status",
		Data: map[string]any{
			"task_id": taskID,
			"status":  string(models.TaskStatusRunning),
		},
		IsFinal: false,
	})

	// Execute the task
	var result *models.Result
	var execErr error

	if m.executor != nil {
		result, execErr = m.executor.Execute(ctx, task)
	} else {
		// Default execution: echo back the message content
		result = &models.Result{
			Content: fmt.Sprintf("Task received: %s", task.Message.Content),
			Format:  "text",
			Data: map[string]any{
				"original_message": task.Message.Content,
				"processed_at":     time.Now().Format(time.RFC3339),
			},
		}
	}

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
		log.Printf("Failed to update task %s with result: %v", taskID, err)
		return
	}

	// Notify subscribers of completion
	m.notifySubscribers(taskID, TaskUpdate{
		Event:   "completed",
		Data:    task,
		IsFinal: true,
	})

	log.Printf("Task %s completed with status %s", taskID, task.Status)
}

// GetStore returns the underlying store (for testing or advanced usage)
func (m *Manager) GetStore() Store {
	return m.store
}
