package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/techbuzzz/agent-shaker/internal/a2a/client"
	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
)

func main() {
	fmt.Println("Testing A2A Agent Discovery with External Agent")
	fmt.Println("=================================================")
	fmt.Println()

	// Create A2A client
	a2aClient := client.NewHTTPClient()
	ctx := context.Background()

	// Test 1: Discover Agent Shaker itself
	fmt.Println("1. Discovering Agent Shaker (localhost:8080)...")
	agentShakerURL := "http://localhost:8080"
	card1, err := a2aClient.Discover(ctx, agentShakerURL)
	if err != nil {
		log.Printf("   ❌ Failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ Success!\n")
		fmt.Printf("   Name: %s\n", card1.Name)
		fmt.Printf("   Agent Version: %s\n", card1.AgentVersion)
		fmt.Printf("   Human ID: %s\n", card1.HumanReadableID)
		fmt.Printf("   A2A Version: %s\n", card1.Capabilities.A2AVersion)
		if card1.Capabilities.MCPVersion != "" {
			fmt.Printf("   MCP Version: %s\n", card1.Capabilities.MCPVersion)
		}
		fmt.Printf("   Skills: %d\n", len(card1.Skills))
		for _, skill := range card1.Skills {
			fmt.Printf("     - %s: %s\n", skill.ID, skill.Name)
		}
	}

	fmt.Println()

	// Test 2: Discover external Hello World Agent
	fmt.Println("2. Discovering Hello World Agent (127.0.0.1:9001)...")
	externalURL := "http://127.0.0.1:9001"
	card2, err := a2aClient.Discover(ctx, externalURL)
	if err != nil {
		log.Printf("   ❌ Failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ Success!\n")
		fmt.Printf("   Name: %s\n", card2.Name)
		fmt.Printf("   Description: %s\n", card2.Description)
		fmt.Printf("   Agent Version: %s\n", card2.AgentVersion)
		fmt.Printf("   Human ID: %s\n", card2.HumanReadableID)

		// Check for legacy capabilities
		if card2.Metadata != nil {
			if legacyCaps, ok := card2.Metadata["legacyCapabilities"].([]models.Capability); ok {
				fmt.Printf("   Legacy Capabilities: %d (converted from object format)\n", len(legacyCaps))
				for _, cap := range legacyCaps {
					fmt.Printf("     - %s: %s\n", cap.Type, cap.Description)
				}
			}
		}

		// Show skills if present
		if len(card2.Skills) > 0 {
			fmt.Printf("   Skills: %d\n", len(card2.Skills))
			for _, skill := range card2.Skills {
				fmt.Printf("     - %s: %s\n", skill.ID, skill.Name)
			}
		}

		// Show raw JSON
		fmt.Println("\n   Normalized Agent Card (as JSON):")
		jsonBytes, _ := json.MarshalIndent(card2, "   ", "  ")
		fmt.Printf("   %s\n", string(jsonBytes))

		// Validate
		if err := client.ValidateAgentCard(card2); err != nil {
			fmt.Printf("   ⚠️  Validation warning: %v\n", err)
		} else {
			fmt.Printf("   ✓ Agent card is valid per official schema v1.0\n")
		}

		// Check capabilities
		if client.HasCapability(card2, "streaming") {
			fmt.Printf("   ✓ Agent supports streaming\n")
		}
	}

	fmt.Println("\n=================================================")
	fmt.Println("All tests completed!")
}
