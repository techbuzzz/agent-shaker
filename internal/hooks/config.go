// Package hooks is the server-side hook layer: a command the server runs
// when a domain event fires.
//
// The design constraint that shapes every file here is that a hook is
// arbitrary command execution. Everything that follows — the argument vector
// instead of a shell string, the allow-list, the working-directory
// constraint, the redaction, the global off switch — exists because the
// alternative is a remote-code-execution surface with no boundary at all.
package hooks

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Config is the process-wide hook configuration.
type Config struct {
	// Enabled is the global switch. While it is false no hook runs, whatever
	// is configured. It is false by default: enabling the feature for the
	// process should not silently enable it for every project.
	Enabled bool

	// AllowedExecutables is the allow-list a hook's executable must appear
	// in. An empty list allows nothing, so a misconfigured deployment fails
	// closed rather than open.
	AllowedExecutables []string

	// MaxConcurrency bounds how many hooks run at once across the process.
	MaxConcurrency int

	// PerProjectConcurrency bounds how many hooks run at once for one
	// project, so a single project's slow command cannot starve another's.
	PerProjectConcurrency int

	// MaxOutputBytes caps how much of a command's output is captured. A
	// command that prints a megabyte must not be able to write a megabyte per
	// run into the database.
	MaxOutputBytes int

	// QueueSize is how many runs may be waiting for a worker. When the queue
	// is full a new run is dropped and logged rather than blocking the
	// publisher: a slow hook must never be able to slow down the agent that
	// caused the event.
	QueueSize int

	// DefaultTimeoutSeconds is the deadline applied to a hook that does not
	// carry its own.
	DefaultTimeoutSeconds int

	// WorkDir is the directory hook commands run in.
	//
	// It is configured rather than derived, because the project_repos table
	// records a repository URL and not a path on disk: there is no local
	// checkout for the server to know about. An empty value leaves the
	// command in the server's own working directory.
	WorkDir string
}

// DefaultConfig returns the configuration the server starts with.
//
// Hooks are off. The rest of the values are the ones that keep a
// misbehaving hook from taking the deployment with it.
func DefaultConfig() Config {
	return Config{
		Enabled:               false,
		AllowedExecutables:    nil,
		MaxConcurrency:        4,
		PerProjectConcurrency: 2,
		MaxOutputBytes:        64 * 1024,
		QueueSize:             64,
		DefaultTimeoutSeconds: 120,
	}
}

// ProjectContext is the project detail a hook template can read.
type ProjectContext struct {
	ID       string
	Name     string
	RepoPath string
}

// ProjectResolver supplies the project detail a hook template reads.
type ProjectResolver interface {
	ResolveProject(ctx context.Context, projectID uuid.UUID) (ProjectContext, error)
}

// SecretResolver resolves a secret reference to its value.
//
// A hook refers to a secret by name; the value is injected at run time and
// never written into the hook row, the logs, or the captured output. A
// resolver that knows nothing resolves nothing, and the hook runs without
// that variable.
type SecretResolver interface {
	Resolve(ctx context.Context, name string) (string, bool)
}

// Clock hands out the current time. It exists so a test can pin the
// timestamps in a hook run.
type Clock interface {
	Now() time.Time
}

// SystemClock is the real clock.
type SystemClock struct{}

// Now returns the current UTC time.
func (SystemClock) Now() time.Time { return time.Now().UTC() }
