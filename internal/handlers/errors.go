package handlers

import (
	"net/http"
	"strings"

	"github.com/techbuzzz/agent-shaker/internal/httpx"
)

// handleNoStore writes a structured 503 response when the underlying
// typed store is unavailable (server running without a database). Called
// at the top of every handler method to short-circuit before the panic
// recovery middleware would otherwise catch a nil-pointer dereference.
func handleNoStore(w http.ResponseWriter, r *http.Request) {
	httpx.WriteError(w, r, httpx.ErrUnavailable)
}

// isNotFound matches the sentinel strings produced by the query helpers
// when a row is missing. Will be replaced by typed sentinels + errors.Is
// in a follow-up.
func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}
