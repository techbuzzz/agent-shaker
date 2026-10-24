package configgen

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// TestVSCodeConfigUsesServersKey pins the difference that makes the file work.
//
// VS Code reads `servers`; everything else here reads `mcpServers`. The two are
// not interchangeable and the failure is silent: a file with the wrong
// top-level key is valid JSON that VS Code ignores without a word, so the
// agent connects to nothing and the cause is not obvious from the file.
func TestVSCodeConfigUsesServersKey(t *testing.T) {
	t.Parallel()

	files, err := Render(testInput(), models.ConfigFormatVSCode)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(files[0].Content), &parsed); err != nil {
		t.Fatalf("rendered config is not valid JSON: %v\n%s", err, files[0].Content)
	}
	if _, wrong := parsed["mcpServers"]; wrong {
		t.Error("config uses mcpServers; VS Code reads `servers`")
	}
	servers, ok := parsed["servers"].(map[string]any)
	if !ok {
		t.Fatalf("no `servers` object in %s", files[0].Content)
	}
	entry, ok := servers[ServerName].(map[string]any)
	if !ok {
		t.Fatalf("no %q entry in %s", ServerName, files[0].Content)
	}
	if entry["type"] != "http" {
		t.Errorf("type = %v, want http", entry["type"])
	}
	if files[0].Filename != ".vscode/mcp.json" {
		t.Errorf("filename = %q, want .vscode/mcp.json", files[0].Filename)
	}
}

// TestCopilotFormatShipsThePairCopilotNeeds pins what Copilot actually reads.
//
// Copilot in VS Code reads its MCP servers from .vscode/mcp.json, and reads
// repository instructions from .github/copilot-instructions.md without being
// asked. The old client-side bundle shipped exactly those two together, and
// moving the generator to the server dropped both.
//
// The pair matters: a Copilot user who received only the instructions would
// have a bundle that connects to nothing.
func TestCopilotFormatShipsThePairCopilotNeeds(t *testing.T) {
	t.Parallel()

	files, err := Render(testInput(), models.ConfigFormatCopilot)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	byName := map[string]models.ConfigFile{}
	for _, f := range files {
		byName[f.Filename] = f
	}

	config, ok := byName[".vscode/mcp.json"]
	if !ok {
		t.Fatalf("no .vscode/mcp.json in the bundle: %v", files)
	}
	instructions, ok := byName[".github/copilot-instructions.md"]
	if !ok {
		t.Fatalf("no .github/copilot-instructions.md in the bundle: %v", files)
	}

	// The instructions have to be instructions, not a config.
	if strings.HasPrefix(strings.TrimSpace(instructions.Content), "{") {
		t.Error("the Copilot instructions file contains JSON, not instructions")
	}
	if !strings.Contains(instructions.Content, "get_my_identity") {
		t.Error("the instructions do not tell the agent how to orient itself")
	}

	// And the config has to be VS Code's shape, or VS Code ignores it.
	var parsed map[string]any
	if err := json.Unmarshal([]byte(config.Content), &parsed); err != nil {
		t.Fatalf("the Copilot bundle's config is not valid JSON: %v", err)
	}
	if _, wrong := parsed["mcpServers"]; wrong {
		t.Error("the Copilot bundle's config uses mcpServers; VS Code reads `servers`")
	}
	if _, ok := parsed["servers"]; !ok {
		t.Error("the Copilot bundle's config has no `servers` object")
	}
}

// TestEveryBundleCarriesASetupDoc pins the document that was lost with the
// client-side generator.
//
// It is what answers the two questions a developer has when a generated bundle
// does not work: which file goes where, and how do I check that the connection
// is alive. Without it, every one of those questions turns into a support
// question.
func TestEveryBundleCarriesASetupDoc(t *testing.T) {
	t.Parallel()

	for _, format := range models.AllConfigFormats {
		t.Run(string(format), func(t *testing.T) {
			t.Parallel()

			files, err := Render(testInput(), format)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}

			var setup *models.ConfigFile
			for i := range files {
				if files[i].Filename == SetupDocFilename {
					setup = &files[i]
				}
			}
			if setup == nil {
				t.Fatalf("no %s in the bundle: %v", SetupDocFilename, files)
			}

			// It has to name the variable, the endpoint and the way to check --
			// those are the three things a reader needs and none of them are
			// obvious from the config file alone.
			for _, want := range []string{APIKeyEnvVar, "initialize", "curl"} {
				if !strings.Contains(setup.Content, want) {
					t.Errorf("the setup document never mentions %q", want)
				}
			}
		})
	}
}

// TestSetupDocNamesTheFilesItShipsWith keeps the document honest about the
// bundle. A setup document that lists files the bundle does not contain is
// worse than none: it sends the reader looking for something that is not there.
func TestSetupDocNamesTheFilesItShipsWith(t *testing.T) {
	t.Parallel()

	for _, format := range models.AllConfigFormats {
		t.Run(string(format), func(t *testing.T) {
			t.Parallel()

			files, err := Render(testInput(), format)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}

			var setup string
			for _, f := range files {
				if f.Filename == SetupDocFilename {
					setup = f.Content
				}
			}

			for _, f := range files {
				if f.Filename == SetupDocFilename {
					continue
				}
				if !strings.Contains(setup, f.Filename) {
					t.Errorf("the setup document never mentions %q, which the bundle contains", f.Filename)
				}
			}
		})
	}
}

// TestSetupDocDoesNotTeachTheWrongVariableSpelling checks the setup document
// does not tell a reader to use the wrong substitution.
//
// The substitution spelling differs per client -- ${env:VAR} for most, {env:VAR}
// for OpenCode -- but it lives inside the config file, which the reader copies.
// The shell export is the same everywhere, so a document that branched on it
// would be teaching a difference that does not exist at that level.
func TestSetupDocDoesNotTeachTheWrongVariableSpelling(t *testing.T) {
	t.Parallel()

	for _, format := range models.AllConfigFormats {
		t.Run(string(format), func(t *testing.T) {
			t.Parallel()

			files, err := Render(testInput(), format)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}

			var setup string
			for _, f := range files {
				if f.Filename == SetupDocFilename {
					setup = f.Content
				}
			}

			// The shell export is always the plain variable name.
			if !strings.Contains(setup, "export "+APIKeyEnvVar) {
				t.Errorf("the setup document does not show the export: %s", setup)
			}
			// And it must not imply a shell-level difference that is not there.
			if strings.Contains(setup, "{"+APIKeyEnvVar+"}") {
				t.Errorf("the setup document uses a brace form the shell does not expand: %s", setup)
			}
		})
	}
}

// TestCopilotToolTypeDefaultsToItsFormat checks the tool-type mapping, which is
// the thing that makes the wizard one step: the operator picks the tool once,
// at registration, and every later request renders the right layout.
func TestCopilotToolTypeDefaultsToItsFormat(t *testing.T) {
	t.Parallel()

	if got := models.ToolCopilotVS.DefaultFormat(); got != models.ConfigFormatVSCode {
		t.Errorf("copilot-vs default = %s, want vscode", got)
	}
	if got := models.ToolCopilotCLI.DefaultFormat(); got != models.ConfigFormatCopilot {
		t.Errorf("copilot-cli default = %s, want copilot", got)
	}
	for _, tt := range []models.ToolType{models.ToolCopilotVS, models.ToolCopilotCLI} {
		if !tt.SupportsConfigGeneration() {
			t.Errorf("%q.SupportsConfigGeneration() = false, want true", tt)
		}
	}
}
