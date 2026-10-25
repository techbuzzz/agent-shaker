package configgen

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// parseOpenCodeConfig renders a format and parses its configuration file.
//
// The tests below parse rather than string-match on purpose: a document can be
// valid JSON, contain every key anyone expects to see, and still be a document
// the host silently ignores. Parsing is what catches a shape that moved.
func parseOpenCodeConfig(t *testing.T, format models.ConfigFormat) map[string]any {
	t.Helper()

	files, err := Render(testInput(), format)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(files[0].Content), &parsed); err != nil {
		t.Fatalf("rendered config is not valid JSON: %v\n%s", err, files[0].Content)
	}
	return parsed
}

// TestOpenCodeV2NestsTheServerUnderServers pins the shape OpenCode 2 reads.
//
// "V2 does not place server names directly under mcp" — so a document with the
// server directly under `mcp` is a document OpenCode 2 ignores, with no error
// to explain why the agent connected to nothing.
func TestOpenCodeV2NestsTheServerUnderServers(t *testing.T) {
	t.Parallel()

	parsed := parseOpenCodeConfig(t, models.ConfigFormatOpenCodeV2)

	mcp, ok := parsed["mcp"].(map[string]any)
	if !ok {
		t.Fatal("no `mcp` object in the rendered config")
	}
	if _, direct := mcp[ServerName]; direct {
		t.Error("the server sits directly under `mcp`; OpenCode 2 nests it under `mcp.servers`")
	}

	servers, ok := mcp["servers"].(map[string]any)
	if !ok {
		t.Fatalf("no `mcp.servers` object in the rendered config")
	}
	entry, ok := servers[ServerName].(map[string]any)
	if !ok {
		t.Fatalf("no %q entry under `mcp.servers`", ServerName)
	}

	if entry["type"] != "remote" {
		t.Errorf(`type = %v, want "remote"`, entry["type"])
	}
	if url, _ := entry["url"].(string); url != testInput().MCPURL() {
		t.Errorf("url = %q, want %q", url, testInput().MCPURL())
	}
	if _, wrong := entry["enabled"]; wrong {
		t.Error("the entry uses `enabled`; OpenCode 2 disables a server with `disabled`")
	}
	if disabled, ok := entry["disabled"].(bool); !ok || disabled {
		t.Errorf("disabled = %v, want false — a disabled server never connects", entry["disabled"])
	}
}

// TestOpenCodeV2TurnsOAuthOff is the sharpest edge in the whole layout.
//
// OpenCode 2 turns OAuth on for every remote server unless the file says
// `oauth: false`. This server authenticates with a header and does not speak
// OAuth, so without the field the client tries to authenticate against a
// server that cannot answer — and the agent connects to nothing.
func TestOpenCodeV2TurnsOAuthOff(t *testing.T) {
	t.Parallel()

	parsed := parseOpenCodeConfig(t, models.ConfigFormatOpenCodeV2)

	mcp := parsed["mcp"].(map[string]any)
	servers := mcp["servers"].(map[string]any)
	entry := servers[ServerName].(map[string]any)

	oauth, ok := entry["oauth"]
	if !ok {
		t.Fatal("no `oauth` field; OpenCode 2 turns OAuth on for remote servers by default")
	}
	if oauth != false {
		t.Errorf("oauth = %v, want false — this server authenticates with a header", oauth)
	}
}

// TestOpenCodeV2PinsTheProtocolRevision checks the revision the file asks for.
//
// `legacy` speaks the pre-2026 revisions, which is what this server serves.
// `2026-07-28` would fail against it, and `auto` would spend a probe on every
// connection to discover what `legacy` already says.
func TestOpenCodeV2PinsTheProtocolRevision(t *testing.T) {
	t.Parallel()

	parsed := parseOpenCodeConfig(t, models.ConfigFormatOpenCodeV2)

	mcp := parsed["mcp"].(map[string]any)
	servers := mcp["servers"].(map[string]any)
	entry := servers[ServerName].(map[string]any)

	if got, _ := entry["protocol"].(string); got != "legacy" {
		t.Errorf("protocol = %q, want \"legacy\" — the server does not speak 2026-07-28 yet", got)
	}
}

// TestOpenCodeV2InstallsThePluginWithItsEndpoint checks the v2 plugin entry.
//
// OpenCode 2 accepts an options object, and it is the only place the endpoint
// can go.
func TestOpenCodeV2InstallsThePluginWithItsEndpoint(t *testing.T) {
	t.Parallel()

	parsed := parseOpenCodeConfig(t, models.ConfigFormatOpenCodeV2)

	plugins, ok := parsed["plugins"].([]any)
	if !ok || len(plugins) == 0 {
		t.Fatal("no `plugins` array in the rendered config")
	}
	entry, ok := plugins[0].(map[string]any)
	if !ok {
		t.Fatalf("the plugin entry is not an object: %v", plugins[0])
	}
	if entry["package"] != OpenCodePluginPackage {
		t.Errorf("package = %v, want %q", entry["package"], OpenCodePluginPackage)
	}

	options, ok := entry["options"].(map[string]any)
	if !ok {
		t.Fatalf("the plugin entry carries no options: %v", entry)
	}
	if endpoint, _ := options["endpoint"].(string); endpoint != testInput().MCPURL() {
		t.Errorf("options.endpoint = %q, want %q", endpoint, testInput().MCPURL())
	}
}

// TestOpenCodeV1InstallsThePluginByName checks the v1 plugin entry.
//
// OpenCode 1 reads a list of bare names and has no options object, so the
// endpoint has to reach the plugin through the environment instead.
func TestOpenCodeV1InstallsThePluginByName(t *testing.T) {
	t.Parallel()

	files, err := Render(testInput(), models.ConfigFormatOpenCode)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	var v1 map[string]any
	if err := json.Unmarshal([]byte(files[0].Content), &v1); err != nil {
		t.Fatalf("rendered config is not valid JSON: %v", err)
	}

	plugins, ok := v1["plugin"].([]any)
	if !ok || len(plugins) != 1 {
		t.Fatalf("no `plugin` array with one entry in the rendered config")
	}
	if plugins[0] != OpenCodePluginPackage {
		t.Errorf("plugin = %v, want %q", plugins[0], OpenCodePluginPackage)
	}
	if _, wrong := v1["plugins"]; wrong {
		t.Error("the v1 document carries `plugins`; OpenCode 1 reads `plugin`")
	}
}

// TestTheTwoOpenCodeDocumentsAreNotInterchangeable states the reason there are
// two formats rather than one.
//
// `mcp` is one key with two shapes, so a single file cannot serve both majors:
// the v1 reader looks for the server directly under `mcp` and the v2 reader
// looks under `mcp.servers`. Whichever one is missing, the agent connects to
// nothing.
func TestTheTwoOpenCodeDocumentsAreNotInterchangeable(t *testing.T) {
	t.Parallel()

	v1 := parseOpenCodeConfig(t, models.ConfigFormatOpenCode)
	v2 := parseOpenCodeConfig(t, models.ConfigFormatOpenCodeV2)

	v1MCP := v1["mcp"].(map[string]any)
	if _, nested := v1MCP["servers"]; nested {
		t.Error("the v1 document nests its server; OpenCode 1 expects it directly under `mcp`")
	}
	if _, direct := v1MCP[ServerName]; !direct {
		t.Error("the v1 document does not carry the server directly under `mcp`")
	}

	v2MCP := v2["mcp"].(map[string]any)
	if _, direct := v2MCP[ServerName]; direct {
		t.Error("the v2 document carries the server directly under `mcp`; OpenCode 2 nests it")
	}
	if _, nested := v2MCP["servers"]; !nested {
		t.Error("the v2 document does not nest its server under `mcp.servers`")
	}
}

// TestOpenCodeV1SetupDocNamesTheEndpointVariable covers the gap the v1 layout
// leaves.
//
// OpenCode 1 has nowhere in the configuration to put the plugin's endpoint, so
// the setup document has to say which variable carries it. A document that
// names the key but not the endpoint produces a plugin that loads and then
// cannot reach the server.
func TestOpenCodeV1SetupDocNamesTheEndpointVariable(t *testing.T) {
	t.Parallel()

	files, err := Render(testInput(), models.ConfigFormatOpenCode)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	var setup string
	for _, file := range files {
		if file.Filename == SetupDocFilename {
			setup = file.Content
		}
	}
	if setup == "" {
		t.Fatalf("no %s in the rendered bundle", SetupDocFilename)
	}
	if !strings.Contains(setup, OpenCodeEndpointEnvVar) {
		t.Errorf("the setup document never mentions %s; the v1 plugin has no other way to learn the endpoint", OpenCodeEndpointEnvVar)
	}
}

// TestOpenCodeV2SetupDocDoesNotNeedTheEndpointVariable is the mirror: the v2
// layout carries the endpoint in the configuration, so telling the reader to
// export a second variable would be advice they do not need.
func TestOpenCodeV2SetupDocDoesNotNeedTheEndpointVariable(t *testing.T) {
	t.Parallel()

	files, err := Render(testInput(), models.ConfigFormatOpenCodeV2)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	var setup string
	for _, file := range files {
		if file.Filename == SetupDocFilename {
			setup = file.Content
		}
	}
	if setup == "" {
		t.Fatalf("no %s in the rendered bundle", SetupDocFilename)
	}
	if strings.Contains(setup, "export "+OpenCodeEndpointEnvVar) {
		t.Errorf("the setup document asks for %s; the v2 layout carries the endpoint in the configuration", OpenCodeEndpointEnvVar)
	}
}
