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

// ValidateAgentCard checks if an agent card has required fields
func ValidateAgentCard(card *models.AgentCard) error {
	if card.Name == "" {
		return fmt.Errorf("agent card missing required field: name")
	}
	if card.Version == "" {
		return fmt.Errorf("agent card missing required field: version")
	}
	return nil
}

// HasCapability checks if an agent has a specific capability
func HasCapability(card *models.AgentCard, capabilityType string) bool {
	for _, cap := range card.Capabilities {
		if cap.Type == capabilityType {
			return true
		}
	}
	return false
}

// GetEndpoint finds an endpoint by path and method
func GetEndpoint(card *models.AgentCard, path, method string) *models.Endpoint {
	for _, endpoint := range card.Endpoints {
		if endpoint.Path == path && endpoint.Method == method {
			return &endpoint
		}
	}
	return nil
}
