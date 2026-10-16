package configgen

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

func testInput() Input {
	return Input{
		Origin:      "https://shaker.example.com",
		ProjectID:   "11111111-1111-1111-1111-111111111111",
		ProjectName: "MyApp",
		AgentID:     "22222222-2222-2222-2222-222222222222",
		AgentName:   "Claude Code Backend",
		ToolType:    models.ToolClaudeCode,
	}
}

// plaintextKeyPattern matches a real credential shape.
//
// Anchored on a word boundary before "ask_", and with a body long enough that
// ordinary prose cannot match: a naive `strings.Contains(content, "ask_")` hits
// every document containing "task_", which is every generated instruction file.
var plaintextKeyPattern = regexp.MustCompile(`\bask_[A-Za-z0-9]{16,}`)

// TestRenderedFilesNeverCarryACredential is the invariant that makes a generated
// config safe to commit, and the reason this package exists on the server rather
// than in the browser.
//
// It checks more than "the placeholder is present". A renderer that emitted both
// the placeholder and a second Authorization header would pass that, and would
// still leak a key into a file the developer is about to push.
func TestRenderedFilesNeverCarryACredential(t *testing.T) {
	t.Parallel()

	for _, format := range models.AllConfigFormats {
		t.Run(string(format), func(t *testing.T) {
			t.Parallel()

			files, err := Render(testInput(), format)
			if err != nil {
				t.Fatalf("Render(%s): %v", format, err)
			}
			if len(files) == 0 {
				t.Fatalf("Render(%s) returned no files", format)
			}

			sawCredentialPlaceholder := false
			for _, f := range files {
				if f.Content == "" {
					t.Errorf("%s: %q is empty", format, fileLabel(f))
				}
				if plaintextKeyPattern.MatchString(f.Content) {
					t.Errorf("%s: %q contains what looks like a plaintext key", format, fileLabel(f))
				}
				// Whichever file carries the credential must name the variable
				// it is read from, and never inline the value. The instructions
				// document mentions the variable in prose rather than holding a
				// header, so it is checked separately below.
				if strings.Contains(f.Content, "Bearer "+APIKeyPlaceholder) {
					sawCredentialPlaceholder = true
					if strings.Count(f.Content, "Authorization") > 1 {
						t.Errorf("%s: %q has more than one Authorization header", format, fileLabel(f))
					}
				}
			}

			if !sawCredentialPlaceholder {
				t.Errorf("%s: no file carries the %s placeholder in a header", format, APIKeyPlaceholder)
			}
			if !strings.Contains(files[1].Content, APIKeyEnvVar) {
				t.Errorf("%s: the instructions never name %s", format, APIKeyEnvVar)
			}
		})
	}
}

// fileLabel identifies a file by its repository path, or by where it goes for a
// user-level one, so a failure message names something recognisable.
func fileLabel(f models.ConfigFile) string {
	if f.Filename != "" {
		return f.Filename
	}
	return f.Target
}

func TestMCPURLShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   Input
		want string
	}{
		{
			// Unchanged from the client-side generator this replaces: a config
			// rendered here has to work against a server that has not been
			// upgraded, and the URL is the part it actually talks to.
			name: "bound to project and agent",
			in:   testInput(),
			want: "https://shaker.example.com/mcp?agent_id=22222222-2222-2222-2222-222222222222&project_id=11111111-1111-1111-1111-111111111111",
		},
		{
			// agent_id is omitted rather than sent blank: "" is a different
			// value from absent on any server that resolves an empty id, and an
			// agent_id= with nothing after it is a config that looks bound and
			// is not.
			name: "unbound omits agent_id entirely",
			in: Input{
				Origin:    "https://shaker.example.com",
				ProjectID: "11111111-1111-1111-1111-111111111111",
			},
			want: "https://shaker.example.com/mcp?project_id=11111111-1111-1111-1111-111111111111",
		},
		{
			name: "no project and no agent",
			in:   Input{Origin: "https://shaker.example.com"},
			want: "https://shaker.example.com/mcp",
		},
		{
			// A relative URL is the honest output when no trustworthy origin
			// could be resolved — better than advertising a guessed host.
			name: "empty origin yields a relative endpoint",
			in:   Input{ProjectID: "11111111-1111-1111-1111-111111111111"},
			want: "/mcp?project_id=11111111-1111-1111-1111-111111111111",
		},
		{
			name: "trailing slash on origin does not double up",
			in:   Input{Origin: "https://shaker.example.com/", ProjectID: "p"},
			want: "https://shaker.example.com/mcp?project_id=p",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.in.MCPURL(); got != tc.want {
				t.Errorf("MCPURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestRenderedConfigIsValidJSON guards the most common way this generator could
// regress: emitting a template that a human reads correctly and every MCP host
// rejects.
func TestRenderedConfigIsValidJSON(t *testing.T) {
	t.Parallel()

	for _, format := range models.AllConfigFormats {
		t.Run(string(format), func(t *testing.T) {
			t.Parallel()

			files, err := Render(testInput(), format)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}

			var config map[string]any
			if err := json.Unmarshal([]byte(files[0].Content), &config); err != nil {
				t.Fatalf("rendered config is not valid JSON: %v\n%s", err, files[0].Content)
			}

			server, ok := config["mcpServers"].(map[string]any)
			if !ok {
				t.Fatalf("no mcpServers entry in %s", files[0].Content)
			}
			entry, ok := server[ServerName].(map[string]any)
			if !ok {
				t.Fatalf("no %q entry in %s", ServerName, files[0].Content)
			}
			if entry["type"] != "http" {
				t.Errorf("type = %v, want http", entry["type"])
			}
			url, _ := entry["url"].(string)
			if !strings.Contains(url, "/mcp?") {
				t.Errorf("url = %q, want the project-scoped /mcp endpoint", url)
			}
		})
	}
}

// TestFormatDefaultsToToolType is what makes the "Add Agent" wizard one step:
// the operator picks the tool once, at registration, and every later request
// renders the right layout without them naming it.
func TestFormatDefaultsToToolType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tool models.ToolType
		want models.ConfigFormat
	}{
		{models.ToolClaudeCode, models.ConfigFormatClaude},
		{models.ToolCodex, models.ConfigFormatCodex},
		{models.ToolCursor, models.ConfigFormatCursor},
		{models.ToolGenericMCP, models.ConfigFormatGeneric},
		// Phase 3 and 4 tools have no dedicated layout yet, and fall back to
		// the convention most hosts read rather than refusing.
		{models.ToolOpenCode, models.ConfigFormatGeneric},
		{models.ToolWindsurf, models.ConfigFormatGeneric},
		{models.ToolCopilotCLI, models.ConfigFormatGeneric},
		{models.ToolA2A, models.ConfigFormatGeneric},
		{models.ToolTypeUnknown, models.ConfigFormatGeneric},
	}

	for _, tc := range tests {
		t.Run(string(tc.tool), func(t *testing.T) {
			t.Parallel()
			in := testInput()
			in.ToolType = tc.tool
			got, err := in.Format("")
			if err != nil {
				t.Fatalf("Format(\"\"): %v", err)
			}
			if got != tc.want {
				t.Errorf("Format(\"\") = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestFormatRejectsAnUnknownValue(t *testing.T) {
	t.Parallel()

	in := testInput()
	_, err := in.Format("emacs")
	if err == nil {
		t.Fatal("Format(\"emacs\") = nil error, want a rejection")
	}
	// The message enumerates the options: the endpoint answers 400 with it, and
	// a client that cannot parse it can still show the operator a choice.
	if !strings.Contains(err.Error(), "claude") {
		t.Errorf("error %q does not name the supported formats", err)
	}
}

func TestExplicitFormatOverridesToolType(t *testing.T) {
	t.Parallel()

	in := testInput()
	got, err := in.Format("cursor")
	if err != nil {
		t.Fatalf("Format(\"cursor\"): %v", err)
	}
	if got != models.ConfigFormatCursor {
		t.Errorf("Format(\"cursor\") = %s, want cursor", got)
	}
}

// TestCodexConfigIsUserLevel documents a decision that looks like a bug: Codex
// has no per-project config file, so pointing it at a repository path would hand
// back a file the tool silently ignores.
func TestCodexConfigIsUserLevel(t *testing.T) {
	t.Parallel()

	files, err := Render(testInput(), models.ConfigFormatCodex)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if files[0].Filename != "" {
		t.Errorf("codex config has repository filename %q, want a user-level target", files[0].Filename)
	}
	if files[0].Target == "" {
		t.Error("codex config has no target, want ~/.codex/mcp.json")
	}
}

func TestInstructionsTellTheAgentToHeartbeat(t *testing.T) {
	t.Parallel()

	files, err := Render(testInput(), models.ConfigFormatClaude)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(files) < 2 {
		t.Fatalf("expected a config and an instructions file, got %d", len(files))
	}

	body := files[1].Content
	// The most common real-world failure is not a broken connection — it is a
	// working connection in which the agent never heartbeats and the reaper
	// reclaims its work. If the generated document stops saying so, the
	// behaviour goes with it.
	if !strings.Contains(body, "send_heartbeat") {
		t.Error("instructions never mention send_heartbeat")
	}
	if !strings.Contains(body, "lease") {
		t.Error("instructions never explain that the heartbeat is a lease")
	}
	if !strings.Contains(body, "MyApp") {
		t.Error("instructions do not name the project")
	}
	if !strings.Contains(body, "Claude Code Backend") {
		t.Error("instructions do not name the agent")
	}
}
