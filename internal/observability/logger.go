package observability

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

// NewLogger returns a slog logger writing JSON to stdout. The default level is
// info; override with the LOG_LEVEL env var (debug | info | warn | error).
//
// When tracing has been initialised via InitTracing, the otelslog bridge is
// used so every log line carries trace_id and span_id when a context is in
// scope. The level is enforced by a thin levelFilter wrapper (otelslog has
// no WithLevel option as of v0.20.x).
func NewLogger() *slog.Logger {
	level := parseLogLevel(os.Getenv("LOG_LEVEL"))
	if TracerEnabled() {
		return slog.New(&levelFilter{level: level, next: otelslog.NewHandler("agent-shaker")})
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// levelFilter wraps another slog.Handler and discards records whose level is
// below the configured threshold. Standard library slog does not provide a
// level-only wrapper prior to Go 1.26; we keep it minimal to avoid a
// third-party dependency.
type levelFilter struct {
	level slog.Level
	next  slog.Handler
}

func (l *levelFilter) Enabled(_ context.Context, level slog.Level) bool {
	return level >= l.level
}

func (l *levelFilter) Handle(ctx context.Context, r slog.Record) error {
	return l.next.Handle(ctx, r)
}

func (l *levelFilter) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelFilter{level: l.level, next: l.next.WithAttrs(attrs)}
}

func (l *levelFilter) WithGroup(name string) slog.Handler {
	return &levelFilter{level: l.level, next: l.next.WithGroup(name)}
}
