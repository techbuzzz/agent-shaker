package models

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ToolType is which client an agent runs in.
//
// It exists so the config generator can answer "where does this file go"
// without a human answering the question a second time, and so the roster can
// be filtered by tool. It is deliberately a client identifier and nothing
// more: what an agent is capable of is a separate concern (Phase 3's
// announce_capability), and a value here never grants an MCP tool.
//
// The set is closed, and the database enforces the same list in
// migrations/017_federation_identity.up.sql. Go stays the single source of
// truth so the validation message can enumerate what is accepted instead of
// surfacing a constraint name.
type ToolType string

const (
	ToolClaudeCode  ToolType = "claude-code"
	ToolCodex       ToolType = "codex"
	ToolOpenCode    ToolType = "opencode"
	ToolCopilotCLI  ToolType = "copilot-cli"
	ToolCopilotVS   ToolType = "copilot-vs"
	ToolCursor      ToolType = "cursor"
	ToolWindsurf    ToolType = "windsurf"
	ToolA2A         ToolType = "a2a"
	ToolGenericMCP  ToolType = "generic-mcp"
	ToolTypeUnknown ToolType = "unknown"
)

// AllToolTypes is every tool type the system understands, in a stable order.
// Exported so the management UI, the API validator and the database constraint
// can be checked against one list rather than three hand-maintained ones.
var AllToolTypes = []ToolType{
	ToolClaudeCode,
	ToolCodex,
	ToolOpenCode,
	ToolCopilotCLI,
	ToolCopilotVS,
	ToolCursor,
	ToolWindsurf,
	ToolA2A,
	ToolGenericMCP,
	ToolTypeUnknown,
}

// ErrUnknownToolType is returned by ParseToolType for a value outside
// AllToolTypes. A distinct sentinel so a handler can tell a bad vocabulary
// value from an arbitrary malformed field.
var ErrUnknownToolType = errors.New("unknown tool type")

// ParseToolType normalises a requested tool type.
//
// Empty is accepted and maps to ToolTypeUnknown rather than being an error: an
// agent created by an MCP runtime, or by an API client that predates this
// field, does not know the concept, and refusing to create it would make the
// field a breaking change instead of an addition. "I do not know" is a true
// answer, and the config endpoint renders the generic format for it.
func ParseToolType(s string) (ToolType, error) {
	candidate := ToolType(strings.ToLower(strings.TrimSpace(s)))
	if candidate == "" {
		return ToolTypeUnknown, nil
	}
	for _, valid := range AllToolTypes {
		if valid == candidate {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%w: %q (accepted: %s)", ErrUnknownToolType, s, JoinValues(AllToolTypes))
}

// IsValidToolType reports whether s names a tool type the system understands.
// Unlike ParseToolType it does not treat the empty string as unknown, because
// the callers that ask are checking a value that is already stored.
func IsValidToolType(s string) bool {
	for _, valid := range AllToolTypes {
		if string(valid) == s {
			return true
		}
	}
	return false
}

// SupportsConfigGeneration reports whether the config endpoint can render a
// file layout for this tool type.
//
// A false result is deliberate: windsurf and a2a are Phase 3 and Phase 4
// work, and an a2a agent is not an MCP host at all. Every false entry still
// gets a usable generic `.mcp.json`, which is what most hosts read —
// rendering a plausible-looking file for a client that has no layout here
// would be worse than being honest about it.
func (t ToolType) SupportsConfigGeneration() bool {
	switch t {
	case ToolClaudeCode, ToolCodex, ToolOpenCode, ToolCursor, ToolCopilotVS, ToolCopilotCLI, ToolGenericMCP:
		return true
	default:
		return false
	}
}

// DefaultFormat is the config format rendered when the caller does not ask for
// one, chosen from the agent's tool type. An agent whose tool type has no
// dedicated layout still gets a usable file: the generic `.mcp.json` is what
// the majority of MCP hosts read.
func (t ToolType) DefaultFormat() ConfigFormat {
	switch t {
	case ToolClaudeCode:
		return ConfigFormatClaude
	case ToolCodex:
		return ConfigFormatCodex
	case ToolOpenCode:
		return ConfigFormatOpenCode
	case ToolCursor:
		return ConfigFormatCursor
	case ToolCopilotVS:
		return ConfigFormatVSCode
	case ToolCopilotCLI:
		return ConfigFormatCopilot
	default:
		return ConfigFormatGeneric
	}
}

// ConfigFormat is one rendered file layout.
type ConfigFormat string

const (
	ConfigFormatClaude   ConfigFormat = "claude"
	ConfigFormatCodex    ConfigFormat = "codex"
	ConfigFormatCursor   ConfigFormat = "cursor"
	ConfigFormatGeneric  ConfigFormat = "generic"
	ConfigFormatOpenCode ConfigFormat = "opencode"
	// ConfigFormatOpenCodeV2 is the OpenCode 2.x layout.
	//
	// It is a separate format rather than a variation of the OpenCode one
	// because the two majors read different documents: 2 nests the servers
	// under mcp.servers, renames the disable flag, turns OAuth on for remote
	// servers unless told otherwise, and takes a different key for plugins.
	// One file cannot be both — `mcp` is one key — so the format has to be
	// chosen by the caller rather than guessed.
	ConfigFormatOpenCodeV2 ConfigFormat = "opencode-v2"
	ConfigFormatVSCode     ConfigFormat = "vscode"
	ConfigFormatCopilot    ConfigFormat = "copilot"
)

// AllConfigFormats is every format the server can render today, in the order
// the endpoint reports them. The same list appears in the 400 a caller gets
// for an unsupported format, so a client can enumerate options from the error.
var AllConfigFormats = []ConfigFormat{
	ConfigFormatClaude,
	ConfigFormatCodex,
	ConfigFormatCursor,
	ConfigFormatOpenCode,
	ConfigFormatOpenCodeV2,
	ConfigFormatVSCode,
	ConfigFormatCopilot,
	ConfigFormatGeneric,
}

// ParseConfigFormat normalises a requested format. Unlike ParseToolType there
// is no empty case: "no format" is not a request the endpoint can act on, and
// silently substituting one would hand back a layout the caller did not ask
// for.
func ParseConfigFormat(s string) (ConfigFormat, error) {
	candidate := ConfigFormat(strings.ToLower(strings.TrimSpace(s)))
	for _, valid := range AllConfigFormats {
		if valid == candidate {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%w: %q (supported: %s)", ErrUnknownToolType, s, JoinValues(AllConfigFormats))
}

// ConfigFile is one rendered configuration file.
//
// It lives here rather than in internal/configgen because it is the wire shape
// the HTTP layer serialises: the generator builds values of this type, the
// handlers hand them back, and web/app/types transcribes them. Defining it in
// the generator would make the transport contract depend on an implementation
// package, and the two shapes would drift into a third.
type ConfigFile struct {
	// Format is the layout this file belongs to.
	Format ConfigFormat `json:"format"`

	// Filename is the path relative to the repository root — `.claude/mcp.json`,
	// `.mcp.json`. Files that belong outside a repository carry an empty
	// Filename and a Target instead.
	Filename string `json:"filename"`

	// Target is where a user-level file goes, for example `~/.codex/mcp.json`.
	// Empty for repository-relative files.
	Target string `json:"target,omitempty"`

	// Description is one line explaining what the file is for, shown in the UI
	// so the operator does not have to know the convention.
	Description string `json:"description"`

	// Content is the file body. Any credential in it is the
	// ${env:AGENT_SHAKER_API_KEY} placeholder — never a plaintext key. That
	// invariant is what makes a generated file safe to commit, and
	// configgen's tests hold it in place.
	Content string `json:"content"`

	// Install is the line a user runs to put the file in place, empty when the
	// file is simply written into the repository.
	Install string `json:"install,omitempty"`
}

// AgentConnectionRequest is the POST /api/agents/{id}/connection body.
//
// Deliberately almost empty. The agent comes from the path, the project comes
// from the agent row, and the scopes are fixed at read+write by the handler —
// an endpoint whose only job is "connect this agent" should not be a way to
// mint an arbitrary credential with arbitrary authority.
type AgentConnectionRequest struct {
	// Format asks for a specific layout. Empty means "the one that fits this
	// agent's tool type".
	Format string `json:"format"`
}

// APIKey is a row of the api_keys table.
//
// KeyHash is tagged `json:"-"` and that tag is load-bearing: it is the only
// thing standing between a hash and every response that serialises a key. The
// value is useless to an attacker on its own, but a hash in an access log or a
// browser history is a credential waiting for a policy mistake.
type APIKey struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	KeyHash     string     `json:"-" db:"key_hash"`
	KeyPrefix   string     `json:"key_prefix" db:"key_prefix"`
	PrincipalID uuid.UUID  `json:"principal_id" db:"principal_id"`
	ProjectID   *uuid.UUID `json:"project_id,omitempty" db:"project_id"`
	Scopes      []Scope    `json:"scopes" db:"scopes"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`

	// AgentID binds the credential to one agent (migration 017). A non-nil
	// value is authoritative on the MCP surface: the agent_id in the connection
	// URL is then either the same agent or a rejected mismatch, never a way to
	// act as somebody else.
	//
	// NULL means unbound, and that is the common case rather than the exception
	// — every key issued before this column existed is NULL, as is the legacy
	// shared-secret path. Those callers keep supplying agent_id in the URL and
	// keep being trusted, because that is the only identity they have.
	AgentID *uuid.UUID `json:"agent_id,omitempty" db:"agent_id"`

	// PrincipalType and PrincipalName are denormalised onto the row for
	// display. PrincipalType exists in the table because the request path must
	// not join people to learn whether the caller is human; PrincipalName does
	// not, and is filled in by the list query's join with people.
	PrincipalType PersonKind `json:"principal_type" db:"principal_type"`
	PrincipalName string     `json:"principal_name,omitempty" db:"principal_name"`
}

// Active reports whether the key may still be used.
func (k APIKey) Active() bool { return k.RevokedAt == nil }

// Allows is the APIKey-side view of the scope rule.
func (k APIKey) Allows(want Scope) bool { return Allows(k.Scopes, want) }

// CreateAPIKeyRequest is the POST /api/keys body.
//
// PrincipalType is not accepted from the client: it is copied from the people
// row so a caller cannot mint a key that claims to belong to a human while
// pointing at a service.
type CreateAPIKeyRequest struct {
	PrincipalID string   `json:"principal_id"`
	ProjectID   string   `json:"project_id"`
	Scopes      []string `json:"scopes"`

	// AgentID binds the minted key to one agent. Optional, and validated
	// against the agents table when present — a key pointing at a row that does
	// not exist would resolve to an identity no tool could act as.
	AgentID string `json:"agent_id"`
}

// IssuedAPIKey is the only response shape in this package that carries a
// plaintext secret, and it does so exactly once: the key is never readable
// again from the API, because the server does not keep it.
type IssuedAPIKey struct {
	APIKey

	// Secret is the plaintext key. The handler that returns it must also set
	// Cache-Control: no-store; see internal/handlers/api_keys.go.
	Secret string `json:"secret"`
}

// SortToolTypes orders a set for storage or display. Used where a UI list has
// to be deterministic without importing the declaration order.
func SortToolTypes(in []ToolType) {
	sort.Slice(in, func(i, j int) bool { return in[i] < in[j] })
}
