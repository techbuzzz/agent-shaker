package models

import (
	"errors"
	"strings"
	"testing"
)

func TestParseToolType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in      string
		want    ToolType
		wantErr bool
	}{
		{in: "claude-code", want: ToolClaudeCode},
		{in: "cursor", want: ToolCursor},
		// Case and surrounding whitespace are normalised: the value arrives from
		// a JSON body and from a form, and "Claude-Code" failing would look like
		// a bug rather than a validation error.
		{in: "  Claude-Code  ", want: ToolClaudeCode},
		{in: "UNKNOWN", want: ToolTypeUnknown},
		// Empty is not an error. An agent created by an MCP runtime, or by a
		// client predating this field, does not know the concept — and
		// refusing to create it would make the field a breaking change.
		{in: "", want: ToolTypeUnknown},
		{in: "   ", want: ToolTypeUnknown},
		{in: "vscode", wantErr: true},
		{in: "emacs", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			got, err := ParseToolType(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseToolType(%q) = %q, want an error", tc.in, got)
				}
				if !errors.Is(err, ErrUnknownToolType) {
					t.Errorf("error %v does not wrap ErrUnknownToolType", err)
				}
				// The message enumerates the vocabulary, because this string is
				// what an operator sees when the check constraint would otherwise
				// have surfaced as a 500 naming a database constraint.
				if !strings.Contains(err.Error(), "claude-code") {
					t.Errorf("error %q does not list the accepted values", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseToolType(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("ParseToolType(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestAllToolTypesParse keeps the vocabulary and its own parser from drifting:
// a value in the list that ParseToolType rejects would be a value the UI offers
// and the API refuses.
func TestAllToolTypesParse(t *testing.T) {
	t.Parallel()

	for _, tt := range AllToolTypes {
		got, err := ParseToolType(string(tt))
		if err != nil {
			t.Errorf("AllToolTypes contains %q which ParseToolType rejects: %v", tt, err)
			continue
		}
		if got != tt {
			t.Errorf("ParseToolType(%q) = %q, want the same value back", tt, got)
		}
	}
}

func TestIsValidToolType(t *testing.T) {
	t.Parallel()

	if IsValidToolType("cursor") != true {
		t.Error("IsValidToolType(\"cursor\") = false, want true")
	}
	// Deliberately unlike ParseToolType: the empty string is not "unknown" here,
	// because the callers that ask are checking a value that is already stored,
	// and a stored value is never empty.
	if IsValidToolType("") != false {
		t.Error("IsValidToolType(\"\") = true, want false")
	}
	if IsValidToolType("nope") != false {
		t.Error("IsValidToolType(\"nope\") = true, want false")
	}
}

func TestSupportsConfigGeneration(t *testing.T) {
	t.Parallel()

	// The layouts the config endpoint renders today.
	for _, tt := range []ToolType{ToolClaudeCode, ToolCodex, ToolOpenCode, ToolCursor, ToolCopilotVS, ToolCopilotCLI, ToolGenericMCP} {
		if !tt.SupportsConfigGeneration() {
			t.Errorf("%q.SupportsConfigGeneration() = false, want true", tt)
		}
	}

	// Phase 3 and Phase 4 surfaces, plus an A2A agent, which is not an MCP host
	// at all. An agent on any of them still gets a working generic `.mcp.json`,
	// but there is no layout here that would be better than the generic one.
	for _, tt := range []ToolType{ToolWindsurf, ToolA2A} {
		if tt.SupportsConfigGeneration() {
			t.Errorf("%q.SupportsConfigGeneration() = true, want false", tt)
		}
	}
}

func TestDefaultFormatFallsBackToGeneric(t *testing.T) {
	t.Parallel()

	// Tool types with no dedicated layout get the convention most hosts read,
	// rather than a refusal: a working file for the wrong client beats a 400
	// that explains nothing about what to do instead.
	for _, tt := range []ToolType{ToolWindsurf, ToolA2A, ToolTypeUnknown} {
		if got := tt.DefaultFormat(); got != ConfigFormatGeneric {
			t.Errorf("%q.DefaultFormat() = %s, want generic", tt, got)
		}
	}

	if got := ToolClaudeCode.DefaultFormat(); got != ConfigFormatClaude {
		t.Errorf("claude-code default = %s, want claude", got)
	}
}

func TestParseConfigFormat(t *testing.T) {
	t.Parallel()

	for _, f := range AllConfigFormats {
		got, err := ParseConfigFormat(string(f))
		if err != nil {
			t.Errorf("ParseConfigFormat(%q): %v", f, err)
			continue
		}
		if got != f {
			t.Errorf("ParseConfigFormat(%q) = %q, want the same value back", f, got)
		}
	}

	if got, err := ParseConfigFormat("  CLAUDE "); err != nil || got != ConfigFormatClaude {
		t.Errorf("ParseConfigFormat(\"  CLAUDE \") = %q, %v; want claude, nil", got, err)
	}

	// Unlike ParseToolType there is no empty case: "no format" is not something
	// the endpoint can act on, and silently substituting one would hand back a
	// layout the caller did not ask for.
	if _, err := ParseConfigFormat(""); err == nil {
		t.Error("ParseConfigFormat(\"\") = nil error, want a rejection")
	}

	_, err := ParseConfigFormat("emacs")
	if err == nil {
		t.Fatal("ParseConfigFormat(\"emacs\") = nil error, want a rejection")
	}
	if !strings.Contains(err.Error(), "codex") {
		t.Errorf("error %q does not list the supported formats", err)
	}
}
