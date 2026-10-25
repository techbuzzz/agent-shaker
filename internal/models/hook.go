package models

import (
	"time"

	"github.com/google/uuid"
)

// Hook is a command the server runs when a domain event fires.
//
// The command is an argument vector, never a shell string. See
// migrations/019_hooks.up.sql for why that distinction is the load-bearing
// one in this whole table.
type Hook struct {
	ID             uuid.UUID    `json:"id" db:"id"`
	ProjectID      uuid.UUID    `json:"project_id" db:"project_id"`
	Event          string       `json:"event" db:"event"`
	Matcher        *HookMatcher `json:"matcher" db:"matcher"`
	Command        []string     `json:"command" db:"command"`
	Env            HookEnv      `json:"env" db:"env"`
	Stdin          HookStdin    `json:"stdin" db:"stdin"`
	Post           *HookPost    `json:"post" db:"post"`
	TimeoutSeconds int          `json:"timeout_seconds" db:"timeout_seconds"`
	Enabled        bool         `json:"enabled" db:"enabled"`
	CreatedBy      *uuid.UUID   `json:"created_by" db:"created_by"`
	CreatedAt      time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at" db:"updated_at"`
}

// HookEnv is the environment a hook's command runs with.
//
// A value written as "{secret:NAME}" is a reference to a secret the server
// resolves at run time. The resolved value is injected into the command's
// environment and redacted from the captured output; it is never stored here.
type HookEnv map[string]string

// SecretReferencePrefix marks an environment value as a reference to a
// secret rather than a literal.
const SecretReferencePrefix = "{secret:"

// IsSecretReference reports whether the value is a secret reference.
func (e HookEnv) IsSecretReference(name string) bool {
	value, ok := e[name]
	return ok && len(value) > len(SecretReferencePrefix) &&
		value[:len(SecretReferencePrefix)] == SecretReferencePrefix
}

// SecretName returns the secret a value refers to.
func (e HookEnv) SecretName(name string) (string, bool) {
	value, ok := e[name]
	if !ok || !e.IsSecretReference(name) {
		return "", false
	}
	return value[len(SecretReferencePrefix) : len(value)-1], true
}

// HookMatcher narrows a hook to the events whose entity matches a pattern.
type HookMatcher struct {
	Field   string `json:"field" db:"field"`
	Pattern string `json:"pattern" db:"pattern"`
}

// HookStdin says how the event reaches the command's standard input.
//
// Mode "event" writes the whole event as JSON. Mode "prompt" renders a
// template instead, which is what a headless agent CLI expects — it reads a
// prompt, not an event.
type HookStdin struct {
	Mode     string `json:"mode" db:"mode"`
	Template string `json:"template,omitempty" db:"template"`
}

// HookPost says what to do with the command's standard output.
//
// Kind "context" writes the output back into the domain as a context on the
// entity's project, which is what turns a hook from a notification into a
// reviewer whose verdict the worker reads on its next turn.
type HookPost struct {
	Kind  string `json:"kind" db:"kind"`
	Title string `json:"title" db:"title"`
}

// HookRun is one execution of a hook.
type HookRun struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	HookID     *uuid.UUID `json:"hook_id" db:"hook_id"`
	DeliveryID uuid.UUID  `json:"delivery_id" db:"delivery_id"`
	ProjectID  uuid.UUID  `json:"project_id" db:"project_id"`
	Event      string     `json:"event" db:"event"`
	EntityType string     `json:"entity_type" db:"entity_type"`
	EntityID   *uuid.UUID `json:"entity_id" db:"entity_id"`
	// The normalised outcome. The raw exit code sits beside it because the
	// two ecosystems disagree on what 2 means.
	Status     string     `json:"status" db:"status"`
	ExitCode   *int       `json:"exit_code" db:"exit_code"`
	Stdout     string     `json:"stdout" db:"stdout"`
	Stderr     string     `json:"stderr" db:"stderr"`
	DurationMS *int       `json:"duration_ms" db:"duration_ms"`
	StartedAt  time.Time  `json:"started_at" db:"started_at"`
	FinishedAt *time.Time `json:"finished_at" db:"finished_at"`
}

// Hook run statuses. A closed set, because the run log is read by a filter
// and an unknown status would be invisible to it.
const (
	HookRunSucceeded = "succeeded"
	HookRunFailed    = "failed"
	HookRunTimeout   = "timeout"
	HookRunCancelled = "cancelled"
	HookRunBlocked   = "blocked"
)

// Hook stdin modes.
const (
	HookStdinEvent  = "event"
	HookStdinPrompt = "prompt"
)

// Hook post kinds.
const HookPostKindContext = "context"

// CreateHookRequest is the body of a hook creation.
//
// The project is not part of it: the route carries the project id, and a
// body that could name a different one would be a way to create a hook in
// a project the caller did not address.
type CreateHookRequest struct {
	Event          string       `json:"event"`
	Matcher        *HookMatcher `json:"matcher"`
	Command        []string     `json:"command"`
	Env            HookEnv      `json:"env"`
	Stdin          HookStdin    `json:"stdin"`
	Post           *HookPost    `json:"post"`
	TimeoutSeconds int          `json:"timeout_seconds"`
	Enabled        *bool        `json:"enabled"`
}

// UpdateHookRequest is the body of a hook update.
//
// The event is not part of it either: a hook's event is what it reacts to,
// and changing that is a different operation from editing the command it
// runs.
type UpdateHookRequest struct {
	Matcher        *HookMatcher `json:"matcher"`
	Command        []string     `json:"command"`
	Env            HookEnv      `json:"env"`
	Stdin          HookStdin    `json:"stdin"`
	Post           *HookPost    `json:"post"`
	TimeoutSeconds int          `json:"timeout_seconds"`
	Enabled        *bool        `json:"enabled"`
}
