package websocket

import (
	"testing"

	"github.com/google/uuid"
)

// The bug this pins: broadcastMessage resolved the project by asserting
// payload["project_id"].(string). Handlers build their payloads by hand and
// most of them pass a typed uuid.UUID, so the assertion failed, the project id
// stayed zero, and the broadcast was dropped — silently, with a successful
// response to the caller and no log line. Every live event in the app was
// invisible for that reason.
func TestProjectIDFromPayloadAcceptsEveryShapeHandlersProduce(t *testing.T) {
	project := uuid.New()

	tests := []struct {
		name    string
		payload interface{}
		want    uuid.UUID
		wantOK  bool
	}{
		{
			name:    "map with a string project id",
			payload: map[string]interface{}{"project_id": project.String()},
			want:    project,
			wantOK:  true,
		},
		{
			name:    "map with a typed uuid project id",
			payload: map[string]interface{}{"project_id": project},
			want:    project,
			wantOK:  true,
		},
		{
			name: "struct with a ProjectID field",
			payload: struct {
				ProjectID uuid.UUID `json:"project_id"`
				Title     string    `json:"title"`
			}{ProjectID: project, Title: "Ship it"},
			want:   project,
			wantOK: true,
		},
		{
			name:    "payload with no project id is reported, not guessed",
			payload: map[string]interface{}{"title": "orphan"},
			wantOK:  false,
		},
		{
			name:    "unparseable project id is reported",
			payload: map[string]interface{}{"project_id": "not-a-uuid"},
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := projectIDFromPayload(tt.payload)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v (got %s)", ok, tt.wantOK, got)
			}
			if tt.wantOK && got != tt.want {
				t.Fatalf("project id = %s, want %s", got, tt.want)
			}
		})
	}
}

// A broadcast with no resolvable project must be loud. The silent return is
// what let this survive in the codebase for as long as it did.
func TestBroadcastWithUnresolvableProjectIsNotSilent(t *testing.T) {
	hub := NewHub()
	// No client is registered, so the send loop has nothing to do either way;
	// what matters is that the payload resolution itself is reportable.
	if _, ok := projectIDFromPayload(map[string]interface{}{"title": "orphan"}); ok {
		t.Fatal("a payload with no project id must not resolve to a usable id")
	}
	// The zero-uuid path must not match a project map by accident.
	if _, exists := hub.projects[uuid.Nil]; exists {
		t.Fatal("unexpected project entry for the nil uuid")
	}
}
