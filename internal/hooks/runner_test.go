package hooks

import (
	"context"
	"strings"
	"testing"
	"time"
)

// testConfig returns a configuration that permits the platform shell.
func testConfig() Config {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.AllowedExecutables = []string{shellExecutable}
	cfg.MaxOutputBytes = 1024
	return cfg
}

func TestRunnerRefusesAnExecutableOutsideTheAllowList(t *testing.T) {
	runner := NewRunner(testConfig())

	_, err := runner.Run(context.Background(), RunRequest{Argv: []string{"definitely-not-allowed"}})
	if err == nil {
		t.Fatal("an executable outside the allow-list must be refused")
	}
	if !strings.Contains(err.Error(), "allow-list") {
		t.Fatalf("the error must name the allow-list, got %q", err.Error())
	}
}

func TestRunnerRefusesAnEmptyCommand(t *testing.T) {
	runner := NewRunner(testConfig())

	if _, err := runner.Run(context.Background(), RunRequest{Argv: nil}); err == nil {
		t.Fatal("a hook with no command must be refused")
	}
}

func TestRunnerCapturesStandardOutput(t *testing.T) {
	runner := NewRunner(testConfig())

	result, err := runner.Run(context.Background(), RunRequest{Argv: echoArgv("hello-from-the-hook")})
	if err != nil {
		t.Fatalf("running the command: %v", err)
	}
	if !strings.Contains(result.Stdout, "hello-from-the-hook") {
		t.Fatalf("the output was %q, want it to contain the text", result.Stdout)
	}
}

func TestRunnerTruncatesOutputAtTheLimit(t *testing.T) {
	cfg := testConfig()
	cfg.MaxOutputBytes = 16
	runner := NewRunner(cfg)

	result, err := runner.Run(context.Background(), RunRequest{Argv: echoArgv(strings.Repeat("a", 500))})
	if err != nil {
		t.Fatalf("running the command: %v", err)
	}
	if len(result.Stdout) > cfg.MaxOutputBytes {
		t.Fatalf("the output is %d bytes, want at most %d", len(result.Stdout), cfg.MaxOutputBytes)
	}
}

func TestRunnerRedactsSecretsFromTheOutput(t *testing.T) {
	runner := NewRunner(testConfig())

	result, err := runner.Run(context.Background(), RunRequest{
		Argv:    echoArgv("the token is s3cr3t-value"),
		Secrets: []string{"s3cr3t-value"},
	})
	if err != nil {
		t.Fatalf("running the command: %v", err)
	}
	if strings.Contains(result.Stdout, "s3cr3t-value") {
		t.Fatal("the secret must not survive into the captured output")
	}
	if !strings.Contains(result.Stdout, RedactedValue) {
		t.Fatalf("the output was %q, want it to carry the redaction marker", result.Stdout)
	}
}

func TestRunnerReportsAFailedCommandAsAResult(t *testing.T) {
	runner := NewRunner(testConfig())

	result, err := runner.Run(context.Background(), RunRequest{Argv: shellArgv("exit 3")})
	if err != nil {
		t.Fatalf("a command that ran and failed is a result, not an error: %v", err)
	}
	if result.ExitCode != 3 {
		t.Fatalf("the exit code was %d, want 3", result.ExitCode)
	}
}

func TestRunnerKillsACommandThatOutlivesItsDeadline(t *testing.T) {
	runner := NewRunner(testConfig())

	start := time.Now()
	result, err := runner.Run(context.Background(), RunRequest{
		Argv:    sleepArgv(30),
		Timeout: 300 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("a timeout is a result, not an error: %v", err)
	}
	if !result.TimedOut {
		t.Fatal("the command must be reported as timed out")
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("the command ran for %s, so the process group was not killed", elapsed)
	}
}

func TestAllowsExecutableMatchesABareName(t *testing.T) {
	cfg := testConfig()

	if !cfg.AllowsExecutable(shellExecutable) {
		t.Fatal("a bare name in the allow-list must match")
	}
	if cfg.AllowsExecutable("something-else") {
		t.Fatal("a name outside the allow-list must not match")
	}
}
