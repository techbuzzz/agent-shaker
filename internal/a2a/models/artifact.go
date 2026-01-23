package models

// Artifact represents an A2A artifact (e.g., markdown context, file)
type Artifact struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"` // "markdown", "json", "binary"
	ContentType string         `json:"content_type"`
	Content     string         `json:"content,omitempty"`
	URL         string         `json:"url,omitempty"`
	Size        int64          `json:"size"`
	CreatedAt   string         `json:"created_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ArtifactListResponse represents the response for listing artifacts
type ArtifactListResponse struct {
	Artifacts  []Artifact `json:"artifacts"`
	TotalCount int        `json:"total"`
}
