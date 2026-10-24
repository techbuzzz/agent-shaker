// Package configgen renders the MCP configuration that connects one agent to
// this server.
//
// It is the single source of truth for what a generated file contains. Before
// this package the logic lived in the browser (web/app/composables/useMcpSetup),
// which meant the answer existed twice the moment a CLI or a plugin needed one:
// two implementations of the same URL, drifting apart, with nothing to say
// which was right.
//
// Three properties are load-bearing and each has a test:
//
//  1. No credential ever appears in a rendered file. Every header references
//     ${env:AGENT_SHAKER_API_KEY}, which is what makes a generated config safe
//     to commit. The browser substitutes the plaintext it is holding, at copy
//     time, in memory — the server never sends it twice.
//  2. The MCP URL is byte-identical to what the client-side generator produced,
//     so a config generated here works against a server that has not been
//     upgraded.
//  3. The generator is pure. It takes an origin and returns strings. It does
//     not read the environment, the database or the request, so the same input
//     always produces the same output and the tests need nothing running.
package configgen

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// APIKeyEnvVar is the environment variable every generated config references.
const APIKeyEnvVar = "AGENT_SHAKER_API_KEY"

// APIKeyPlaceholder is the literal that appears in a rendered file where the
// credential goes.
//
// It has to survive a round trip through JSON, through the database-free
// renderer and through a clipboard, so it is written as the literal characters
// `${env:AGENT_SHAKER_API_KEY}` — not as an interpolated Go string, which would
// have produced an empty header.
//
// The web layer substitutes the plaintext for exactly this substring, with
// String.split/join rather than String.replace: the placeholder contains `$`,
// and `$&` / `$'` / “ $` “ in a replacement string are substitution patterns,
// so replace() corrupts any secret that happens to contain those characters.
const APIKeyPlaceholder = "${env:" + APIKeyEnvVar + "}"

// OpenCodeAPIKeyPlaceholder is OpenCode's spelling of the same idea.
//
// One dollar-less brace pair, per https://opencode.ai/docs/mcp-servers/. A
// config written with the ${env:VAR} form is accepted by OpenCode and expands
// to nothing, so the agent sends `Bearer ` and gets a 401 that looks like a
// revoked key rather than a formatting mistake.
const OpenCodeAPIKeyPlaceholder = "{env:" + APIKeyEnvVar + "}"

// ServerName is the key the server entry gets inside mcpServers.
const ServerName = "agent-shaker"

// Input is everything the renderer needs. Every field is passed in: the
// generator deliberately cannot look an origin up for itself, because "which
// host can the outside world reach" is a deployment question with three
// legitimate answers (publicurl.Resolve), and a fourth guessed one would send
// every generated config to an address that only works on the developer's
// machine.
type Input struct {
	// Origin is the scheme and host an agent should connect to, with no
	// trailing slash. May be empty, in which case the URL is relative.
	Origin string

	ProjectID   string
	ProjectName string

	// AgentID is optional. An unbound connection is legitimate — it is how an
	// agent registers itself with register_self — so this does not fail when
	// empty.
	AgentID   string
	AgentName string

	// ToolType picks the default format when the caller does not name one.
	ToolType models.ToolType
}

// Format resolves which layout to render: the requested one when the caller
// named a valid format, otherwise the one that fits the agent's tool type.
func (in Input) Format(requested string) (models.ConfigFormat, error) {
	if strings.TrimSpace(requested) != "" {
		return models.ParseConfigFormat(requested)
	}
	return in.ToolType.DefaultFormat(), nil
}

// MCPURL is the endpoint an agent connects to.
//
// The shape is fixed: {origin}/mcp?project_id=…&agent_id=…, with the agent id
// omitted entirely when unbound. An empty parameter would be a *different*
// identity from no parameter on any server that treats "" as "the agent with
// the empty id", so it is left out rather than sent blank.
func (in Input) MCPURL() string {
	base := strings.TrimRight(strings.TrimSpace(in.Origin), "/") + "/mcp"
	q := url.Values{}
	if in.ProjectID != "" {
		q.Set("project_id", in.ProjectID)
	}
	if in.AgentID != "" {
		q.Set("agent_id", in.AgentID)
	}
	if len(q) == 0 {
		return base
	}
	return base + "?" + q.Encode()
}

// Render produces every file for one format: the connection config, and the
// instructions that tell the agent how the server is meant to be used.
//
// The instructions are not decoration. The most common failure in this product
// is not a broken connection — it is a working connection in which the agent
// never calls send_heartbeat, because nothing told it that a lease exists. The
// generated document is where that is said.
func Render(in Input, format models.ConfigFormat) ([]models.ConfigFile, error) {
	switch format {
	case models.ConfigFormatClaude, models.ConfigFormatCodex, models.ConfigFormatCursor,
		models.ConfigFormatOpenCode, models.ConfigFormatVSCode, models.ConfigFormatCopilot,
		models.ConfigFormatGeneric:
		return renderFor(format, in)
	default:
		// Unreachable through ParseConfigFormat; kept so a future format added
		// to AllConfigFormats without a case here fails loudly instead of
		// returning an empty file list that looks like success.
		return nil, fmt.Errorf("configgen: no renderer for format %q", format)
	}
}

// layout is the per-format difference: where the file goes, what it is called,
// and how the client spells an environment variable.
//
// Everything else about the file is identical across formats, which is why the
// renderers below are one function with a lookup rather than four near-copies.
type layout struct {
	filename    string
	target      string
	description string
	install     string

	// serverKey is the top-level object the server entry lives under. Most
	// hosts read `mcpServers`; OpenCode reads `mcp`.
	serverKey string

	// credential is how this client refers to an environment variable inside a
	// config file. It is NOT a style preference — it is the difference between
	// the client expanding the value and handing the agent the literal string.
	//
	// Claude Code, Cursor and the mcpServers convention expand ${env:VAR}.
	// OpenCode expands {env:VAR}, per https://opencode.ai/docs/mcp-servers/.
	// Emitting the other spelling produces a file that is valid JSON, shows a
	// real-looking header, and sends a 401 that reads like a wrong key.
	credential string

	// entryType is the discriminator OpenCode requires on every remote server.
	entryType string

	// instructionsFile is where the working document goes for this client.
	//
	// Copilot is the odd one: it reads `.github/copilot-instructions.md`
	// automatically, so its working document has to live there rather than
	// beside the config under a name nothing would pick up.
	instructionsFile string
}

func layoutFor(format models.ConfigFormat) layout {
	switch format {
	case models.ConfigFormatClaude:
		return layout{
			filename:         ".claude/mcp.json",
			description:      "Claude Code reads this from the project directory or ~/.claude",
			serverKey:        "mcpServers",
			credential:       APIKeyPlaceholder,
			instructionsFile: "AGENT_SHAKER.md",
		}
	case models.ConfigFormatOpenCode:
		// opencode.json, not a dedicated mcp file: OpenCode keeps MCP servers
		// alongside its other settings in one document. Writing .opencode/mcp.json
		// would produce a file the tool never reads.
		return layout{
			filename:         "opencode.json",
			description:      "OpenCode reads this from the project directory or ~/.config/opencode",
			serverKey:        "mcp",
			credential:       OpenCodeAPIKeyPlaceholder,
			entryType:        "remote",
			instructionsFile: "AGENT_SHAKER.md",
		}
	case models.ConfigFormatCodex:
		return layout{
			// Codex reads a user-level config, not a repository one: there is
			// no per-project Codex config, so pointing at the repo would hand
			// back a file the tool silently ignores.
			target:           "~/.codex/mcp.json",
			description:      "Codex CLI reads this from your home directory",
			install:          "mkdir -p ~/.codex && cat > ~/.codex/mcp.json",
			serverKey:        "mcpServers",
			credential:       APIKeyPlaceholder,
			instructionsFile: "AGENT_SHAKER.md",
		}
	case models.ConfigFormatCursor:
		return layout{
			filename:         ".cursor/mcp.json",
			description:      "Cursor reads this from the project directory",
			serverKey:        "mcpServers",
			credential:       APIKeyPlaceholder,
			instructionsFile: "AGENT_SHAKER.md",
		}
	case models.ConfigFormatVSCode:
		// VS Code reads `servers`, not `mcpServers`. The two look interchangeable
		// and are not: a file written with the wrong top-level key is valid JSON
		// that VS Code ignores in silence, and the agent connects to nothing.
		return layout{
			filename:         ".vscode/mcp.json",
			description:      "VS Code reads this from the project directory",
			serverKey:        "servers",
			credential:       APIKeyPlaceholder,
			instructionsFile: "AGENT_SHAKER.md",
		}
	case models.ConfigFormatCopilot:
		// The pair Copilot needs: VS Code's config, because that is where Copilot
		// in VS Code reads its MCP servers from, and the instructions file at the
		// path Copilot reads without being asked.
		//
		// The old client-side bundle shipped exactly these two together, and
		// splitting them would be worse: a Copilot user who only received the
		// instructions would have a bundle that connects to nothing.
		return layout{
			filename:         ".vscode/mcp.json",
			description:      "VS Code reads this from the project directory; Copilot in VS Code reads its MCP servers here",
			serverKey:        "servers",
			credential:       APIKeyPlaceholder,
			instructionsFile: ".github/copilot-instructions.md",
		}
	default:
		return layout{
			filename:         ".mcp.json",
			description:      "The convention read by Visual Studio 2026 and most MCP hosts",
			serverKey:        "mcpServers",
			credential:       APIKeyPlaceholder,
			instructionsFile: "AGENT_SHAKER.md",
		}
	}
}

func renderFor(format models.ConfigFormat, in Input) ([]models.ConfigFile, error) {
	lay := layoutFor(format)

	// Every host in mcpServers form uses the same shape; OpenCode differs in
	// three ways at once (the `mcp` key, an explicit `type: remote`, and its own
	// {env:VAR} syntax), so it gets its own branch rather than three booleans
	// threaded through one template.
	var config string
	if lay.entryType != "" {
		config = fmt.Sprintf(`{
  %q: {
    %q: {
      "type": %q,
      "url": %q,
      "enabled": true,
      "headers": {
        "Authorization": "Bearer %s"
      }
    }
  }
}
`, lay.serverKey, ServerName, lay.entryType, in.MCPURL(), lay.credential)
	} else {
		config = fmt.Sprintf(`{
  %q: {
    %q: {
      "type": "http",
      "url": %q,
      "headers": {
        "Authorization": "Bearer %s"
      }
    }
  }
}
`, lay.serverKey, ServerName, in.MCPURL(), lay.credential)
	}

	instructions := in.instructions(lay.credential)

	return []models.ConfigFile{
		{
			Format:      format,
			Filename:    lay.filename,
			Target:      lay.target,
			Description: lay.description,
			Install:     lay.install,
			Content:     config,
		},
		{
			Format:      format,
			Filename:    lay.instructionsFile,
			Description: "How the agent is expected to work with this server",
			Content:     instructions.body,
		},
		{
			Format:      format,
			Filename:    SetupDocFilename,
			Description: "How to set the key up and check the connection",
			Content:     in.setupDoc(lay, format),
		},
	}, nil
}

type instructionsFile struct {
	filename string
	target   string
	install  string
	body     string
}

// SetupDocFilename is the setup document emitted alongside every bundle.
//
// It was in the original client-side bundle and was lost when the generator
// moved to the server, along with the VS Code and Copilot files. It is the
// document that answers the two questions a developer actually has when a
// generated bundle does not work: which file goes where, and how do I check
// that the connection is alive. Everything else in the bundle assumes both are
// obvious.
const SetupDocFilename = "AGENT_SHAKER_SETUP.md"

// setupDoc explains the bundle: what to set, where the files go, and how to
// verify the connection.
func (in Input) setupDoc(lay layout, format models.ConfigFormat) string {
	projectLabel := in.ProjectName
	if projectLabel == "" {
		projectLabel = in.ProjectID
	}

	files := []string{}
	if lay.filename != "" {
		files = append(files, "- `"+lay.filename+"` — "+lay.description)
	}
	if lay.target != "" {
		files = append(files, "- `"+lay.target+"` — "+lay.description)
	}
	// The working document is listed too: it is what tells the agent how the
	// server is meant to be used, and a reader who does not know it exists
	// will never open it.
	if lay.instructionsFile != "" {
		files = append(files, "- `"+lay.instructionsFile+"` — how the agent is expected to work with this server")
	}
	files = append(files, "- `"+SetupDocFilename+"` — this file")

	// The shell export is the same for every client. What differs is the
	// spelling inside the config file — ${env:VAR} for most, {env:VAR} for
	// OpenCode — and the reader never types that, they copy the file. So this
	// document does not branch on the client: there is nothing for it to say
	// that the file itself does not already say.
	return "# Agent Shaker setup\n" +
		"\n" +
		"Generated for project **" + projectLabel + "**" +
		", format `" + string(format) + "`.\n" +
		"\n" +
		"## 1. Set the API key\n" +
		"\n" +
		"The files reference `" + APIKeyEnvVar + "`; they never contain the key itself,\n" +
		"so they are safe to commit.\n" +
		"\n" +
		"```bash\n" + "export " + APIKeyEnvVar + "=<your key>\n" + "```\n" +
		"\n" +
		"Keep it in your shell or a git-ignored `.env` — never in a tracked file.\n" +
		"\n" +
		"## 2. The files\n" +
		"\n" +
		strings.Join(files, "\n") + "\n" +
		"\n" +
		"## 3. Verify\n" +
		"\n" +
		"```bash\n" +
		"curl -sS -X POST \"" + in.MCPURL() + "\" \\\n" +
		"  -H \"Authorization: Bearer $" + APIKeyEnvVar + "\" \\\n" +
		"  -H 'Content-Type: application/json' \\\n" +
		"  -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{}}'\n" +
		"```\n" +
		"\n" +
		"An `initialize` result means the connection is good. A `401` means the key\n" +
		"is wrong. A `403` naming an agent id means the key is bound to a different\n" +
		"agent — use the key issued for this one. A `404` means the edge is not\n" +
		"routing `/mcp`; see docs/DEPLOYMENT.md.\n"
}

// instructions builds the markdown document that accompanies the config.
//
// The credential spelling in the prose has to match the spelling in the config
// file beside it, or the agent copies a working config and then writes its own
// header wrong: OpenCode expands {env:VAR}, everything else ${env:VAR}. One
// document, one spelling, taken from the layout rather than re-derived here.
func (in Input) instructions(credential string) instructionsFile {
	scope := fmt.Sprintf("- Project: **%s** (`%s`)", in.ProjectName, in.ProjectID)
	if in.AgentName != "" {
		scope += fmt.Sprintf("\n- Agent: **%s** (`%s`)", in.AgentName, in.AgentID)
	} else if in.AgentID != "" {
		scope += fmt.Sprintf("\n- Agent: `%s`", in.AgentID)
	} else {
		scope += "\n- Agent: not bound — call `register_self`, or bind one with a key issued for that agent"
	}

	toolSection := renderToolGroups()

	// Built as a quoted string rather than a raw one because the document is
	// full of backticks — they are what make the tool names read as code in a
	// markdown renderer, and a raw Go string cannot contain them.
	body := fmt.Sprintf("# Agent Shaker — how to work\n"+
		"\n"+
		"This repository is coordinated through the **Agent Shaker** server. Use it\n"+
		"rather than tracking work in local notes: it is the shared source of truth,\n"+
		"so other agents see what you do and you see what they do.\n"+
		"\n"+
		"%s\n"+
		"\n"+
		"Endpoint: `%s`\n"+
		// %s carries the client's own variable spelling (${env:VAR} or
		// {env:VAR}) straight through, unescaped. An earlier version doubled the
		// dollar sign to survive fmt.Sprintf, which taught every agent to write
		// `$${...}` — a literal dollar in front of the key, and therefore a 401
		// on the first authenticated call.
		"Authentication: `Authorization: Bearer %s`\n"+
		"\n"+
		"## The loop\n"+
		"\n"+
		"1. `get_my_identity` first. It tells you which agent and project this\n"+
		"   connection is bound to. If it is not bound, `register_self`.\n"+
		"2. `get_my_tasks` for the work assigned to you.\n"+
		"3. `claim_task` before you start, so other agents can see it is in progress.\n"+
		"4. `send_heartbeat` while you work. **This is not optional.** The heartbeat is\n"+
		"   a lease: stop sending and the reaper marks you offline and returns your\n"+
		"   tasks to the backlog while you are still working on them. Send it every\n"+
		"   30 seconds, with `progress_pct` and an `eta` when you have them.\n"+
		"5. `complete_task_with_handoff` when the work is genuinely done. If the next\n"+
		"   person needs context you have, hand it off rather than leaving them to\n"+
		"   rediscover it.\n"+
		"6. `raise_blocker` the moment you are stuck. A typed blocker is routed to a\n"+
		"   human; silence is not.\n"+
		"7. Read `list_contexts` before deciding, and record the decision with\n"+
		"   `add_context` so the next agent inherits it.\n"+
		"\n"+
		"## Available tools\n"+
		"\n"+
		"%s\n"+
		"## Notes\n"+
		"\n"+
		"- Writes to tasks and milestones are PM-gated. A non-PM agent gets a clear\n"+
		"  refusal rather than a silent no-op.\n"+
		"- Live updates also arrive over the app's WebSocket. The MCP connection is\n"+
		"  for acting; the UI is for watching.\n",
		scope, in.MCPURL(), credential, toolSection)

	return instructionsFile{
		filename: "AGENT_SHAKER.md",
		body:     body,
	}
}

// toolGroup is one section of the generated tool list.
type toolGroup struct {
	heading string
	note    string
	tools   []string
}

// toolGroups is the tool list, carried over from the client-side generator it
// replaces so a generated document is no thinner than what ships today.
//
// The Coordination Plane groups are the ones an agent working alone is most
// likely to skip, which is exactly why they are enumerated rather than left to
// the tool listing the client already has.
var toolGroups = []toolGroup{
	{
		heading: "Orientation",
		note:    "Call these first. They resolve the agent and project from the connection.",
		tools:   []string{"get_my_identity", "get_my_project"},
	},
	{
		heading: "Working on tasks",
		note:    "The core loop. Claim a task, do the work, report back.",
		tools:   []string{"get_my_tasks", "claim_task", "complete_task", "complete_task_with_handoff", "reassign_task", "update_my_status"},
	},
	{
		heading: "Staying visible",
		note:    "The lease. An agent that stops heartbeating is reaped and its tasks go back to the backlog.",
		tools:   []string{"send_heartbeat"},
	},
	{
		heading: "Blocked",
		note:    "Escalate rather than going quiet. A blocker is routed to a human.",
		tools:   []string{"raise_blocker", "check_blocker_status", "get_my_blockers"},
	},
	{
		heading: "Dependencies",
		note:    "Work that is ordered behind other work.",
		tools:   []string{"add_task_dependency", "get_task_dependencies"},
	},
	{
		heading: "Tasks and planning",
		note:    "Broader project views, for a PM role or a wider look.",
		tools: []string{"list_tasks", "create_task", "update_task_status", "get_dashboard",
			"list_milestones", "create_milestone", "assign_task_to_milestone"},
	},
	{
		heading: "Shared knowledge",
		note:    "Markdown contexts shared between agents. Read before acting, write after deciding.",
		tools: []string{"list_contexts", "search_contexts", "add_context",
			"list_global_contexts", "read_global_context", "publish_global_context"},
	},
	{
		heading: "Other agents",
		note:    "Discover and delegate to external agents over A2A.",
		tools:   []string{"discover_a2a_agent", "delegate_to_a2a_agent", "get_a2a_task_status"},
	},
	{
		heading: "Onboarding",
		note:    "Only needed the very first time an agent connects to a new project.",
		tools:   []string{"register_self"},
	},
}

func renderToolGroups() string {
	var b strings.Builder
	for _, g := range toolGroups {
		fmt.Fprintf(&b, "### %s\n\n%s\n\n", g.heading, g.note)
		for _, t := range g.tools {
			b.WriteString("  - `" + t + "`\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}
