package task

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
)

// Store defines the interface for task persistence
type Store interface {
	CreateTask(ctx context.Context, task *models.Task) error
	GetTask(ctx context.Context, taskID string) (*models.Task, error)
	UpdateTask(ctx context.Context, task *models.Task) error
	ListTasks(ctx context.Context, filter *Filter) ([]models.Task, error)
	DeleteTask(ctx context.Context, taskID string) error
}

// Filter defines options for filtering task lists
type Filter struct {
	Status string
	Limit  int
	Offset int
}

// MemoryStore implements Store using in-memory storage with optional file persistence
type MemoryStore struct {
	tasks    map[string]*models.Task
	mu       sync.RWMutex
	basePath string // optional: for file-based persistence
}

// NewMemoryStore creates a new in-memory task store
func NewMemoryStore(basePath string) *MemoryStore {
	store := &MemoryStore{
		tasks:    make(map[string]*models.Task),
		basePath: basePath,
	}

	// Load existing tasks from disk if basePath is provided
	if basePath != "" {
		store.loadFromDisk()
	}

	return store
}

// CreateTask stores a new task
func (s *MemoryStore) CreateTask(ctx context.Context, task *models.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("task %s already exists", task.ID)
	}

	s.tasks[task.ID] = task

	if s.basePath != "" {
		return s.saveTaskToDisk(task)
	}

	return nil
}

// GetTask retrieves a task by ID
func (s *MemoryStore) GetTask(ctx context.Context, taskID string) (*models.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	// Return a copy to prevent external modification
	taskCopy := *task
	return &taskCopy, nil
}

// UpdateTask updates an existing task
func (s *MemoryStore) UpdateTask(ctx context.Context, task *models.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[task.ID]; !exists {
		return fmt.Errorf("task %s not found", task.ID)
	}

	s.tasks[task.ID] = task

	if s.basePath != "" {
		return s.saveTaskToDisk(task)
	}

	return nil
}

// ListTasks returns tasks matching the filter criteria
func (s *MemoryStore) ListTasks(ctx context.Context, filter *Filter) ([]models.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tasks []models.Task

	for _, task := range s.tasks {
		// Apply status filter
		if filter != nil && filter.Status != "" {
			if string(task.Status) != filter.Status {
				continue
			}
		}
		tasks = append(tasks, *task)
	}

	// Apply offset and limit
	if filter != nil {
		if filter.Offset > 0 && filter.Offset < len(tasks) {
			tasks = tasks[filter.Offset:]
		} else if filter.Offset >= len(tasks) {
			return []models.Task{}, nil
		}

		if filter.Limit > 0 && filter.Limit < len(tasks) {
			tasks = tasks[:filter.Limit]
		}
	}

	return tasks, nil
}

// DeleteTask removes a task by ID
func (s *MemoryStore) DeleteTask(ctx context.Context, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[taskID]; !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	delete(s.tasks, taskID)

	if s.basePath != "" {
		return s.deleteTaskFromDisk(taskID)
	}

	return nil
}

// saveTaskToDisk persists a task to disk
func (s *MemoryStore) saveTaskToDisk(task *models.Task) error {
	if err := os.MkdirAll(s.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create tasks directory: %w", err)
	}

	filePath := filepath.Join(s.basePath, task.ID+".json")
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write task file: %w", err)
	}

	return nil
}

// deleteTaskFromDisk removes a task file from disk
func (s *MemoryStore) deleteTaskFromDisk(taskID string) error {
	filePath := filepath.Join(s.basePath, taskID+".json")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete task file: %w", err)
	}
	return nil
}

// loadFromDisk loads all tasks from disk
func (s *MemoryStore) loadFromDisk() {
	if s.basePath == "" {
		return
	}

	files, err := os.ReadDir(s.basePath)
	if err != nil {
		// Directory doesn't exist yet, that's fine
		return
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(s.basePath, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var task models.Task
		if err := json.Unmarshal(data, &task); err != nil {
			continue
		}

		s.tasks[task.ID] = &task
	}
}
