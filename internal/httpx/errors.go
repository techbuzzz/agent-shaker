// Package httpx contains HTTP transport helpers shared by handlers across
// packages: a uniform JSON error envelope and an errors.Is-driven status code
// mapper.
package httpx

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/techbuzzz/agent-shaker/internal/middleware"
	"github.com/techbuzzz/agent-shaker/internal/task"
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
	switch {
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
