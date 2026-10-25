//go:build !windows

package hooks

import (
	"os/exec"
	"syscall"
)

// configureProcessGroup puts the command in its own process group.
//
// Without this, killing the command on a timeout leaves everything it
// started running: a shell that spawned a build, a test runner that spawned
// a fixture. The group is what makes the timeout mean what it says.
func configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup kills the command and everything in its group.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	// A negative pid addresses the process group.
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
