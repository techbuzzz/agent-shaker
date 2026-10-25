package hooks

import (
	"testing"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

func TestCompileMatcherRejectsAnEmptyField(t *testing.T) {
	if _, err := CompileMatcher(&models.HookMatcher{Pattern: ".*"}); err == nil {
		t.Fatal("a matcher without a field must be rejected")
	}
}

func TestCompileMatcherRejectsABrokenPattern(t *testing.T) {
	if _, err := CompileMatcher(&models.HookMatcher{Field: "title", Pattern: "([a-z"}); err == nil {
		t.Fatal("a matcher with an uncompilable pattern must be rejected")
	}
}

func TestNilMatcherMatchesEveryEvent(t *testing.T) {
	matcher, err := CompileMatcher(nil)
	if err != nil {
		t.Fatalf("compiling a nil matcher: %v", err)
	}
	if !matcher.Matches(map[string]any{"title": "anything"}) {
		t.Fatal("a nil matcher must match every event")
	}
}

func TestMatcherReadsANestedField(t *testing.T) {
	matcher, err := CompileMatcher(&models.HookMatcher{Field: "actor.name", Pattern: "^backend"})
	if err != nil {
		t.Fatalf("compiling the matcher: %v", err)
	}

	payload := map[string]any{"actor": map[string]any{"name": "backend-1"}}
	if !matcher.Matches(payload) {
		t.Fatal("the matcher must read a nested field")
	}
}

func TestMatcherDoesNotMatchAMissingField(t *testing.T) {
	matcher, err := CompileMatcher(&models.HookMatcher{Field: "title", Pattern: ".*"})
	if err != nil {
		t.Fatalf("compiling the matcher: %v", err)
	}

	if matcher.Matches(map[string]any{"other": "value"}) {
		t.Fatal("a payload without the field must not match")
	}
}

func TestRenderResolvesTheKnownPlaceholders(t *testing.T) {
	rc := RenderContext{
		Event:      "task.completed",
		Project:    ProjectContext{ID: "p1", Name: "platform", RepoPath: "/srv/platform"},
		Entity:     EntityContext{ID: "t1", Title: "add pagination", Status: "done"},
		Actor:      ActorContext{ID: "a1", Name: "backend-1"},
		DeliveryID: "d1",
	}

	got, err := Render("{{project.name}}/{{entity.title}} by {{actor.name}}", rc)
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}

	want := "platform/add pagination by backend-1"
	if got != want {
		t.Fatalf("rendered %q, want %q", got, want)
	}
}

func TestRenderRejectsAnUnknownPlaceholder(t *testing.T) {
	if _, err := Render("{{nope}}", RenderContext{}); err == nil {
		t.Fatal("an unknown placeholder must be an error, not an empty string")
	}
}

func TestRenderRejectsAnUnterminatedPlaceholder(t *testing.T) {
	if _, err := Render("{{project.name", RenderContext{}); err == nil {
		t.Fatal("an unterminated placeholder must be an error")
	}
}

func TestRenderLeavesTextWithoutPlaceholdersAlone(t *testing.T) {
	got, err := Render("plain text", RenderContext{})
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}
	if got != "plain text" {
		t.Fatalf("rendered %q, want %q", got, "plain text")
	}
}

func TestRenderArgvKeepsEachArgumentSeparate(t *testing.T) {
	rc := RenderContext{Project: ProjectContext{RepoPath: "/srv/my project"}}

	argv, err := RenderArgv([]string{"run", "--cwd", "{{project.repo_path}}"}, rc)
	if err != nil {
		t.Fatalf("rendering the argument vector: %v", err)
	}

	if len(argv) != 3 {
		t.Fatalf("got %d arguments, want 3", len(argv))
	}
	if argv[2] != "/srv/my project" {
		t.Fatalf("the argument was %q, want %q", argv[2], "/srv/my project")
	}
}

func TestRenderArgvReportsWhichArgumentFailed(t *testing.T) {
	_, err := RenderArgv([]string{"ok", "{{missing}}"}, RenderContext{})
	if err == nil {
		t.Fatal("an unknown placeholder must be an error")
	}
}

func TestClassifyMapsTheExitCodes(t *testing.T) {
	cases := []struct {
		name     string
		exitCode int
		timedOut bool
		blocking bool
		want     string
	}{
		{name: "success", exitCode: 0, want: models.HookRunSucceeded},
		{name: "failure", exitCode: 1, want: models.HookRunFailed},
		{name: "two is a failure by default", exitCode: 2, want: models.HookRunFailed},
		{name: "two blocks when the hook says so", exitCode: 2, blocking: true, want: models.HookRunBlocked},
		{name: "cancelled", exitCode: 130, want: models.HookRunCancelled},
		{name: "timeout wins over the exit code", exitCode: 0, timedOut: true, want: models.HookRunTimeout},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Classify(tc.exitCode, tc.timedOut, tc.blocking)
			if got != tc.want {
				t.Fatalf("classified %q, want %q", got, tc.want)
			}
		})
	}
}
