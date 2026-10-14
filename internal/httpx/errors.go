// Package httpx contains HTTP transport helpers shared by handlers across
// packages: a uniform JSON error envelope and an errors.Is-driven status code
// mapper.
package httpx

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/techbuzzz/agent-shaker/internal/middleware"
	"github.com/techbuzzz/agent-shaker/internal/task"
)

// Sentinel errors that handlers wrap into fmt.Errorf("context: %w", ...)
// when they want a specific HTTP status. classify() maps these via errors.Is.
var (
	// ErrBadRequest → 400. Use for malformed JSON, parse errors, and
	// user-input validation failures the validator surfaced.
	ErrBadRequest = errors.New("bad request")

	// ErrUnauthorized → 401. Reserved; auth is out of scope today.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden → 403. Reserved for future role-based access control.
	ErrForbidden = errors.New("forbidden")

	// ErrNotFound → 404. Use when a resource lookup found nothing.
	ErrNotFound = errors.New("not found")

	// ErrUnavailable → 503. Use when a backing service (database, cache,
	// upstream API) is not reachable. The handler MUST NOT panic; this
	// sentinel exists so the no-DB path can degrade gracefully instead of
	// tripping middleware.Recovery.
	ErrUnavailable = errors.New("service unavailable")
)

// ErrorEnvelope is the wire shape returned for every non-2xx response.
// {"error": {"code": "...", "message": "...", "request_id": "..."}}
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody is the inner object of ErrorEnvelope.
type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// WriteError writes a JSON error envelope using a status code derived from
// err. The mapping favours domain sentinels first, then transport sentinels
// (sql.ErrNoRows), and falls back to 500 for everything else.
//
// If err is nil, WriteError does nothing and returns. Callers should treat the
// request as successful in that case; this function is intended to be used in
// error branches only.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	code, msg, httpStatus := classify(err)
	rid := middleware.RequestIDFromContext(r.Context())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(ErrorEnvelope{
		Error: ErrorBody{
			Code:      code,
			Message:   msg,
			RequestID: rid,
		},
	})
}

// classify maps a Go error to an HTTP status code and a stable error code
// identifier that clients can switch on.
func classify(err error) (code, msg string, httpStatus int) {
	// The default message is the error's own text with the wrapped sentinel's
	// prefix removed.
	//
	// It used to be the empty string. Handlers wrap these sentinels exactly as
	// documented — fmt.Errorf("%w: progress_pct must be between 0 and 100",
	// ErrBadRequest) — and every one of those responses went out with
	// "message": "". The status was right, the code was right, and the part a
	// person or an agent actually reads was missing. A UI that renders
	// error.message showed an empty red box.
	msg = withoutSentinelPrefix(err)
	switch {
	// Phase-1 milestone close guard. errMilestoneOpenTasks.Error() contains
	// "milestone has N open task(s)" — match on the prefix so the wire stays
	// stable without exporting the struct from the handlers package.
	case strings.HasPrefix(err.Error(), "milestone has ") && strings.Contains(err.Error(), "open task"):
		return "milestone_open_tasks", err.Error(), http.StatusConflict

	// Phase-3 global_contexts conflict (title already taken in this scope).
	case strings.Contains(err.Error(), "global context with title") && strings.Contains(err.Error(), "already exists"):
		return "global_context_conflict", err.Error(), http.StatusConflict

	case errors.Is(err, ErrBadRequest):
		return "bad_request", msg, http.StatusBadRequest
	case errors.Is(err, ErrUnauthorized):
		return "unauthorized", msg, http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return "forbidden", msg, http.StatusForbidden
	case errors.Is(err, ErrNotFound):
		return "not_found", msg, http.StatusNotFound
	case errors.Is(err, ErrUnavailable):
		return "unavailable", msg, http.StatusServiceUnavailable
	case errors.Is(err, task.ErrTaskNotFound):
		return "task_not_found", "the requested task does not exist", http.StatusNotFound
	case errors.Is(err, task.ErrTaskTerminal):
		return "task_terminal", "task is already in a terminal state", http.StatusConflict
	case errors.Is(err, sql.ErrNoRows):
		return "not_found", "resource not found", http.StatusNotFound
	case errors.Is(err, context.Canceled):
		return "canceled", "request canceled", 499 // nginx-style "client closed request"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout", "request deadline exceeded", http.StatusGatewayTimeout
	case errors.Is(err, sql.ErrConnDone):
		return "db_unavailable", "database connection unavailable", http.StatusServiceUnavailable
	}

	// Fallback: surface a generic 500 with the error's message so operators can
	// diagnose from logs, but never leak it to unauthenticated callers in
	// production. (Auth is out of scope; this matches current behaviour.)
	return "internal", err.Error(), http.StatusInternalServerError
}

// withoutSentinelPrefix strips a leading "<sentinel>: " from a wrapped error.
//
// The sentinels are transport vocabulary, not something to show a caller:
// "not found: agent 8f2… not found" is the same fact twice, and the duplicate is
// the part that leaks the implementation. What is left after the prefix is the
// part the handler wrote for the reader.
func withoutSentinelPrefix(err error) string {
	text := err.Error()
	for _, sentinel := range []error{ErrBadRequest, ErrUnauthorized, ErrForbidden, ErrNotFound, ErrUnavailable} {
		prefix := sentinel.Error() + ": "
		if strings.HasPrefix(text, prefix) {
			return strings.TrimPrefix(text, prefix)
		}
	}
	return text
}

// WriteValidationError writes a 400 for a rejected request body.
//
// It exists because the alternative was a 500. Handlers wrapped the validator's
// error as fmt.Errorf("validation: %w", err) and handed it to WriteError, which
// classifies by sentinel — an unwrapped error is "unknown", so a caller that
// posted a task without an agent_id got a 500 and a server-error page, for what
// is a client mistake with a perfectly good explanation attached.
//
// The message is kept verbatim, because the validator's text ("agent_id is
// required") is the part the caller needs; the prefix is dropped so it does not
// leak the internal naming of the layer that produced it.
func WriteValidationError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}
	WriteError(w, r, fmt.Errorf("%w: %s", ErrBadRequest, withoutPrefix(err.Error(), "validation: ")))
}

// withoutPrefix removes a leading "prefix" if present.
func withoutPrefix(text, prefix string) string {
	return strings.TrimPrefix(text, prefix)
}

// WriteJSON writes payload as JSON with the supplied status code. Encode
// errors are logged at error level (they typically mean the client has
// already disconnected) rather than returned, since the response has
// already been partially written.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// No context: WriteJSON has no *http.Request and is called from every
		// handler in the codebase. Threading one through all of them to enrich
		// a swallowed encode error is not worth the signature churn — the
		// response is already partially written and the client has usually
		// disconnected. Structured logging still beats the unstructured default
		// logger this replaced.
		slog.Error("encode response", "error", err, "status", status)
	}
}
