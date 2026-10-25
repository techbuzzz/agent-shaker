package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

// Idempotency keys on the mutating MCP tools.
//
// The 2026-07-28 revision of the MCP specification makes retries ordinary
// traffic: a broken response stream loses the in-flight request and the
// client is required to re-issue it, and a multi-round-trip request retries
// the original request. Both are documented behaviours of a conforming
// client, not error paths.
//
// The failure they would otherwise cause is not a failed request. It is a
// task claimed twice, or a task completed twice — a state the server then
// reports as truth, and one the client cannot tell apart from the truth.
// So the server absorbs the retry: the first request stores its answer under
// the key, and a repeat of the same request replays that answer instead of
// computing a second one.
//
// The key is optional. A client that does not send one gets the old
// behaviour, because the alternative — requiring one — would break every
// existing agent on the day the revision landed.
//
// A key identifies one logical operation, and the request's fingerprint is
// part of the identity. A client that reuses a key for a *different* request
// gets a different row and a real execution, rather than a replay of the
// first answer: being told "claimed" for a task that was never claimed is
// worse than the duplicate the key was meant to prevent.

// idempotencyKey reads the optional idempotency_key argument.
func idempotencyKey(args map[string]interface{}) string {
	raw, ok := args["idempotency_key"]
	if !ok {
		return ""
	}

	key, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(key)
}

// replayIdempotent returns the stored answer for a request.
//
// The second return value says whether there was one. A store read that
// fails is treated as "there was none": the request proceeds and the write it
// was about to make is the one that matters.
func (h *MCPHandler) replayIdempotent(ctx MCPContext, tool string, args map[string]interface{}) (string, bool) {
	key := idempotencyKey(args)
	if key == "" || h.db == nil {
		return "", false
	}

	agentID := parseUUID(ctx.AgentID)
	if agentID == uuid.Nil {
		return "", false
	}

	var response []byte
	err := h.db.QueryRow(
		`SELECT response FROM idempotency_keys
		 WHERE tool = $1 AND agent_id = $2 AND key = $3 AND request_hash = $4`,
		tool, agentID, key, requestHash(tool, args),
	).Scan(&response)
	if err != nil {
		return "", false
	}

	return string(response), true
}

// recordIdempotent stores an answer for a request.
//
// A conflict is not an error: the first writer won, and its answer is the
// one that will be replayed. Recording is best-effort — a failure here means
// the next retry computes the answer again, which is the old behaviour, not
// a broken one.
func (h *MCPHandler) recordIdempotent(ctx MCPContext, tool string, args map[string]interface{}, response string) {
	key := idempotencyKey(args)
	if key == "" || h.db == nil {
		return
	}

	agentID := parseUUID(ctx.AgentID)
	if agentID == uuid.Nil {
		return
	}

	_, _ = h.db.Exec(
		`INSERT INTO idempotency_keys (tool, agent_id, key, request_hash, response)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (tool, agent_id, key, request_hash) DO NOTHING`,
		tool, agentID, key, requestHash(tool, args), response,
	)
}

// requestHash is the fingerprint of a request, without the key itself.
//
// The tool name is part of it so that the same key on two different tools is
// two different operations, and the arguments are part of it so that the same
// key on two different tasks is two different operations too.
func requestHash(tool string, args map[string]interface{}) string {
	raw, err := json.Marshal(struct {
		Tool string                 `json:"tool"`
		Args map[string]interface{} `json:"args"`
	}{Tool: tool, Args: argsWithoutKey(args)})
	if err != nil {
		// A request that cannot be encoded is a request that cannot be
		// fingerprinted; the tool name alone still separates the tools.
		raw = []byte(tool)
	}

	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// argsWithoutKey copies the arguments with the idempotency key removed.
//
// The key identifies the request, so it cannot also be part of the request's
// fingerprint — otherwise no two attempts would ever agree.
func argsWithoutKey(args map[string]interface{}) map[string]interface{} {
	if args == nil {
		return map[string]interface{}{}
	}

	copied := make(map[string]interface{}, len(args))
	for name, value := range args {
		if name == "idempotency_key" {
			continue
		}
		copied[name] = value
	}
	return copied
}
