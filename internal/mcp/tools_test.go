package mcp

import (
	"sort"
	"testing"
)

// TestEveryAdvertisedToolIsCallable is the invariant that holds the registry's
// two halves in step.
//
// The advertised schemas and the dispatch that serves them were two lists kept
// in step by hand, in a file of three thousand lines. A tool could be
// advertised and never dispatched, or dispatched and never advertised, and
// nothing noticed until an agent called it and got "unknown tool" — or, worse,
// until a tool the server never advertised turned out to be callable.
//
// The check is deliberately symmetric: an executor without a schema is a tool
// nobody can find, and a schema without an executor is a tool that fails when
// it is called.
func TestEveryAdvertisedToolIsCallable(t *testing.T) {
	t.Parallel()

	advertised := make(map[string]bool)
	for _, tool := range toolDefinitions() {
		if advertised[tool.Name] {
			t.Errorf("tool %q is advertised twice", tool.Name)
		}
		advertised[tool.Name] = true
	}

	callable := make(map[string]bool)
	for name := range toolExecutors {
		callable[name] = true
	}

	for _, name := range sortedKeys(advertised) {
		if !callable[name] {
			t.Errorf("tool %q is advertised but has no executor", name)
		}
	}
	for _, name := range sortedKeys(callable) {
		if !advertised[name] {
			t.Errorf("tool %q has an executor but is not advertised", name)
		}
	}
}

// TestEveryToolHasADescriptionAndAnObjectSchema checks the shape of what is
// advertised, because a tool with an empty description is one an agent has no
// way to choose between.
func TestEveryToolHasADescriptionAndAnObjectSchema(t *testing.T) {
	t.Parallel()

	for _, tool := range toolDefinitions() {
		if tool.Description == "" {
			t.Errorf("tool %q has no description", tool.Name)
		}
		if tool.InputSchema.Type != "object" {
			t.Errorf("tool %q has schema type %q, want object", tool.Name, tool.InputSchema.Type)
		}
	}
}

func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
