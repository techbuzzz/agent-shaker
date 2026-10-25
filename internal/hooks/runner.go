package hooks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RunRequest is one execution of a hook.
type RunRequest struct {
	// Argv is the argument vector. The first element is the executable; the
	// rest are its arguments. There is no shell.
	Argv []string
	// Stdin is written to the command's standard input.
	Stdin []byte
	// WorkDir is the directory the command runs in. The dispatcher constrains
	// it to a registered repository path for the project.
	WorkDir string
	// Env is the command's entire environment. The server builds it rather
	// than inheriting its own: a hook that inherits the server's environment
	// inherits the server's database credentials.
	Env []string
	// Timeout is the deadline. Zero means the configured default.
	Timeout time.Duration
	// Secrets are the values that must not survive into the captured output.
	Secrets []string
}

// RunResult is what the command left behind.
type RunResult struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	DurationMS int
	TimedOut   bool
	Signaled   bool
	// Status is the normalised outcome: one of the models.HookRun* values.
	// The exit code beside it is the raw one, because the two vocabularies
	// disagree about what a code means.
	Status string
}

// Runner executes a hook command.
//
// It enforces the executable allow-list, gives the command a minimal
// environment rather than the server's own, and kills the whole process group
// when the deadline passes — a command that spawned a child must not leave
// that child running after the hook is over.
type Runner struct {
	cfg Config
}

// NewRunner returns a runner with the given configuration.
func NewRunner(cfg Config) *Runner {
	return &Runner{cfg: cfg}
}

// AllowsExecutable reports whether an executable may be run.
//
// An exact match always passes. A bare name — an entry with no directory
// separator — also matches by base name, which is how an operator writes
// "mcode" rather than a full path. The risk of that convenience is bounded
// by the fact that the runner sets the command's PATH itself, so the name
// resolves against the paths the operator chose.
func (c Config) AllowsExecutable(exe string) bool {
	if exe == "" {
		return false
	}

	base := filepath.Base(exe)
	for _, allowed := range c.AllowedExecutables {
		if allowed == exe {
			return true
		}
		if !strings.ContainsRune(allowed, filepath.Separator) && allowed == base {
			return true
		}
	}
	return false
}

// AllowsExecutable reports whether an executable may be run.
//
// The runner enforces the allow-list on every execution, and the handler
// enforces it again at creation time — a hook that names an executable the
// server would refuse should never be stored in the first place.
func (r *Runner) AllowsExecutable(exe string) bool {
	return r.cfg.AllowsExecutable(exe)
}

// Run executes the command and waits for it.
//
// A failure to start the command is an error; a command that ran and failed
// is a result. The distinction matters: a hook whose executable is missing is
// a configuration problem, and a hook whose command exited non-zero is the
// command's own business.
func (r *Runner) Run(ctx context.Context, req RunRequest) (RunResult, error) {
	if len(req.Argv) == 0 {
		return RunResult{}, errors.New("hook has no command")
	}
	if !r.cfg.AllowsExecutable(req.Argv[0]) {
		return RunResult{}, fmt.Errorf("executable %q is not in the allow-list", req.Argv[0])
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = time.Duration(r.cfg.DefaultTimeoutSeconds) * time.Second
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, req.Argv[0], req.Argv[1:]...)
	cmd.Dir = req.WorkDir
	cmd.Env = req.Env
	cmd.WaitDelay = 5 * time.Second
	configureProcessGroup(cmd)

	var stdout, stderr limitedBuffer
	stdout.limit = r.cfg.MaxOutputBytes
	stderr.limit = r.cfg.MaxOutputBytes
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if len(req.Stdin) > 0 {
		cmd.Stdin = bytes.NewReader(req.Stdin)
	}

	start := time.Now()
	err := r.wait(runCtx, cmd)
	elapsed := time.Since(start)

	result := RunResult{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		DurationMS: int(elapsed.Milliseconds()),
	}

	// The timeout fires first, then the process group is killed, and only
	// then does the wait return. Checking the context before the error is
	// what tells the two apart: a killed process also produces an error, but
	// it is not a failure of the command.
	if runCtx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		result.Signaled = true
		result.ExitCode = NormalizeExitCode(0, true)
		r.redactResult(&result, req.Secrets)
		return result, nil
	}

	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			r.redactResult(&result, req.Secrets)
			return result, fmt.Errorf("run hook command: %w", err)
		}

		// A command that ran and exited non-zero is not a signalled command.
		// Conflating the two would report every failure as 128, and the exit
		// code is the one thing a hook author has to work with.
		if exitErr.ProcessState != nil && exitErr.Exited() {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.Signaled = true
			result.ExitCode = NormalizeExitCode(0, true)
		}
	}

	r.redactResult(&result, req.Secrets)
	return result, nil
}

// wait runs the command and kills its process group when the context ends.
//
// exec.CommandContext kills only the direct child, so a command that spawned
// a grandchild would leave that grandchild running. The group kill is what
// makes the timeout mean what it says.
//
// The process is started in the calling goroutine on purpose. cmd.Run starts
// the process and publishes cmd.Process from inside the goroutine it runs
// in, and this function reads cmd.Process to kill the group — so calling
// cmd.Run here and reading the field from the caller is a data race on
// exactly the field the timeout path depends on. The race detector finds it
// in the timeout test every time.
func (r *Runner) wait(ctx context.Context, cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		_ = killProcessGroup(cmd)
		return <-done
	}
}

// redactResult removes the secret values from the captured output.
//
// Redaction happens after capture, not before: the command legitimately needs
// the secret to do its work, and what must not leak is the output that comes
// back.
func (r *Runner) redactResult(result *RunResult, secrets []string) {
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		result.Stdout = strings.ReplaceAll(result.Stdout, secret, RedactedValue)
		result.Stderr = strings.ReplaceAll(result.Stderr, secret, RedactedValue)
	}
}

// RedactedValue replaces a secret in captured output.
const RedactedValue = "***REDACTED***"

// limitedBuffer collects at most limit bytes.
//
// Write always reports the full length of the input, even when it discards
// part of it: reporting a short write would make the command fail with a
// broken pipe, and the hook would report a failure that was really the
// server's truncation.
type limitedBuffer struct {
	limit int
	buf   bytes.Buffer
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		return len(p), nil
	}
	if len(p) > remaining {
		b.buf.Write(p[:remaining])
		return len(p), nil
	}
	b.buf.Write(p)
	return len(p), nil
}

func (b *limitedBuffer) String() string {
	return b.buf.String()
}
