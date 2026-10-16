package server

import (
	"net/http"

	"github.com/techbuzzz/agent-shaker/internal/publicurl"
)

// resolvePublicBaseURL returns the origin that external clients should use to
// reach this server, as observed on the request `r`.
//
// A thin delegation to internal/publicurl. The rules — BASE_URL first, then the
// forwarded headers, then the request's own hop — and the reasoning behind each
// live in that package, which the MCP config generator also depends on. This
// wrapper stays so the A2A call sites read the same as before and so this
// package's own tests keep testing the same seam.
func resolvePublicBaseURL(configured string, r *http.Request) string {
	return publicurl.Resolve(configured, r)
}
