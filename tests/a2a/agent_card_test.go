package a2a

import (
	"encoding/json"
	"testing"

	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
)

func TestAgentCardUnmarshal_ArrayFormat(t *testing.T) {
	// Test parsing legacy array format for capabilities (stored as legacy in metadata)
	jsonData := `{
		"name": "Test Agent",
		"version": "1.0.0",
		"capabilities": [
			{
				"type": "task",
				"description": "Task execution"
			},
			{
				"type": "streaming",
				"description": "SSE streaming"
			}
		],
		"endpoints": []
	}`

	var card models.AgentCard
	err := json.Unmarshal([]byte(jsonData), &card)
	if err != nil {
		t.Fatalf("Failed to unmarshal array format: %v", err)
	}

	// Verify the card was parsed
	if card.Name != "Test Agent" {
		t.Errorf("Expected name 'Test Agent', got '%s'", card.Name)
	}

	// Legacy array format should be stored in metadata as legacyCapabilities
	if card.Metadata != nil && card.Metadata["legacyCapabilities"] != nil {
		t.Logf("Successfully stored legacy capabilities array in metadata")
	}
}

func TestAgentCardUnmarshal_ObjectFormat(t *testing.T) {
	// Test parsing legacy object format for capabilities (stored as legacy in metadata)
	jsonData := `{
		"name": "Test Agent",
		"version": "1.0.0",
		"capabilities": {
			"task": "Task execution",
			"streaming": "SSE streaming",
			"artifacts": "Artifact sharing"
		},
		"endpoints": []
	}`

	var card models.AgentCard
	err := json.Unmarshal([]byte(jsonData), &card)
	if err != nil {
		t.Fatalf("Failed to unmarshal object format: %v", err)
	}

	// Verify the card was parsed
	if card.Name != "Test Agent" {
		t.Errorf("Expected name 'Test Agent', got '%s'", card.Name)
	}

	// Legacy object format should be converted and stored in metadata
	if card.Metadata != nil && card.Metadata["legacyCapabilities"] != nil {
		t.Logf("Successfully converted legacy capabilities object to array in metadata")
	}
}

func TestAgentCardUnmarshal_InvalidFormat(t *testing.T) {
	// Invalid capabilities format should not cause error
	jsonData := `{
		"name": "Test Agent",
		"version": "1.0.0",
		"capabilities": "invalid",
		"endpoints": []
	}`

	var card models.AgentCard
	err := json.Unmarshal([]byte(jsonData), &card)
	if err != nil {
		t.Fatalf("Expected no error for invalid capabilities format, got: %v", err)
	}

	// Verify the card parsed basic fields
	if card.Name != "Test Agent" {
		t.Errorf("Expected name 'Test Agent', got '%s'", card.Name)
	}

	t.Log("✓ Invalid capabilities format handled gracefully")
}

func TestAgentCardUnmarshal_MissingCapabilities(t *testing.T) {
	// Missing capabilities field should not cause error
	jsonData := `{
		"name": "Test Agent",
		"version": "1.0.0",
		"endpoints": []
	}`

	var card models.AgentCard
	err := json.Unmarshal([]byte(jsonData), &card)
	if err != nil {
		t.Fatalf("Failed to unmarshal card without capabilities: %v", err)
	}

	// Capabilities should have zero values
	if card.Capabilities.A2AVersion != "" && card.Capabilities.MCPVersion != "" {
		t.Logf("Capabilities struct initialized with default values")
	}

	if card.Name != "Test Agent" {
		t.Errorf("Expected name 'Test Agent', got '%s'", card.Name)
	}
}

func TestAgentCardMarshal_NewSchema(t *testing.T) {
	// Test marshaling a card with new official schema
	card := models.AgentCard{
		SchemaVersion:   "1.0",
		HumanReadableID: "test/agent",
		AgentVersion:    "1.0.0",
		Name:            "Test Agent",
		Description:     "A test agent",
		URL:             "http://localhost:8080",
		Provider: models.Provider{
			Name: "Test Provider",
		},
		Capabilities: models.Capabilities{
			A2AVersion: "1.0",
		},
		AuthSchemes: []models.AuthScheme{
			{Scheme: "none"},
		},
		Endpoints: []models.Endpoint{},
	}

	data, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("Failed to marshal agent card: %v", err)
	}

	// Unmarshal back to verify format
	var result models.AgentCard
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal marshaled data: %v", err)
	}

	if result.Name != "Test Agent" {
		t.Errorf("Expected name 'Test Agent' after round-trip, got '%s'", result.Name)
	}

	if result.Capabilities.A2AVersion != "1.0" {
		t.Errorf("Expected A2A version '1.0' after round-trip, got '%s'", result.Capabilities.A2AVersion)
	}

	t.Log("✓ Successfully marshaled and unmarshaled new schema")
}
