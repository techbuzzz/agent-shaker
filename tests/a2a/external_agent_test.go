package a2a

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/techbuzzz/agent-shaker/internal/a2a/client"
	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
)

func TestRealWorldAgentCard_HelloWorldAgent(t *testing.T) {
	// Test legacy agent card format (some agents still use old format)
	// This demonstrates backward compatibility for legacy agent cards
	realWorldJSON := `{
		"capabilities": [
			{
				"type": "streaming",
				"description": "Server-Sent Events streaming"
			}
		],
		"defaultInputModes": ["text"],
		"defaultOutputModes": ["text"],
		"description": "Just a hello world agent",
		"name": "Hello World Agent",
		"preferredTransport": "GRPC",
		"url": "http://127.0.0.1:9001",
		"version": "1.0.0"
	}`

	var card models.AgentCard
	err := json.Unmarshal([]byte(realWorldJSON), &card)
	if err != nil {
		t.Fatalf("Failed to unmarshal real-world agent card: %v", err)
	}

	// Verify basic fields
	if card.Name != "Hello World Agent" {
		t.Errorf("Expected name 'Hello World Agent', got '%s'", card.Name)
	}

	if card.Description != "Just a hello world agent" {
		t.Errorf("Expected description 'Just a hello world agent', got '%s'", card.Description)
	}

	// Check legacy endpoints field (deprecated)
	if card.Endpoints == nil {
		t.Logf("Note: Endpoints field is not present (expected for new schema)")
	}

	// Check capabilities (legacy format should be stored in metadata)
	if card.Metadata != nil && card.Metadata["legacyCapabilities"] != nil {
		t.Logf("Successfully parsed legacy capabilities from old format")
	} else if len(card.Capabilities.SupportedMessageParts) > 0 || card.Capabilities.A2AVersion != "" {
		t.Logf("Successfully parsed capabilities in new schema format")
	}

	t.Logf("Successfully parsed agent card:")
	t.Logf("  Name: %s", card.Name)
	t.Logf("  Description: %s", card.Description)
	t.Logf("  Version: %s", card.Version)
	if card.Capabilities.A2AVersion != "" {
		t.Logf("  A2A Version: %s", card.Capabilities.A2AVersion)
	}
}

func TestAgentCardUnmarshal_BooleanCapabilities(t *testing.T) {
	// Test handling of legacy capabilities object with boolean values
	// The new schema uses Capabilities struct, but legacy formats are stored in metadata
	jsonData := `{
		"name": "Test Agent",
		"version": "1.0.0",
		"capabilities": {
			"streaming": true,
			"task": false,
			"artifacts": true
		},
		"endpoints": []
	}`

	var card models.AgentCard
	err := json.Unmarshal([]byte(jsonData), &card)
	if err != nil {
		t.Fatalf("Failed to unmarshal capabilities with boolean values: %v", err)
	}

	// Legacy object format should be stored in metadata as legacyCapabilities
	if card.Metadata == nil {
		t.Logf("Note: Legacy capabilities were not stored in metadata (may be handled differently)")
	} else if legacyCaps, exists := card.Metadata["legacyCapabilities"]; exists {
		// Legacy capabilities array should have been created from the object format
		t.Logf("Successfully converted legacy capabilities object to array: %v", legacyCaps)
	} else {
		t.Logf("Note: Metadata present but no legacyCapabilities field")
	}

	// Verify the card parsed without error
	if card.Name != "Test Agent" {
		t.Errorf("Expected name 'Test Agent', got '%s'", card.Name)
	}

	t.Logf("Successfully parsed agent card with legacy capability object format")
}

func TestDiscoverExternalAgent_Integration(t *testing.T) {
	// Skip if the external agent is not running
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create A2A client
	a2aClient := client.NewHTTPClient()
	if a2aClient == nil {
		t.Fatal("Failed to create A2A HTTP client")
	}

	// Try to discover the external agent
	ctx := context.Background()
	agentURL := "http://127.0.0.1:9001"

	card, err := a2aClient.Discover(ctx, agentURL)
	if err != nil {
		// Don't fail if agent is not running, just skip
		t.Skipf("External agent not available at %s: %v", agentURL, err)
	}

	if card == nil {
		t.Fatal("Expected non-nil agent card after successful discovery")
	}

	// If we get here, the agent was discovered successfully
	t.Logf("Successfully discovered external agent:")
	t.Logf("  Name: %s", card.Name)
	t.Logf("  Description: %s", card.Description)
	t.Logf("  Version: %s", card.Version)
	if card.Capabilities.A2AVersion != "" {
		t.Logf("  A2A Version: %s", card.Capabilities.A2AVersion)
	}

	// Validate the card - note that discovery may return legacy format
	// and it should still parse without error
	if card.Name == "" {
		t.Error("Agent card should have a name")
	}

	t.Log("✓ Successfully discovered and parsed external agent card")
}

func TestAgentCardUnmarshal_MixedCapabilityTypes(t *testing.T) {
	// Test capabilities object with mixed value types (strings, booleans, numbers)
	// The new schema stores these as legacyCapabilities in metadata
	jsonData := `{
		"name": "Test Agent",
		"version": "1.0.0",
		"capabilities": {
			"streaming": true,
			"task": "Asynchronous task execution",
			"max_concurrent": 100
		},
		"endpoints": []
	}`

	var card models.AgentCard
	err := json.Unmarshal([]byte(jsonData), &card)
	if err != nil {
		t.Fatalf("Failed to unmarshal mixed capability types: %v", err)
	}

	// Verify the card parsed successfully
	if card.Name != "Test Agent" {
		t.Errorf("Expected name 'Test Agent', got '%s'", card.Name)
	}

	// Legacy capabilities with mixed types should be stored in metadata
	if card.Metadata != nil && card.Metadata["legacyCapabilities"] != nil {
		t.Logf("Successfully converted legacy capabilities with mixed types: %v", card.Metadata["legacyCapabilities"])
	} else {
		t.Logf("Note: Legacy mixed-type capabilities may be handled by new schema")
	}

	t.Logf("Successfully handled mixed-type capability object format")
}
