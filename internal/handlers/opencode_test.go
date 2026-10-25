package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestOpenCodeManifestPublishesTheServerAndThePlugin checks the document an
// organisation hands to its whole fleet.
//
// OpenCode fetches this before any credential exists, so the document has to
// carry everything a client needs to reach the server — and nothing it should
// not carry.
func TestOpenCodeManifestPublishesTheServerAndThePlugin(t *testing.T) {
	t.Parallel()

	handler := NewOpenCodeManifestHandler("https://shaker.example.com")

	req := httptest.NewRequest(http.MethodGet, "/.well-known/opencode", nil)
	rec := httptest.NewRecorder()
	handler.Manifest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var manifest map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &manifest); err != nil {
		t.Fatalf("the manifest is not valid JSON: %v\n%s", err, rec.Body.String())
	}

	mcp, ok := manifest["mcp"].(map[string]any)
	if !ok {
		t.Fatalf("no `mcp` object in the manifest:\n%s", rec.Body.String())
	}
	servers, ok := mcp["servers"].(map[string]any)
	if !ok {
		t.Fatalf("no `mcp.servers` object in the manifest:\n%s", rec.Body.String())
	}
	entry, ok := servers["agent-shaker"].(map[string]any)
	if !ok {
		t.Fatalf("no agent-shaker server in the manifest:\n%s", rec.Body.String())
	}

	if url, _ := entry["url"].(string); url != "https://shaker.example.com/mcp" {
		t.Errorf("url = %q, want the origin's /mcp endpoint", url)
	}
	if oauth, ok := entry["oauth"]; !ok || oauth != false {
		t.Error("the manifest must turn OAuth off — this server authenticates with a header")
	}

	plugins, ok := manifest["plugins"].([]any)
	if !ok || len(plugins) == 0 {
		t.Fatalf("no `plugins` array in the manifest:\n%s", rec.Body.String())
	}
	plugin, ok := plugins[0].(map[string]any)
	if !ok {
		t.Fatalf("the plugin entry is not an object: %v", plugins[0])
	}
	if plugin["package"] == "" {
		t.Error("the plugin entry names no package")
	}
	options, ok := plugin["options"].(map[string]any)
	if !ok {
		t.Fatalf("the plugin entry carries no options: %v", plugin)
	}
	if endpoint, _ := options["endpoint"].(string); endpoint != "https://shaker.example.com/mcp" {
		t.Errorf("options.endpoint = %q, want the origin's /mcp endpoint", endpoint)
	}
}

// TestOpenCodeManifestCarriesNoSecret is the invariant that makes the
// manifest safe to serve from an unauthenticated path.
//
// It checks the shape, not the string: a document that carried the key
// anywhere at all would be a credential published on the open internet.
func TestOpenCodeManifestCarriesNoSecret(t *testing.T) {
	t.Parallel()

	handler := NewOpenCodeManifestHandler("https://shaker.example.com")

	req := httptest.NewRequest(http.MethodGet, "/.well-known/opencode", nil)
	rec := httptest.NewRecorder()
	handler.Manifest(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, "Bearer sk-") {
		t.Fatal("the manifest carries a credential")
	}
	if !strings.Contains(body, "{env:AGENT_SHAKER_API_KEY}") {
		t.Error("the manifest does not reference the key placeholder; it should never carry the key itself")
	}
}
