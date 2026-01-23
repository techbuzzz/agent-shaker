package models

// AgentCard represents the A2A agent discovery card
// published at /.well-known/agent-card.json
type AgentCard struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Version      string         `json:"version"`
	Capabilities []Capability   `json:"capabilities"`
	Endpoints    []Endpoint     `json:"endpoints"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// Capability describes a specific capability of the agent
type Capability struct {
	Type        string `json:"type"` // e.g., "task", "streaming", "artifacts"
	Description string `json:"description"`
}

// Endpoint describes an API endpoint provided by the agent
type Endpoint struct {
	Path        string            `json:"path"`
	Method      string            `json:"method"`
	Description string            `json:"description"`
	Protocol    string            `json:"protocol"` // "A2A", "MCP"
	Params      map[string]string `json:"params,omitempty"`
}
