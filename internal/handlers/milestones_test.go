package handlers

import (
	"errors"
	"testing"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// TestMilestoneCloseGuardErrorFormatting covers the error string format
// that httpx.classify matches on. If the format ever changes, both must
// be updated together or the wire contract silently breaks (we'd start
// surfacing 500 instead of 409 for the "milestone has open tasks" case).
func TestMilestoneCloseGuardErrorFormatting(t *testing.T) {
	err := errMilestoneOpenTasks{Open: 3}
	msg := err.Error()
	if msg == "" {
		t.Fatal("empty error message")
	}
	// httpx.classify matches:
	//   strings.HasPrefix(err.Error(), "milestone has ")
	//   strings.Contains(err.Error(), "open task")
	if got := "milestone has 3 open task"; !startsWith(got, msg) {
		t.Fatalf("expected prefix %q in %q", got, msg)
	}
	if !contains(msg, "open task") {
		t.Fatalf("expected substring 'open task' in %q", msg)
	}
}

func TestGlobalContextConflictErrorFormatting(t *testing.T) {
	err := errGlobalContextConflict{Title: "On-call playbook"}
	msg := err.Error()
	if !contains(msg, "global context with title") || !contains(msg, "already exists") {
		t.Fatalf("expected conflict markers in %q", msg)
	}
	if !contains(msg, "On-call playbook") {
		t.Fatalf("expected title in %q", msg)
	}
}

// TestMilestoneStatusEnumIsClosed — guards the type system so a new
// status added to ValidMilestoneStatuses matches the DB CHECK constraint.
func TestMilestoneStatusEnumIsClosed(t *testing.T) {
	want := map[models.MilestoneStatus]bool{
		models.MilestonePlanned: true,
		models.MilestoneActive:  true,
		models.MilestoneDone:    true,
		models.MilestoneDropped: true,
	}
	for s, ok := range want {
		if !models.ValidMilestoneStatuses[s] || !ok {
			t.Fatalf("ValidMilestoneStatuses missing %q", s)
		}
	}
	if got := len(models.ValidMilestoneStatuses); got != len(want) {
		t.Fatalf("ValidMilestoneStatuses has %d entries, want %d (open enum)", got, len(want))
	}
}

func TestGlobalContextScopeIsClosed(t *testing.T) {
	want := []models.GlobalContextScope{models.ScopeGlobal, models.ScopeProject}
	for _, s := range want {
		if !models.ValidGlobalContextScopes[s] {
			t.Fatalf("ValidGlobalContextScopes missing %q", s)
		}
	}
	if got := len(models.ValidGlobalContextScopes); got != len(want) {
		t.Fatalf("ValidGlobalContextScopes has %d entries, want %d (open enum)", got, len(want))
	}
}

// TestProjectRepoRolesOpenButKnown — the project_repos.role column is an
// open VARCHAR in the DB; we only validate the well-known values in the
// validator. This test ensures we do not accidentally close the enum
// (which would break already-deployed rows whose role is a custom value).
func TestProjectRepoRolesOpenButKnown(t *testing.T) {
	known := []string{"code", "infra", "docs", "design"}
	for _, r := range known {
		if !models.ValidProjectRepoRoles[r] {
			t.Fatalf("ValidProjectRepoRoles missing %q", r)
		}
	}
	// empty string means "default to 'code'" — must remain valid
	if !models.ValidProjectRepoRoles[""] {
		t.Fatal("empty role must remain valid (defaults to 'code')")
	}
}

// TestRolePMExists guards the Phase 2 PM role against accidental removal.
func TestRolePMExists(t *testing.T) {
	if !models.RolePM.IsPM() {
		t.Fatalf("RolePM.IsPM() should be true; got false")
	}
	if models.RoleBackend.IsPM() {
		t.Fatalf("RoleBackend.IsPM() must be false")
	}
}

// helpers — kept local to avoid pulling strings-only test deps
func startsWith(prefix, s string) bool {
	if len(prefix) > len(s) {
		return false
	}
	return s[:len(prefix)] == prefix
}

func contains(s, sub string) bool {
	return len(sub) <= len(s) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

var _ = errors.New
