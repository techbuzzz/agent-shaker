package models

import "testing"

// TestIsValidTaskStatus pins the vocabulary every surface now shares.
//
// The regression this guards against is real and shipped: the REST validator
// accepted one set, the MCP tools another, and the database accepted anything.
// An agent following our own generated instructions could write a status the
// API then rejected. These cases include the values that used to disagree.
func TestIsValidTaskStatus(t *testing.T) {
	valid := []string{"pending", "in_progress", "blocked", "done", "cancelled"}
	for _, s := range valid {
		if !IsValidTaskStatus(s) {
			t.Errorf("%q should be a valid task status", s)
		}
	}

	invalid := []string{
		"",
		"completed", // the A2A protocol's vocabulary, not the coordination one
		"failed",    // likewise
		"working",
		"DONE", // case-sensitive on purpose: it is stored verbatim
		"in progress",
		"in_progress ",
	}
	for _, s := range invalid {
		if IsValidTaskStatus(s) {
			t.Errorf("%q should not be a valid task status", s)
		}
	}
}

// TestIsValidAgentStatus covers the vocabulary that had drifted furthest.
//
// The MCP surface offered idle/working/blocked/offline while the REST validator
// accepted active/idle/offline — so an agent told by our own onboarding text to
// report "blocked" or "working" wrote a value the API rejected, and "active"
// was unreachable from MCP at all.
func TestIsValidAgentStatus(t *testing.T) {
	for _, s := range []string{"active", "idle", "blocked", "offline"} {
		if !IsValidAgentStatus(s) {
			t.Errorf("%q should be a valid agent status", s)
		}
	}

	for _, s := range []string{"", "working", "in_progress", "Active", "offline "} {
		if IsValidAgentStatus(s) {
			t.Errorf("%q should not be a valid agent status", s)
		}
	}
}

// TestIsTerminalTaskStatus fixes the definition the claim guard depends on.
// The SQL states terminal states literally, so this and the migration have to
// agree; a value that is terminal in Go but claimable in SQL would let a late
// heartbeat resurrect finished work.
func TestIsTerminalTaskStatus(t *testing.T) {
	for _, s := range []string{"done", "cancelled"} {
		if !IsTerminalTaskStatus(s) {
			t.Errorf("%q should be terminal", s)
		}
	}
	for _, s := range []string{"pending", "in_progress", "blocked"} {
		if IsTerminalTaskStatus(s) {
			t.Errorf("%q should not be terminal", s)
		}
	}
}

// TestCoordinationVocabularyExcludesA2AValues guards the confusion this file
// exists to prevent.
//
// There are two status vocabularies in this codebase and they belong to
// different domains: these task states, and the A2A protocol's own task states
// in internal/a2a/models (pending/working/completed/failed). They must never be
// interchanged — a value from one is not valid in the other, which is exactly
// the mistake that produced the contradictory constants removed from task.go.
//
// Note that "blocked" is legitimately present in both the task and agent
// vocabularies: a task can be blocked and an agent can be blocked, and those
// are independent facts. Only the A2A-specific values are checked here.
func TestCoordinationVocabularyExcludesA2AValues(t *testing.T) {
	a2aOnly := []string{"completed", "failed", "working", "input-required", "auth-required"}

	task := make(map[string]bool, len(AllTaskStatuses))
	for _, s := range AllTaskStatuses {
		task[string(s)] = true
	}
	agent := make(map[string]bool, len(AllAgentStatuses))
	for _, s := range AllAgentStatuses {
		agent[string(s)] = true
	}

	for _, s := range a2aOnly {
		if task[s] {
			t.Errorf("task vocabulary contains A2A-only value %q", s)
		}
		if agent[s] {
			t.Errorf("agent vocabulary contains A2A-only value %q", s)
		}
	}
}

// TestJoinValuesFeedsErrorMessages checks the derived message the MCP surface
// returns for a bad status. It must name every accepted value, or an agent
// cannot recover without guessing.
func TestJoinValuesFeedsErrorMessages(t *testing.T) {
	got := JoinValues(AllTaskStatuses)
	want := "pending, in_progress, blocked, done, cancelled"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got := JoinValues(AllAgentStatuses); got != "active, idle, blocked, offline" {
		t.Fatalf("got %q", got)
	}
}
