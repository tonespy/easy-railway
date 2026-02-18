package main

import (
	"bytes"
	"strings"
	"testing"
)

func runTestCLI(args ...string) (int, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(args, &stdout, &stderr)

	return code, stdout.String(), stderr.String()
}

func TestRunHelpWritesToStdout(t *testing.T) {
	code, stdout, stderr := runTestCLI("--help")

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout, "Usage: easy-railway <command> [arguments]") {
		t.Fatalf("expected help usage in stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestRunVersionWritesToStdout(t *testing.T) {
	code, stdout, stderr := runTestCLI("--version")

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout, "easy-railway") {
		t.Fatalf("expected version output in stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestRunNoArgsWritesUsageToStderr(t *testing.T) {
	code, stdout, stderr := runTestCLI()

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Usage: easy-railway <command> [arguments]") {
		t.Fatalf("expected usage in stderr, got %q", stderr)
	}
}

func TestRunUnknownCommandWritesErrorToStderr(t *testing.T) {
	code, stdout, stderr := runTestCLI("unknown-command")

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "unknown command: unknown-command") {
		t.Fatalf("expected unknown command error in stderr, got %q", stderr)
	}
}

func TestRunLoginFlagParseError(t *testing.T) {
	code, stdout, stderr := runTestCLI("login", "--unknown-flag")

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "flag provided but not defined: -unknown-flag") {
		t.Fatalf("expected parse error in stderr, got %q", stderr)
	}
}

func TestExtractVerbosityDefault(t *testing.T) {
	level, args := extractVerbosity([]string{"login", "--api-key"})

	if level != 0 {
		t.Fatalf("expected LevelInfo (0), got %d", level)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
}

func TestExtractVerbosityVV(t *testing.T) {
	level, args := extractVerbosity([]string{"-vv", "login", "--api-key"})

	if level != 2 {
		t.Fatalf("expected LevelTrace (2), got %d", level)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args after stripping -vv, got %d: %v", len(args), args)
	}
}

func TestExtractVerbosityVerbose(t *testing.T) {
	level, args := extractVerbosity([]string{"--verbose", "whoami"})

	if level != 1 {
		t.Fatalf("expected LevelDebug (1), got %d", level)
	}
	if len(args) != 1 {
		t.Fatalf("expected 1 arg after stripping --verbose, got %d: %v", len(args), args)
	}
}
