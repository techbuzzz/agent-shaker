package a2a

import (
	"encoding/json"
	"testing"

	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
)

func TestAgentCardUnmarshal_ArrayFormat(t *testing.T) {
	// Standard format with capabilities as an array
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

	if len(card.Capabilities) != 2 {
		t.Errorf("Expected 2 capabilities, got %d", len(card.Capabilities))
	}

	if card.Capabilities[0].Type != "task" {
		t.Errorf("Expected first capability type 'task', got '%s'", card.Capabilities[0].Type)
	}

	if card.Capabilities[1].Type != "streaming" {
		t.Errorf("Expected second capability type 'streaming', got '%s'", card.Capabilities[1].Type)
	}
}

func TestAgentCardUnmarshal_ObjectFormat(t *testing.T) {
	// Alternative format with capabilities as an object (some A2A implementations)
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

	if len(card.Capabilities) != 3 {
		t.Errorf("Expected 3 capabilities, got %d", len(card.Capabilities))
	}

	// Check that all capability types are present (order may vary due to map iteration)
	types := make(map[string]bool)
	for _, cap := range card.Capabilities {
		types[cap.Type] = true
	}

	expectedTypes := []string{"task", "streaming", "artifacts"}
	for _, expectedType := range expectedTypes {
		if !types[expectedType] {
			t.Errorf("Expected capability type '%s' not found", expectedType)
		}
	}
}

func TestAgentCardUnmarshal_InvalidFormat(t *testing.T) {
	// Invalid capabilities format should not cause error, just empty array
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

	if len(card.Capabilities) != 0 {
		t.Errorf("Expected 0 capabilities for invalid format, got %d", len(card.Capabilities))
	}
}

func TestAgentCardUnmarshal_MissingCapabilities(t *testing.T) {
	// Missing capabilities field should result in empty array
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

	if card.Capabilities == nil {
		t.Error("Expected non-nil capabilities slice")
	}

	if len(card.Capabilities) != 0 {
		t.Errorf("Expected 0 capabilities, got %d", len(card.Capabilities))
	}
}

func TestAgentCardMarshal(t *testing.T) {
	// Ensure marshaling always produces array format (standard)
	card := models.AgentCard{
		Name:    "Test Agent",
		Version: "1.0.0",
		Capabilities: []models.Capability{
			{Type: "task", Description: "Task execution"},
			{Type: "streaming", Description: "SSE streaming"},
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

	if len(result.Capabilities) != 2 {
		t.Errorf("Expected 2 capabilities after round-trip, got %d", len(result.Capabilities))
	}
}
