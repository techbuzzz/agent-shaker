//go:build windows

package hooks

import "fmt"

// shellArgv builds a command that runs a shell script.
//
// The runner never uses a shell — that is the point of the argument vector —
// so the test has to name one explicitly.
func shellArgv(script string) []string {
	return []string{"cmd", "/c", script}
}

// shellExecutable is the name the allow-list has to carry for the shell tests.
const shellExecutable = "cmd"

// echoArgv builds a command that prints text.
//
// cmd's echo adds a space and a newline; the tests only look for a substring
// and a length bound, so the difference does not matter.
func echoArgv(text string) []string {
	return shellArgv(fmt.Sprintf("echo %s", text))
}

// sleepArgv builds a command that outlives a short deadline.
//
// `ping -n` is the portable way to wait on Windows without a console-bound
// `timeout`.
func sleepArgv(seconds int) []string {
	return shellArgv(fmt.Sprintf("ping -n %d 127.0.0.1 > nul", seconds+1))
}
