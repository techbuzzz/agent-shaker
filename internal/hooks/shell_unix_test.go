//go:build !windows

package hooks

import "fmt"

// shellArgv builds a command that runs a shell script.
//
// The runner never uses a shell — that is the point of the argument vector —
// so the test has to name one explicitly. Naming it in the argument vector is
// exactly how an operator would run a script from a hook.
func shellArgv(script string) []string {
	return []string{"sh", "-c", script}
}

// shellExecutable is the name the allow-list has to carry for the shell tests.
const shellExecutable = "sh"

// echoArgv builds a command that prints text without a trailing newline.
func echoArgv(text string) []string {
	return shellArgv(fmt.Sprintf("printf '%%s' %q", text))
}

// sleepArgv builds a command that outlives a short deadline.
func sleepArgv(seconds int) []string {
	return shellArgv(fmt.Sprintf("sleep %d", seconds))
}
