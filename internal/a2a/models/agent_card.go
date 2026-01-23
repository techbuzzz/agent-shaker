package models

import (
	"encoding/json"
	"fmt"
)

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

// UnmarshalJSON implements custom JSON unmarshaling to handle both array and object formats
// for capabilities field (for compatibility with different A2A implementations)
func (a *AgentCard) UnmarshalJSON(data []byte) error {
	// First try to unmarshal into a temporary struct with capabilities as interface{}
	type Alias AgentCard
	aux := &struct {
		CapabilitiesRaw json.RawMessage `json:"capabilities"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Try to unmarshal capabilities as array first (standard format)
	var capArray []Capability
	if err := json.Unmarshal(aux.CapabilitiesRaw, &capArray); err == nil {
		a.Capabilities = capArray
		return nil
	}

	// If array format fails, try object format (map of capability type to description/value)
	// This handles both string values and other types (bool, number) by converting to interface{}
	var capMap map[string]interface{}
	if err := json.Unmarshal(aux.CapabilitiesRaw, &capMap); err == nil {
		// Convert map to array of Capability structs
		a.Capabilities = make([]Capability, 0, len(capMap))
		for capType, value := range capMap {
			// Convert any value to string representation
			var description string
			switch v := value.(type) {
			case string:
				description = v
			case bool:
				if v {
					description = "true"
				} else {
					description = "false"
				}
			case float64:
				description = fmt.Sprintf("%v", v)
			default:
				// For any other type, marshal back to JSON string
				if jsonBytes, err := json.Marshal(v); err == nil {
					description = string(jsonBytes)
				} else {
					description = ""
				}
			}

			a.Capabilities = append(a.Capabilities, Capability{
				Type:        capType,
				Description: description,
			})
		}
		return nil
	}

	// If both formats fail, return empty capabilities array (non-fatal)
	a.Capabilities = []Capability{}
	return nil
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
