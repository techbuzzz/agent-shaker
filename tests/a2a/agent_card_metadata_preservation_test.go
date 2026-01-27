package a2a

import (
	"encoding/json"
	"testing"

	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
)

func TestAgentCardUnmarshal_MetadataPreservation(t *testing.T) {
	// Test that legacy capabilities are preserved in metadata after struct assignment
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
		"endpoints": [],
		"metadata": {
			"existingKey": "existingValue"
		}
	}`

	var card models.AgentCard
	err := json.Unmarshal([]byte(jsonData), &card)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Check if metadata exists
	if card.Metadata == nil {
		t.Fatal("Metadata is nil")
	}

	// Check if legacy capabilities are preserved
	legacyCaps, ok := card.Metadata["legacyCapabilities"]
	if !ok {
		t.Fatal("legacyCapabilities not found in metadata - the fix failed!")
	}

	caps, ok := legacyCaps.([]models.Capability)
	if !ok {
		t.Fatalf("legacyCapabilities is not []Capability, got: %T", legacyCaps)
	}

	if len(caps) != 2 {
		t.Fatalf("Expected 2 legacy capabilities, got %d", len(caps))
	}

	if caps[0].Type != "task" || caps[0].Description != "Task execution" {
		t.Errorf("First capability incorrect: %+v", caps[0])
	}

	if caps[1].Type != "streaming" || caps[1].Description != "SSE streaming" {
		t.Errorf("Second capability incorrect: %+v", caps[1])
	}

	// Check if existing metadata is also preserved
	if existingVal, ok := card.Metadata["existingKey"].(string); !ok || existingVal != "existingValue" {
		t.Errorf("Existing metadata not preserved correctly: %v", card.Metadata["existingKey"])
	}

	t.Log("✓ Legacy capabilities correctly preserved in metadata after struct assignment")
}

func TestAgentCardUnmarshal_ObjectFormat_MetadataPreservation(t *testing.T) {
	// Test that legacy object format capabilities are also preserved
	jsonData := `{
		"name": "Test Agent",
		"version": "1.0.0",
		"capabilities": {
			"task": "Task execution",
			"streaming": "SSE streaming",
			"artifacts": "Artifact sharing"
		},
		"endpoints": [],
		"metadata": {
			"customField": "customValue"
		}
	}`

	var card models.AgentCard
	err := json.Unmarshal([]byte(jsonData), &card)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Check if metadata exists
	if card.Metadata == nil {
		t.Fatal("Metadata is nil")
	}

	// Check if legacy capabilities are preserved
	legacyCaps, ok := card.Metadata["legacyCapabilities"]
	if !ok {
		t.Fatal("legacyCapabilities not found in metadata - the fix failed!")
	}

	caps, ok := legacyCaps.([]models.Capability)
	if !ok {
		t.Fatalf("legacyCapabilities is not []Capability, got: %T", legacyCaps)
	}

	if len(caps) != 3 {
		t.Fatalf("Expected 3 legacy capabilities, got %d", len(caps))
	}

	// Check if existing metadata is also preserved
	if customVal, ok := card.Metadata["customField"].(string); !ok || customVal != "customValue" {
		t.Errorf("Existing metadata not preserved correctly: %v", card.Metadata["customField"])
	}

	t.Log("✓ Legacy object format capabilities correctly preserved in metadata")
}
