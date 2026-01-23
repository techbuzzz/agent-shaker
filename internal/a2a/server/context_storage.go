package server

import (
	"fmt"

	"github.com/techbuzzz/agent-shaker/internal/database"
)

// DatabaseContextStorage adapts the database to the ContextStorage interface
type DatabaseContextStorage struct {
	db *database.DB
}

// NewDatabaseContextStorage creates a new database context storage adapter
func NewDatabaseContextStorage(db *database.DB) *DatabaseContextStorage {
	return &DatabaseContextStorage{db: db}
}

// ListContexts retrieves all contexts from the database
func (s *DatabaseContextStorage) ListContexts() ([]ContextData, error) {
	if s.db == nil {
		return []ContextData{}, nil
	}

	rows, err := s.db.Query(`
		SELECT id, title, content, tags, created_at, updated_at
		FROM contexts
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query contexts: %w", err)
	}
	defer rows.Close()

	var contexts []ContextData
	for rows.Next() {
		var ctx ContextData
		var tags []byte

		if err := rows.Scan(&ctx.ID, &ctx.Name, &ctx.Content, &tags, &ctx.CreatedAt, &ctx.UpdatedAt); err != nil {
			continue
		}

		// Parse tags if present
		if len(tags) > 0 {
			// PostgreSQL array format
			ctx.Tags = parsePostgresArray(string(tags))
		}

		contexts = append(contexts, ctx)
	}

	return contexts, nil
}

// GetContext retrieves a specific context by ID
func (s *DatabaseContextStorage) GetContext(id string) (*ContextData, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var ctx ContextData
	var tags []byte

	err := s.db.QueryRow(`
		SELECT id, title, content, tags, created_at, updated_at
		FROM contexts
		WHERE id = $1
	`, id).Scan(&ctx.ID, &ctx.Name, &ctx.Content, &tags, &ctx.CreatedAt, &ctx.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("context not found: %w", err)
	}

	// Parse tags if present
	if len(tags) > 0 {
		ctx.Tags = parsePostgresArray(string(tags))
	}

	return &ctx, nil
}

// parsePostgresArray parses PostgreSQL array format to slice of strings
func parsePostgresArray(s string) []string {
	// Handle empty array
	if s == "{}" || s == "" {
		return []string{}
	}

	// Remove braces
	s = s[1 : len(s)-1]
	if s == "" {
		return []string{}
	}

	// Split by comma
	var result []string
	var current string
	inQuote := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			inQuote = !inQuote
		case ',':
			if !inQuote {
				result = append(result, current)
				current = ""
				continue
			}
			fallthrough
		default:
			current += string(c)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}

// InMemoryContextStorage provides an in-memory implementation for testing
type InMemoryContextStorage struct {
	contexts map[string]*ContextData
}

// NewInMemoryContextStorage creates a new in-memory context storage
func NewInMemoryContextStorage() *InMemoryContextStorage {
	return &InMemoryContextStorage{
		contexts: make(map[string]*ContextData),
	}
}

// AddContext adds a context to the in-memory storage
func (s *InMemoryContextStorage) AddContext(ctx *ContextData) {
	s.contexts[ctx.ID] = ctx
}

// ListContexts returns all contexts
func (s *InMemoryContextStorage) ListContexts() ([]ContextData, error) {
	contexts := make([]ContextData, 0, len(s.contexts))
	for _, ctx := range s.contexts {
		contexts = append(contexts, *ctx)
	}
	return contexts, nil
}

// GetContext returns a specific context by ID
func (s *InMemoryContextStorage) GetContext(id string) (*ContextData, error) {
	ctx, exists := s.contexts[id]
	if !exists {
		return nil, fmt.Errorf("context %s not found", id)
	}
	return ctx, nil
}
