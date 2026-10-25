package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// executeAnnounceCapability records what this agent's client says it can do.
//
// A client announces once, at setup. The server needs the answer before it
// routes work: an agent whose client cannot accept a pushed context, or author
// a compaction summary, is an agent the server must not ask to.
//
// The announcement belongs to the agent the connection is bound to, so a client
// can only ever describe itself — and a connection with no agent cannot
// describe anything, because there is nothing to describe it as.
func (h *MCPHandler) executeAnnounceCapability(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if ctx.AgentID == "" {
		return `{"error": "This connection is not bound to an agent. Add ?agent_id=UUID to the connection URL and announce again."}`, true
	}
	if h.agents == nil {
		return `{"error": "Database not connected"}`, true
	}

	agentID, err := uuid.Parse(ctx.AgentID)
	if err != nil {
		return `{"error": "This connection carries an agent id that is not a UUID."}`, true
	}

	names, ok := stringSliceArg(args, "capabilities")
	if !ok {
		return `{"error": "capabilities is required and must be a list of strings"}`, true
	}

	known := make([]string, 0, len(names))
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		if !models.IsValidAgentCapability(trimmed) {
			return fmt.Sprintf(
				`{"error": "unknown capability %q (known: %s)"}`,
				trimmed, models.JoinValues(models.AllAgentCapabilities),
			), true
		}
		known = append(known, trimmed)
	}
	if len(known) == 0 {
		return `{"error": "capabilities must name at least one capability"}`, true
	}

	caps := &models.AgentCapabilities{
		Capabilities:    known,
		Host:            optionalString(args, "host"),
		HostVersion:     optionalString(args, "host_version"),
		PluginVersion:   optionalString(args, "plugin_version"),
		ProtocolVersion: ctx.negotiatedVersion(),
		AnnouncedAt:     time.Now().UTC(),
	}

	if err := h.agents.SetCapabilities(ctx.requestContext(), agentID, caps); err != nil {
		return fmt.Sprintf(`{"error": "Could not record the announcement: %s"}`, err.Error()), true
	}

	result, _ := json.MarshalIndent(map[string]any{
		"success":          true,
		"agent_id":         agentID,
		"capabilities":     caps.Capabilities,
		"host":             caps.Host,
		"host_version":     caps.HostVersion,
		"plugin_version":   caps.PluginVersion,
		"protocol_version": caps.ProtocolVersion,
	}, "", "  ")

	return string(result), false
}

// stringSliceArg reads a list-of-strings argument.
//
// The type assertion is strict on purpose: an agent that sends the
// capabilities as a comma-separated string gets a clear refusal rather than a
// silent empty list, which would read as "this client supports nothing".
func stringSliceArg(args map[string]interface{}, name string) ([]string, bool) {
	raw, ok := args[name]
	if !ok {
		return nil, false
	}

	switch value := raw.(type) {
	case []string:
		return value, true
	case []interface{}:
		out := make([]string, 0, len(value))
		for _, item := range value {
			text, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, text)
		}
		return out, true
	default:
		return nil, false
	}
}
