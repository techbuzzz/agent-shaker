package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/events"
	"github.com/techbuzzz/agent-shaker/internal/hooks"
	"github.com/techbuzzz/agent-shaker/internal/httpx"
	"github.com/techbuzzz/agent-shaker/internal/middleware"
	"github.com/techbuzzz/agent-shaker/internal/models"
)

// HookHandler is the human-facing surface of the hook layer.
//
// A hook is arbitrary command execution, so the mutating routes are gated
// behind the admin scope (see cmd/server/routes.go) rather than trusted to
// whoever holds a write key.
type HookHandler struct {
	store      *queries.HooksStore
	dispatcher *hooks.Dispatcher
	runner     *hooks.Runner
}

// NewHookHandler returns a handler over the hook store and dispatcher.
func NewHookHandler(store *queries.HooksStore, dispatcher *hooks.Dispatcher, runner *hooks.Runner) *HookHandler {
	return &HookHandler{store: store, dispatcher: dispatcher, runner: runner}
}

// ListHooks returns the hooks configured for a project.
func (h *HookHandler) ListHooks(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}

	projectID, err := uuid.Parse(projectPathID(r))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid project id", httpx.ErrBadRequest))
		return
	}

	hooks, err := h.store.ListHooks(r.Context(), projectID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"hooks": hooks})
}

// CreateHook stores a new hook.
func (h *HookHandler) CreateHook(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}

	projectID, err := uuid.Parse(projectPathID(r))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid project id", httpx.ErrBadRequest))
		return
	}

	var req models.CreateHookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: malformed body", httpx.ErrBadRequest))
		return
	}

	if err := h.validate(req.Event, req.Command, req.Matcher, req.Stdin, req.Post, req.TimeoutSeconds); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	now := time.Now().UTC()
	hook := &models.Hook{
		ID:             uuid.New(),
		ProjectID:      projectID,
		Event:          req.Event,
		Matcher:        req.Matcher,
		Command:        req.Command,
		Env:            req.Env,
		Stdin:          req.Stdin,
		Post:           req.Post,
		TimeoutSeconds: h.timeout(req.TimeoutSeconds),
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if req.Enabled != nil {
		hook.Enabled = *req.Enabled
	}
	if principal, ok := middleware.PrincipalFromContext(r.Context()); ok {
		creator := principal.ID
		hook.CreatedBy = &creator
	}

	if err := h.store.CreateHook(r.Context(), hook); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	// Creating a hook turns the hook layer on for this project.
	//
	// The alternative is a separate switch, and a separate switch that nobody
	// knows about is a hook that silently never fires. Creating a hook IS the
	// decision to run commands for this project; the closed default lives in
	// the fact that a project with no hooks has nothing to run, and in the
	// explicit switch below for turning it back off.
	if err := h.store.SetHooksEnabled(r.Context(), projectID, true); err != nil {
		slog.WarnContext(r.Context(), "hook created but the project opt-in could not be set",
			"error", err,
			"project_id", projectID,
			"hook_id", hook.ID,
		)
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"hook": hook})
}

// GetHookSettings reports whether the hook layer is enabled for a project.
//
// The switch is per project: creating a hook turns it on, and this endpoint
// is how the interface learns the state it did not set itself.
func (h *HookHandler) GetHookSettings(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}

	projectID, err := uuid.Parse(projectPathID(r))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid project id", httpx.ErrBadRequest))
		return
	}

	enabled, err := h.store.HooksEnabled(r.Context(), projectID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"project_id": projectID,
		"enabled":    enabled,
	})
}

// UpdateHookSettings turns the hook layer on or off for one project.
//
// It is the explicit switch: creating a hook turns the project on, and this
// turns it back off without deleting the hooks.
func (h *HookHandler) UpdateHookSettings(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}

	projectID, err := uuid.Parse(projectPathID(r))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid project id", httpx.ErrBadRequest))
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: malformed body", httpx.ErrBadRequest))
		return
	}

	if err := h.store.SetHooksEnabled(r.Context(), projectID, req.Enabled); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"project_id": projectID,
		"enabled":    req.Enabled,
	})
}

// UpdateHook edits a hook.
func (h *HookHandler) UpdateHook(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}

	hookID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid hook id", httpx.ErrBadRequest))
		return
	}

	existing, err := h.store.GetHook(r.Context(), hookID)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: hook not found", httpx.ErrNotFound))
		return
	}

	var req models.UpdateHookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: malformed body", httpx.ErrBadRequest))
		return
	}

	if err := h.validate(existing.Event, req.Command, req.Matcher, req.Stdin, req.Post, req.TimeoutSeconds); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	existing.Matcher = req.Matcher
	existing.Command = req.Command
	existing.Env = req.Env
	existing.Stdin = req.Stdin
	existing.Post = req.Post
	existing.TimeoutSeconds = h.timeout(req.TimeoutSeconds)
	existing.UpdatedAt = time.Now().UTC()
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}

	if err := h.store.UpdateHook(r.Context(), existing); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"hook": existing})
}

// DeleteHook removes a hook. Its runs survive.
func (h *HookHandler) DeleteHook(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}

	hookID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid hook id", httpx.ErrBadRequest))
		return
	}

	if err := h.store.DeleteHook(r.Context(), hookID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListHookRuns returns the recent runs of a project's hooks.
func (h *HookHandler) ListHookRuns(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}

	projectID, err := uuid.Parse(projectPathID(r))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid project id", httpx.ErrBadRequest))
		return
	}

	limit := 50
	runs, err := h.store.ListRuns(r.Context(), projectID, limit)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

// TestHook runs a hook against a synthetic payload.
//
// It does not touch real work: the event is fabricated, so a command that
// reacts to "task completed" sees a task that never existed. That is the
// point — a hook that can only be validated by firing it at real work is a
// hook nobody will enable.
func (h *HookHandler) TestHook(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}

	hookID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid hook id", httpx.ErrBadRequest))
		return
	}

	hook, err := h.store.GetHook(r.Context(), hookID)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: hook not found", httpx.ErrNotFound))
		return
	}

	run, err := h.dispatcher.TestHook(r.Context(), *hook, syntheticEvent(*hook))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"run": run})
}

// syntheticEvent builds the event a test run acts on.
func syntheticEvent(hook models.Hook) events.Event {
	return events.New(hook.Event, hook.ProjectID, events.Options{
		EntityType: events.EntityTask,
		EntityID:   uuid.New(),
		ActorType:  events.ActorAgent,
		ActorID:    uuid.New(),
		Payload: map[string]any{
			events.PayloadTitle:     "Synthetic task for a hook test",
			events.PayloadStatus:    "done",
			events.PayloadOutput:    "This run was a test. Nothing real happened.",
			events.PayloadActorName: "hook-test",
			events.PayloadActorRole: "tester",
		},
	})
}

// validate rejects a hook that could never work.
//
// The checks are at creation time on purpose. A hook that is bound to an
// event the server never publishes, or that names an executable outside the
// allow-list, is not a hook that is temporarily misconfigured — it is a hook
// that will never run, and it should never have been saved.
func (h *HookHandler) validate(
	eventType string,
	command []string,
	matcher *models.HookMatcher,
	stdin models.HookStdin,
	post *models.HookPost,
	timeoutSeconds int,
) error {
	if !events.IsKnown(eventType) {
		return fmt.Errorf("%w: unknown event %q", httpx.ErrBadRequest, eventType)
	}

	if len(command) == 0 {
		return fmt.Errorf("%w: a hook needs a command", httpx.ErrBadRequest)
	}
	if h.runner != nil && !h.runner.AllowsExecutable(command[0]) {
		return fmt.Errorf("%w: executable %q is not in the allow-list", httpx.ErrBadRequest, command[0])
	}

	if matcher != nil {
		if _, err := hooks.CompileMatcher(matcher); err != nil {
			return fmt.Errorf("%w: %s", httpx.ErrBadRequest, err.Error())
		}
	}

	switch stdin.Mode {
	case models.HookStdinEvent, models.HookStdinPrompt, "":
	default:
		return fmt.Errorf("%w: unknown stdin mode %q", httpx.ErrBadRequest, stdin.Mode)
	}
	if stdin.Mode == models.HookStdinPrompt && stdin.Template == "" {
		return fmt.Errorf("%w: the prompt stdin mode needs a template", httpx.ErrBadRequest)
	}

	if post != nil && post.Kind != models.HookPostKindContext {
		return fmt.Errorf("%w: unknown post kind %q", httpx.ErrBadRequest, post.Kind)
	}

	if timeoutSeconds < 0 || timeoutSeconds > 3600 {
		return fmt.Errorf("%w: timeout_seconds must be between 1 and 3600", httpx.ErrBadRequest)
	}

	return nil
}

// timeout applies the default when the caller did not choose one.
func (h *HookHandler) timeout(seconds int) int {
	if seconds <= 0 {
		return hooks.DefaultConfig().DefaultTimeoutSeconds
	}
	return seconds
}
