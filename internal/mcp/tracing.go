package mcp

import (
	"sort"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// mcpTracer is the tracer for the MCP surface. Using an explicit tracer
// instead of the package-global one keeps every span from this package under a
// single instrumentation scope, which is what a backend filters on.
var mcpTracer = otel.Tracer("github.com/techbuzzz/agent-shaker/internal/mcp")

// mcpProtocolVersion is the MCP revision this server implements.
//
// It is a constant because it is reported in three places (server info,
// initialize, and the tool-call span) and a hardcoded literal in each of them
// is exactly the kind of drift that goes unnoticed: a future revision bump has
// to change one line, not three, and the span attribute must not be able to
// disagree with the value the client was told.
const mcpProtocolVersion = "2024-11-05"

// argumentKeys returns the sorted top-level keys of a tool-call argument
// object, for logging.
//
// Tool arguments carry task titles, context bodies and standup text. Logging
// the shape instead of the payload keeps the log stream useful for debugging
// ("which tool, with which fields") without copying customer content into
// wherever logs are shipped. The result is a bounded list of field names.
func argumentKeys(arguments map[string]interface{}) []string {
	if len(arguments) == 0 {
		return nil
	}

	keys := make([]string, 0, len(arguments))
	for k := range arguments {
		keys = append(keys, k)
	}
	// Sorted so the log line is stable between identical calls; a map range is
	// randomised, which makes log diffing useless.
	sort.Strings(keys)
	return keys
}

var _ trace.Tracer = mcpTracer
