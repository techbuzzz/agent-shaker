package mcp

import (
	"encoding/json"
	"testing"
)

// TestRequireArgRejectsBlankValues is the regression test for a class of bug
// that produced a real data defect.
//
// `args["title"].(string)` succeeds for "": an empty string is a perfectly good
// string, so a type assertion alone accepted a task titled with nothing. That
// row then rendered as a nameless card in every list, and the empty title was
// the only symptom. Eleven sites had the type-only check; thirteen had the
// blank check. Nothing about the two forms looks different to a reader, which
// is why one helper now serves both.
func TestRequireArgRejectsBlankValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantOK  bool
		wantVal string
	}{
		{name: "present and non-blank", args: map[string]interface{}{"title": "REST API"}, wantOK: true, wantVal: "REST API"},
		{name: "absent", args: map[string]interface{}{}, wantOK: false},
		{name: "nil value", args: map[string]interface{}{"title": nil}, wantOK: false},
		{name: "empty string", args: map[string]interface{}{"title": ""}, wantOK: false},
		{name: "spaces only", args: map[string]interface{}{"title": "   "}, wantOK: false},
		{name: "tabs and newlines", args: map[string]interface{}{"title": "\t\n "}, wantOK: false},
		{name: "wrong type", args: map[string]interface{}{"title": 42}, wantOK: false},
		{name: "wrong type object", args: map[string]interface{}{"title": map[string]interface{}{}}, wantOK: false},
		// Surrounding whitespace on a real value is preserved, not trimmed: the
		// blankness test trims, the returned value does not. A markdown context
		// body that begins with indentation is content, not noise.
		{name: "padded value is kept verbatim", args: map[string]interface{}{"title": "  spaced  "}, wantOK: true, wantVal: "  spaced  "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := requireArg(tc.args, "title")
			if ok != tc.wantOK {
				t.Fatalf("requireArg ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && got != tc.wantVal {
				t.Errorf("requireArg = %q, want %q", got, tc.wantVal)
			}
		})
	}
}

// TestListOrEmptyMarshalsAsArray pins the second defect: a list tool answering
// `null` when the answer is "none".
//
// encoding/json renders a nil slice as null and an empty one as [], so the shape
// depended on whether the query happened to return a row — and the case where it
// returns none is exactly the one a caller has to handle. An agent that has to
// special-case "null or array" reads it as "sometimes this key is missing",
// which is a different and more alarming statement than "this list is empty".
func TestListOrEmptyMarshalsAsArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []string
	}{
		{name: "nil", in: nil},
		{name: "empty", in: []string{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			out, err := json.Marshal(listOrEmpty(tc.in))
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(out) != "[]" {
				t.Errorf("marshalled %v as %s, want []", tc.in, out)
			}
		})
	}
}

func TestListOrEmptyPreservesContent(t *testing.T) {
	t.Parallel()

	in := []string{"a", "b"}
	out := listOrEmpty(in)
	if len(out) != 2 || out[0] != "a" || out[1] != "b" {
		t.Errorf("listOrEmpty changed a populated slice: %v", out)
	}
}

// TestRequiredStringSchemasDeclareMinLength closes the loop on the same defect
// from the client side.
//
// The server now rejects a blank title, but a client that validates against the
// published schema should never send one. `required` only covers presence — the
// key has to exist — so without minLength a client is told that `{"title": ""}`
// is valid and only learns otherwise from a round trip it would rather not make.
func TestRequiredStringSchemasDeclareMinLength(t *testing.T) {
	t.Parallel()

	h := NewMCPHandler(nil, nil)
	result, rpcErr := h.handleToolsList(MCPContext{})
	if rpcErr != nil {
		t.Fatalf("handleToolsList: %v", rpcErr)
	}
	list := result.(ToolsListResult)

	// The string fields that carry an identifier or a human-readable name.
	// Optional/free-text fields are deliberately excluded: minLength on them
	// would forbid arguments the tools treat as empty-is-absent.
	requiredNonBlank := []string{"title", "task_id", "agent_id", "project_id", "milestone_id", "content", "name", "message", "agent_url", "status"}

	seen := 0
	for _, tool := range list.Tools {
		for _, field := range tool.InputSchema.Required {
			if !contains(requiredNonBlank, field) {
				continue
			}
			prop, ok := tool.InputSchema.Properties[field].(map[string]interface{})
			if !ok {
				continue
			}
			if prop["type"] != "string" {
				continue
			}
			seen++
			if _, has := prop["minLength"]; !has {
				t.Errorf("%s.%s is required but has no minLength", tool.Name, field)
			}
		}
	}

	if seen == 0 {
		t.Fatal("no required string fields found; the loop above is not testing anything")
	}
}

// The empty-query path of executeSearchContexts is deliberately not asserted
// here. It checks h.db before short-circuiting, so the only way to reach it is
// against a real database — and listOrEmpty is covered directly above, which is
// the part this work changed. Asserting it through a nil handler would pin the
// current check order rather than the behaviour anyone cares about.

func contains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}
