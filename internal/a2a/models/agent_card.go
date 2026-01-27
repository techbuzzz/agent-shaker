package models

import (
	"encoding/json"
	"fmt"
)

// AgentCard represents the A2A agent discovery card following the official schema v1.0
// published at /.well-known/agent-card.json
// See: https://gist.githubusercontent.com/SecureAgentTools/0815a2de9cc31c71468afd3d2eef260a/raw/agent-card-schema.md
type AgentCard struct {
	// Required fields
	SchemaVersion   string       `json:"schemaVersion"`   // Version of Agent Card schema (e.g., "1.0")
	HumanReadableID string       `json:"humanReadableId"` // Unique identifier (e.g., "org/agent-name")
	AgentVersion    string       `json:"agentVersion"`    // Version of agent software (semantic versioning)
	Name            string       `json:"name"`            // Human-readable display name
	Description     string       `json:"description"`     // Detailed description
	URL             string       `json:"url"`             // Primary A2A endpoint URL
	Provider        Provider     `json:"provider"`        // Provider information
	Capabilities    Capabilities `json:"capabilities"`    // Protocol capabilities
	AuthSchemes     []AuthScheme `json:"authSchemes"`     // Supported authentication schemes (min 1)

	// Optional fields
	Skills            []Skill  `json:"skills,omitempty"`            // List of agent skills
	Tags              []string `json:"tags,omitempty"`              // Keywords for discovery
	PrivacyPolicyURL  string   `json:"privacyPolicyUrl,omitempty"`  // Privacy policy URL
	TermsOfServiceURL string   `json:"termsOfServiceUrl,omitempty"` // Terms of service URL
	IconURL           string   `json:"iconUrl,omitempty"`           // Icon URL
	LastUpdated       string   `json:"lastUpdated,omitempty"`       // ISO 8601 timestamp

	// Legacy fields for backward compatibility (deprecated)
	Version   string         `json:"version,omitempty"`   // Deprecated: use AgentVersion
	Endpoints []Endpoint     `json:"endpoints,omitempty"` // Deprecated: use Skills
	Metadata  map[string]any `json:"metadata,omitempty"`  // Deprecated: use specific fields
}

// Provider information about the agent provider/developer
type Provider struct {
	Name           string `json:"name"`                      // Required: Provider name
	URL            string `json:"url,omitempty"`             // Optional: Provider homepage
	SupportContact string `json:"support_contact,omitempty"` // Optional: Support contact
}

// Capabilities describes protocol capabilities and features
type Capabilities struct {
	A2AVersion                string      `json:"a2aVersion"`                          // Required: A2A protocol version
	MCPVersion                string      `json:"mcpVersion,omitempty"`                // Optional: MCP protocol version
	SupportedMessageParts     []string    `json:"supportedMessageParts,omitempty"`     // Optional: Message part types
	SupportsPushNotifications bool        `json:"supportsPushNotifications,omitempty"` // Optional: Push notification support
	TEEDetails                *TEEDetails `json:"teeDetails,omitempty"`                // Optional: TEE information
}

// TEEDetails describes Trusted Execution Environment setup
type TEEDetails struct {
	Type                string `json:"type"`                          // Required: TEE technology (e.g., "Intel SGX")
	AttestationEndpoint string `json:"attestationEndpoint,omitempty"` // Optional: Attestation verification URL
	PublicKey           string `json:"publicKey,omitempty"`           // Optional: Public key for secure communication
	Description         string `json:"description,omitempty"`         // Optional: Human-readable TEE description
}

// AuthScheme describes an authentication method
type AuthScheme struct {
	Scheme            string   `json:"scheme"`                       // Required: "apiKey", "oauth2", "bearer", "none"
	Description       string   `json:"description,omitempty"`        // Optional: Human-readable description
	TokenURL          string   `json:"tokenUrl,omitempty"`           // Required for oauth2: Token endpoint
	Scopes            []string `json:"scopes,omitempty"`             // Optional for oauth2: Required scopes
	ServiceIdentifier string   `json:"service_identifier,omitempty"` // Optional: Identifier for key managers
}

// Skill describes a specific capability/function the agent can perform
type Skill struct {
	ID           string                 `json:"id"`                      // Required: Unique skill identifier
	Name         string                 `json:"name"`                    // Required: Human-readable skill name
	Description  string                 `json:"description"`             // Required: Detailed description
	InputSchema  map[string]interface{} `json:"input_schema,omitempty"`  // Optional: JSON Schema for input
	OutputSchema map[string]interface{} `json:"output_schema,omitempty"` // Optional: JSON Schema for output
}

// Endpoint describes an API endpoint (legacy, kept for backward compatibility)
type Endpoint struct {
	Path        string            `json:"path"`
	Method      string            `json:"method"`
	Description string            `json:"description"`
	Protocol    string            `json:"protocol"` // "A2A", "MCP"
	Params      map[string]string `json:"params,omitempty"`
}

// Capability describes a generic capability (legacy, kept for backward compatibility)
type Capability struct {
	Type        string `json:"type"` // e.g., "task", "streaming", "artifacts"
	Description string `json:"description"`
}

// UnmarshalJSON implements custom JSON unmarshaling to handle both the new official schema
// and legacy formats for backward compatibility
func (a *AgentCard) UnmarshalJSON(data []byte) error {
	// Define an alias to prevent recursion
	type Alias AgentCard

	// First, parse the raw JSON to handle legacy capabilities formats
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Check if capabilities is present and try legacy formats first
	var hasLegacyCap bool
	if capRaw, exists := raw["capabilities"]; exists {
		// Try legacy array format []Capability
		var legacyCaps []Capability
		if err := json.Unmarshal(capRaw, &legacyCaps); err == nil && len(legacyCaps) > 0 {
			// Convert legacy array to new Capabilities object structure
			// Store in metadata for backward compatibility
			if a.Metadata == nil {
				a.Metadata = make(map[string]any)
			}
			a.Metadata["legacyCapabilities"] = legacyCaps
			hasLegacyCap = true
		} else if err == nil {
			// Empty array, still legacy format
			hasLegacyCap = true
		} else {
			// Try legacy object format map[string]string or map[string]interface{}
			var legacyCapMap map[string]interface{}
			if err := json.Unmarshal(capRaw, &legacyCapMap); err == nil && len(legacyCapMap) > 0 {
				// Check if it looks like legacy format (has keys like "streaming": true)
				// vs new format (has keys like "a2aVersion")
				if _, hasA2A := legacyCapMap["a2aVersion"]; !hasA2A {
					// This is legacy object format, convert it
					legacyCaps := make([]Capability, 0, len(legacyCapMap))
					for capType, value := range legacyCapMap {
						var description string
						switch v := value.(type) {
						case string:
							description = v
						case bool:
							description = fmt.Sprintf("%v", v)
						case float64:
							description = fmt.Sprintf("%v", v)
						default:
							if jsonBytes, err := json.Marshal(v); err == nil {
								description = string(jsonBytes)
							}
						}
						legacyCaps = append(legacyCaps, Capability{
							Type:        capType,
							Description: description,
						})
					}
					if a.Metadata == nil {
						a.Metadata = make(map[string]any)
					}
					a.Metadata["legacyCapabilities"] = legacyCaps
					hasLegacyCap = true
				}
			} else {
				// Even if capabilities is invalid (e.g. plain string), mark as legacy to skip new format parsing
				hasLegacyCap = true
			}
		}

		// If we detected legacy format, remove it from raw so alias unmarshal doesn't fail
		if hasLegacyCap {
			delete(raw, "capabilities")
		}
	}

	// Reconstruct JSON without problematic capabilities field if it was legacy format
	if hasLegacyCap {
		cleanedData, err := json.Marshal(raw)
		if err != nil {
			return err
		}
		// Unmarshal cleaned data into alias
		aux := &Alias{}
		if err := json.Unmarshal(cleanedData, aux); err != nil {
			return err
		}
		*a = AgentCard(*aux)
	} else {
		// New schema format, unmarshal normally
		aux := &Alias{}
		if err := json.Unmarshal(data, aux); err != nil {
			return err
		}
		*a = AgentCard(*aux)
	}

	return nil
}
