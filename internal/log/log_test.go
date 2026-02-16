package log

import (
	"bytes"
	"strings"
	"testing"
)

func newTestLogger() (*Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	l := New(&buf)
	return l, &buf
}

func TestInfoOutput(t *testing.T) {
	l, buf := newTestLogger()
	l.Info("hello %s", "world")

	got := buf.String()
	if !strings.Contains(got, symbolInfo) {
		t.Fatalf("expected info symbol %q in output, got %q", symbolInfo, got)
	}
	if !strings.Contains(got, "hello world") {
		t.Fatalf("expected formatted message in output, got %q", got)
	}
}

func TestSuccessOutput(t *testing.T) {
	l, buf := newTestLogger()
	l.Success("deployed %s", "api")

	got := buf.String()
	if !strings.Contains(got, symbolSuccess) {
		t.Fatalf("expected success symbol %q in output, got %q", symbolSuccess, got)
	}
	if !strings.Contains(got, "deployed api") {
		t.Fatalf("expected formatted message in output, got %q", got)
	}
}

func TestWarnOutput(t *testing.T) {
	l, buf := newTestLogger()
	l.Warn("retrying in %d seconds", 5)

	got := buf.String()
	if !strings.Contains(got, symbolWarn) {
		t.Fatalf("expected warn symbol %q in output, got %q", symbolWarn, got)
	}
	if !strings.Contains(got, "retrying in 5 seconds") {
		t.Fatalf("expected formatted message in output, got %q", got)
	}
}

func TestErrorOutput(t *testing.T) {
	l, buf := newTestLogger()
	l.Error("connection failed")

	got := buf.String()
	if !strings.Contains(got, symbolError) {
		t.Fatalf("expected error symbol %q in output, got %q", symbolError, got)
	}
	if !strings.Contains(got, "connection failed") {
		t.Fatalf("expected message in output, got %q", got)
	}
}

func TestNoColorForNonTerminal(t *testing.T) {
	l, buf := newTestLogger()
	if l.color {
		t.Fatal("expected color to be false for bytes.Buffer writer")
	}

	l.Info("plain")
	got := buf.String()
	if strings.Contains(got, colorReset) {
		t.Fatalf("expected no ANSI codes in output, got %q", got)
	}
}

func TestOutputFormat(t *testing.T) {
	l, buf := newTestLogger()
	l.Info("test")

	got := buf.String()
	expected := "  " + symbolInfo + " test\n"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestMultipleWrites(t *testing.T) {
	l, buf := newTestLogger()
	l.Info("first")
	l.Error("second")

	got := buf.String()
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), got)
	}
}
