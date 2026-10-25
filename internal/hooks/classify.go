package hooks

import (
	"github.com/techbuzzz/agent-shaker/internal/models"
)

// Classify turns a process outcome into a normalised hook status.
//
// The exit code is kept verbatim beside the status because the two
// ecosystems disagree about what the codes mean. `mcode exec` uses 2 for an
// invalid invocation; Claude Code uses 2 for a blocking decision. A hook
// declares which convention it follows, and until it does, 2 is a failure —
// because the more common case is a command that could not run.
func Classify(exitCode int, timedOut bool, exitTwoIsBlocking bool) string {
	if timedOut {
		return models.HookRunTimeout
	}
	if exitCode == 0 {
		return models.HookRunSucceeded
	}
	if exitCode == 2 && exitTwoIsBlocking {
		return models.HookRunBlocked
	}
	if exitCode == 130 {
		return models.HookRunCancelled
	}
	return models.HookRunFailed
}

// NormalizeExitCode maps a process outcome onto the exit codes the runner
// reports.
//
// A process killed by a signal has no exit code of its own, so it is reported
// as 128 plus the signal number — the convention shells use. Without it a
// killed process and a process that exited 0 would look alike in the log.
func NormalizeExitCode(exitCode int, signaled bool) int {
	if signaled {
		return 128
	}
	return exitCode
}
