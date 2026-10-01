package server

import "net/http"

// a2aVars is a thin shim that bridges the old gorilla/mux mux.Vars(r)["x"]
// pattern to the stdlib http.Request.PathValue API for the A2A routes.
// New code should call r.PathValue("name") directly.
func a2aVars(r *http.Request) map[string]string {
	return map[string]string{
		"taskId":     r.PathValue("taskId"),
		"projectId":  r.PathValue("projectId"),
		"id":         r.PathValue("id"),
		"artifactId": r.PathValue("artifactId"),
	}
}
