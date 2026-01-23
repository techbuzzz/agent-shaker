package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
)

// Discover fetches the agent card from an external A2A agent
func (c *HTTPClient) Discover(ctx context.Context, agentURL string) (*models.AgentCard, error) {
	url := fmt.Sprintf("%s/.well-known/agent-card.json", agentURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery failed: received status %d", resp.StatusCode)
	}

	var card models.AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, fmt.Errorf("failed to decode agent card: %w", err)
	}

	return &card, nil
}

// ValidateAgentCard checks if an agent card has required fields per official schema v1.0
func ValidateAgentCard(card *models.AgentCard) error {
	if card.SchemaVersion == "" {
		return fmt.Errorf("agent card missing required field: schemaVersion")
	}
	if card.HumanReadableID == "" {
		return fmt.Errorf("agent card missing required field: humanReadableId")
	}
	if card.AgentVersion == "" && card.Version == "" {
		return fmt.Errorf("agent card missing required field: agentVersion")
	}
	if card.Name == "" {
		return fmt.Errorf("agent card missing required field: name")
	}
	if card.Description == "" {
		return fmt.Errorf("agent card missing required field: description")
	}
	if card.URL == "" {
		return fmt.Errorf("agent card missing required field: url")
	}
	if card.Provider.Name == "" {
		return fmt.Errorf("agent card missing required field: provider.name")
	}
	if card.Capabilities.A2AVersion == "" {
		return fmt.Errorf("agent card missing required field: capabilities.a2aVersion")
	}
	if len(card.AuthSchemes) == 0 {
		return fmt.Errorf("agent card missing required field: authSchemes (must have at least one)")
	}
	return nil
}

// HasSkill checks if an agent has a specific skill by ID
func HasSkill(card *models.AgentCard, skillID string) bool {
	for _, skill := range card.Skills {
		if skill.ID == skillID {
			return true
		}
	}
	return false
}

// HasCapability checks for legacy capability format (for backward compatibility)
// Note: New agent cards should use HasSkill() instead
func HasCapability(card *models.AgentCard, capabilityType string) bool {
	// Check legacy capabilities stored in metadata
	if card.Metadata != nil {
		if legacyCaps, ok := card.Metadata["legacyCapabilities"].([]models.Capability); ok {
			for _, cap := range legacyCaps {
				if cap.Type == capabilityType {
					return true
				}
			}
		}
	}

	// For new schema, map common capability types to features
	switch capabilityType {
	case "streaming":
		return card.Capabilities.SupportsPushNotifications ||
			len(card.Capabilities.SupportedMessageParts) > 0
	case "task":
		return HasSkill(card, "task_execution")
	case "artifacts":
		return HasSkill(card, "artifact_sharing")
	case "mcp":
		return card.Capabilities.MCPVersion != ""
	}

	return false
}

// GetSkill finds a skill by ID
func GetSkill(card *models.AgentCard, skillID string) *models.Skill {
	for _, skill := range card.Skills {
		if skill.ID == skillID {
			return &skill
		}
	}
	return nil
}

// GetEndpoint finds an endpoint by path and method (legacy)
func GetEndpoint(card *models.AgentCard, path, method string) *models.Endpoint {
	for _, endpoint := range card.Endpoints {
		if endpoint.Path == path && endpoint.Method == method {
			return &endpoint
		}
	}
	return nil
}

// SupportsAuth checks if an agent supports a specific authentication scheme
func SupportsAuth(card *models.AgentCard, scheme string) bool {
	for _, auth := range card.AuthSchemes {
		if auth.Scheme == scheme {
			return true
		}
	}
	return false
}
