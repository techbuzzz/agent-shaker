package handlers

import "net/http"

// muxVars returns a map[string]string populated from the path variables
// registered against the current request. It exists to bridge the old
// gorilla/mux `mux.Vars(r)["id"]` call sites to the stdlib
// `http.Request.PathValue` API without rewriting every call. New code should
// call r.PathValue("name") directly.
//
// Known path-variable names are read explicitly; stdlib ServeMux does not
// expose the list of registered variables, so this list must be kept in
// sync with the route registrations in cmd/server/routes.go.
func muxVars(r *http.Request) map[string]string {
	return map[string]string{
		"id":        r.PathValue("id"),
		"taskId":    r.PathValue("taskId"),
		"projectId": r.PathValue("projectId"),
	}
}
