package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/techbuzzz/agent-shaker/internal/a2a/client"
)

func main() {
	fmt.Println("Testing A2A Agent Discovery with External Agent")
	fmt.Println("=================================================\n")

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
		fmt.Printf("   Version: %s\n", card1.Version)
		fmt.Printf("   Capabilities: %d\n", len(card1.Capabilities))
		for _, cap := range card1.Capabilities {
			fmt.Printf("     - %s: %s\n", cap.Type, cap.Description)
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
		fmt.Printf("   Version: %s\n", card2.Version)
		fmt.Printf("   Capabilities: %d (converted from object format)\n", len(card2.Capabilities))
		for _, cap := range card2.Capabilities {
			fmt.Printf("     - %s: %s\n", cap.Type, cap.Description)
		}

		// Show raw JSON
		fmt.Println("\n   Normalized Agent Card (as JSON):")
		jsonBytes, _ := json.MarshalIndent(card2, "   ", "  ")
		fmt.Printf("   %s\n", string(jsonBytes))

		// Validate
		if err := client.ValidateAgentCard(card2); err != nil && card2.Version != "" {
			fmt.Printf("   ⚠️  Validation warning: %v\n", err)
		} else if card2.Version == "" {
			fmt.Printf("   ℹ️  Note: Empty version field (non-fatal, agent still usable)\n")
		}

		// Check capabilities
		if client.HasCapability(card2, "streaming") {
			fmt.Printf("   ✓ Agent supports streaming\n")
		}
	}

	fmt.Println("\n=================================================")
	fmt.Println("All tests completed!")
}
