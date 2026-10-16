package mcp

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// Protocol revision negotiation.
//
// The server speaks 2024-11-05 by default and accepts the revisions between it
// and the 2026-07-28 stateless core. It does not implement that core: 2026-07-28
// retires the initialize handshake and the Mcp-Session-Id header, so adopting it
// is a transport rewrite rather than a version bump, and it would break every
// agent host deployed against this server today. Serving the older revisions
// honestly, and refusing the ones we cannot honour, is the behaviour that keeps
// an existing agent fleet working.
//
// Revisions in order: 2024-11-05, 2025-03-26, 2025-06-18, 2025-11-25,
// 2026-07-28. The first four still carry the session-based lifecycle this
// server implements.

// mcpProtocolVersion is the revision this server reports and defaults to.
const mcpProtocolVersion = "2024-11-05"

// supportedProtocolVersions is every revision this server will accept, oldest
// first. Anything outside this set is refused with a machine-readable code and
// this list, so a client can discover the answer without reading prose.
var supportedProtocolVersions = []string{
	"2024-11-05",
	"2025-03-26",
	"2025-06-18",
	"2025-11-25",
}

// errUnsupportedVersion is the JSON-RPC error code sent for a version
// mismatch.
//
// -32000..-32099 is the implementation-defined range: the MCP specification
// does not assign this code itself, it only requires a refusal that names the
// versions the server accepts. -32022 is what deployments of this revision
// (AWS AgentCore Gateway among them) use for exactly this case, so it is
// followed here for interoperability rather than invented.
//
// Note that -32022 is also LSP's ContentModified. The ranges are per-protocol,
// so there is no collision in practice.
const errUnsupportedVersion = -32022

// errIdentityMismatch is the JSON-RPC error code sent when a credential bound
// to one agent is used with a different agent_id.
//
// Also in the implementation-defined range, and deliberately different from
// errUnsupportedVersion: a client that receives this has a *credential*
// problem, not a protocol one, and retrying with another revision will not help.
// The message names the agent the key is bound to so an operator reading a log
// can see which credential was used, and the correction — drop the id from the
// URL — is in the message rather than left to be guessed.
const errIdentityMismatch = -32001

// protocolVersionHeader carries the client's chosen revision on Streamable
// HTTP. Version selection is per request, not per connection, because the
// stateless revisions dropped connection state.
const protocolVersionHeader = "MCP-Protocol-Version"

// DefaultProtocolVersion returns the revision this server reports when a
// client has not negotiated one.
//
// Exported so the A2A agent card publishes the same string the MCP surface
// serves. The card used to hardcode "0.6", which is not a revision of this
// protocol at all — a peer reading it had no way to tell what it would get.
func DefaultProtocolVersion() string { return mcpProtocolVersion }

// SupportedProtocolVersions returns a copy of the accepted revision set, so a
// caller cannot mutate the package-level slice.
func SupportedProtocolVersions() []string {
	out := make([]string, len(supportedProtocolVersions))
	copy(out, supportedProtocolVersions)
	return out
}

// supportsProtocolVersion reports whether v is in the accepted set.
func supportsProtocolVersion(v string) bool {
	for _, supported := range supportedProtocolVersions {
		if supported == v {
			return true
		}
	}
	return false
}

// negotiateProtocolVersion resolves the revision for one request.
//
// A missing header is not an error: hosts predating the header simply omit it,
// and defaulting is what keeps them working. An unrecognised value is an error,
// because silently downgrading a client that asked for 2026-07-28 would leave it
// waiting for stateless semantics this server does not have.
func negotiateProtocolVersion(r *http.Request) (string, error) {
	requested := r.Header.Get(protocolVersionHeader)
	if requested == "" {
		return mcpProtocolVersion, nil
	}
	if !supportsProtocolVersion(requested) {
		return "", fmt.Errorf("protocol version %q is not supported", requested)
	}
	return requested, nil
}

// writeUnsupportedVersion answers a request that named an unusable revision.
//
// HTTP 400 plus the specification's -32022 code and the supported list: the
// status lets a gateway and a client react without parsing JSON, and the body
// tells a capable client exactly what to retry with.
func writeUnsupportedVersion(w http.ResponseWriter, requested string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"error": map[string]any{
			"code":    errUnsupportedVersion,
			"message": fmt.Sprintf("protocol version %q is not supported", requested),
			"data": map[string]any{
				"requested":         requested,
				"supportedVersions": supportedProtocolVersions,
			},
		},
	})
}

// writeIdentityMismatch answers a request whose credential does not name the
// agent it claimed to be.
//
// HTTP 403 rather than 401: the credential presented was valid and was
// accepted for authentication. What it is not allowed to do is act as that
// agent, and a 401 would tell the client to go and fetch a different key — the
// wrong advice, since re-authenticating with the same key produces the same
// refusal.
func writeIdentityMismatch(w http.ResponseWriter, r *http.Request, cause error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"error": map[string]any{
			"code":    errIdentityMismatch,
			"message": cause.Error(),
		},
	})
}

// logNegotiation records the resolved revision, which is the only place an
// operator can see which revision a given client fleet is actually on.
func logNegotiation(ctx MCPContext, version string) {
	slog.InfoContext(ctx.requestContext(), "mcp protocol version negotiated",
		"protocol_version", version,
		"project_id", ctx.ProjectID,
		"agent_id", ctx.AgentID,
	)
}
