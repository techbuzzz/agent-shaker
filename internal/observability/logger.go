package observability

import (
	"log/slog"
	"os"
	"strings"
)

// NewLogger returns a slog.Logger writing JSON to stdout. The default level is
// info; override with the LOG_LEVEL env var (debug | info | warn | error).
func NewLogger() *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
