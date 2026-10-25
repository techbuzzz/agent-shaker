//go:build windows

package hooks

import (
	"fmt"
	"os/exec"
	"strconv"
)

// configureProcessGroup is a no-op on Windows.
//
// Windows has no process groups in the POSIX sense and no setpgid, so the
// group kill is done by taskkill instead.
func configureProcessGroup(cmd *exec.Cmd) {}

// killProcessGroup kills the command and its descendants.
//
// taskkill with /T walks the tree, which is the Windows equivalent of
// killing the process group. It is a separate process rather than a syscall
// because the tree walk lives in taskkill itself.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	pid := strconv.Itoa(cmd.Process.Pid)
	kill := exec.Command("taskkill", "/T", "/F", "/PID", pid)
	if err := kill.Run(); err != nil {
		// The direct child is killed as a fallback: a failure to walk the
		// tree must not leave the command itself running.
		if killErr := cmd.Process.Kill(); killErr != nil {
			return fmt.Errorf("kill process tree: %w (fallback: %w)", err, killErr)
		}
	}
	return nil
}
