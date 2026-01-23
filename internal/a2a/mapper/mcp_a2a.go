package mapper

import (
	"fmt"
	"time"

	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
)

// ContextData represents context data from the existing system
type ContextData struct {
	ID          string
	Name        string
	Content     string
	Tags        []string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ContextToArtifact converts a context to an A2A artifact
func ContextToArtifact(ctx *ContextData, baseURL string) models.Artifact {
	return models.Artifact{
		ID:          ctx.ID,
		Name:        ctx.Name,
		Type:        "markdown",
		ContentType: "text/markdown",
		Content:     ctx.Content,
		URL:         fmt.Sprintf("%s/a2a/v1/artifacts/%s", baseURL, ctx.ID),
		Size:        int64(len(ctx.Content)),
		CreatedAt:   ctx.CreatedAt.Format(time.RFC3339),
		Metadata: map[string]any{
			"tags":        ctx.Tags,
			"description": ctx.Description,
			"updated_at":  ctx.UpdatedAt.Format(time.RFC3339),
		},
	}
}

// ArtifactToContext converts an A2A artifact back to context data
func ArtifactToContext(artifact *models.Artifact) *ContextData {
	ctx := &ContextData{
		ID:      artifact.ID,
		Name:    artifact.Name,
		Content: artifact.Content,
	}

	// Extract metadata
	if artifact.Metadata != nil {
		if tags, ok := artifact.Metadata["tags"].([]string); ok {
			ctx.Tags = tags
		}
		if desc, ok := artifact.Metadata["description"].(string); ok {
			ctx.Description = desc
		}
	}

	// Parse timestamps
	if artifact.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, artifact.CreatedAt); err == nil {
			ctx.CreatedAt = t
		}
	}

	if artifact.Metadata != nil {
		if updatedAt, ok := artifact.Metadata["updated_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
				ctx.UpdatedAt = t
			}
		}
	}

	return ctx
}

// TaskToA2ATask converts internal task models to A2A task format
func TaskToA2ATask(t interface{}, baseURL string) *models.Task {
	// This is a placeholder for converting between different task models
	// Implementation depends on the internal task model structure
	return nil
}
