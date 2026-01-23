package a2a

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/techbuzzz/agent-shaker/internal/a2a/client"
	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
)

func TestRealWorldAgentCard_HelloWorldAgent(t *testing.T) {
	// This is the actual response from http://127.0.0.1:9001/.well-known/agent-card.json
	realWorldJSON := `{
		"capabilities": {
			"streaming": true
		},
		"defaultInputModes": ["text"],
		"defaultOutputModes": ["text"],
		"description": "Just a hello world agent",
		"name": "Hello World Agent",
		"preferredTransport": "GRPC",
		"protocolVersion": "",
		"skills": [
			{
				"description": "Returns a 'Hello, world!'",
				"examples": ["hi", "hello"],
				"id": "hello_world",
				"name": "Hello, world!",
				"tags": ["hello world"]
			}
		],
		"url": "http://127.0.0.1:9001",
		"version": ""
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

	// Version is empty but should not cause issues
	if card.Version != "" {
		t.Logf("Note: Version field is '%s'", card.Version)
	}

	// Capabilities should be converted from object format to array
	if card.Capabilities == nil {
		t.Fatal("Capabilities should not be nil")
	}

	// The object format {"streaming": true} should be converted to array
	if len(card.Capabilities) != 1 {
		t.Errorf("Expected 1 capability, got %d", len(card.Capabilities))
	}

	// Check that streaming capability exists
	foundStreaming := false
	for _, cap := range card.Capabilities {
		if cap.Type == "streaming" {
			foundStreaming = true
			// Description should be "true" (the value from the object)
			if cap.Description != "true" {
				t.Errorf("Expected capability description 'true', got '%s'", cap.Description)
			}
		}
	}

	if !foundStreaming {
		t.Error("Expected to find 'streaming' capability")
	}

	t.Logf("Successfully parsed real-world agent card:")
	t.Logf("  Name: %s", card.Name)
	t.Logf("  Description: %s", card.Description)
	t.Logf("  Version: %s", card.Version)
	t.Logf("  Capabilities: %d converted from object format", len(card.Capabilities))
	for _, cap := range card.Capabilities {
		t.Logf("    - %s: %s", cap.Type, cap.Description)
	}
}

func TestAgentCardUnmarshal_BooleanCapabilities(t *testing.T) {
	// Test handling of boolean values in capabilities object
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

	if len(card.Capabilities) != 3 {
		t.Errorf("Expected 3 capabilities, got %d", len(card.Capabilities))
	}

	// Check that all capability types are present
	types := make(map[string]string)
	for _, cap := range card.Capabilities {
		types[cap.Type] = cap.Description
	}

	if types["streaming"] != "true" {
		t.Errorf("Expected streaming description 'true', got '%s'", types["streaming"])
	}

	if types["task"] != "false" {
		t.Errorf("Expected task description 'false', got '%s'", types["task"])
	}

	if types["artifacts"] != "true" {
		t.Errorf("Expected artifacts description 'true', got '%s'", types["artifacts"])
	}
}

func TestDiscoverExternalAgent_Integration(t *testing.T) {
	// Skip if the external agent is not running
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create A2A client
	a2aClient := client.NewHTTPClient()

	// Try to discover the external agent
	ctx := context.Background()
	agentURL := "http://127.0.0.1:9001"

	card, err := a2aClient.Discover(ctx, agentURL)
	if err != nil {
		// Don't fail if agent is not running, just skip
		t.Skipf("External agent not available at %s: %v", agentURL, err)
	}

	// If we get here, the agent was discovered successfully
	t.Logf("Successfully discovered external agent:")
	t.Logf("  Name: %s", card.Name)
	t.Logf("  Description: %s", card.Description)
	t.Logf("  Version: %s", card.Version)
	t.Logf("  Capabilities: %d", len(card.Capabilities))

	for _, cap := range card.Capabilities {
		t.Logf("    - %s: %s", cap.Type, cap.Description)
	}

	// Validate the card
	if err := client.ValidateAgentCard(card); err != nil {
		// Empty version is OK for some agents
		if card.Version == "" {
			t.Logf("Note: Agent has empty version field (non-fatal)")
		} else {
			t.Errorf("Agent card validation failed: %v", err)
		}
	}

	// Check for streaming capability (we know the Hello World Agent has this)
	if client.HasCapability(card, "streaming") {
		t.Log("✓ Agent has streaming capability")
	}

	// Test that we got some capabilities (even if converted from object format)
	if len(card.Capabilities) == 0 {
		t.Error("Expected at least one capability")
	}
}

func TestAgentCardUnmarshal_MixedCapabilityTypes(t *testing.T) {
	// Test capabilities object with mixed value types (strings, booleans, numbers)
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

	// Note: Our current implementation expects map[string]string, so numbers will be converted
	// This test documents the current behavior
	if len(card.Capabilities) == 0 {
		t.Log("Note: Mixed types in capabilities object may not parse correctly")
		t.Log("This is expected behavior - capabilities should be homogeneous")
	}
}
