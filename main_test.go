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

func TestRunSubcommandUsageError(t *testing.T) {
	code, stdout, stderr := runTestCLI("auth")

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "usage: easy-railway auth <login | logout | whoami>") {
		t.Fatalf("expected auth usage error in stderr, got %q", stderr)
	}
}

func TestRunSubcommandFlagParseError(t *testing.T) {
	code, stdout, stderr := runTestCLI("auth", "--unknown-flag")

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
