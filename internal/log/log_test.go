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

func TestDebugHiddenAtInfoLevel(t *testing.T) {
	l, buf := newTestLogger()
	l.Debug("should not appear")

	if buf.Len() != 0 {
		t.Fatalf("expected no output at LevelInfo, got %q", buf.String())
	}
}

func TestDebugVisibleAtDebugLevel(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithLevel(&buf, LevelDebug)
	l.Debug("visible")

	got := buf.String()
	if !strings.Contains(got, symbolDebug) {
		t.Fatalf("expected debug symbol %q in output, got %q", symbolDebug, got)
	}
	if !strings.Contains(got, "visible") {
		t.Fatalf("expected message in output, got %q", got)
	}
}

func TestTraceHiddenAtDebugLevel(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithLevel(&buf, LevelDebug)
	l.Trace("should not appear")

	if buf.Len() != 0 {
		t.Fatalf("expected no output at LevelDebug, got %q", buf.String())
	}
}

func TestTraceVisibleAtTraceLevel(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithLevel(&buf, LevelTrace)
	l.Trace("full body")

	got := buf.String()
	if !strings.Contains(got, symbolTrace) {
		t.Fatalf("expected trace symbol %q in output, got %q", symbolTrace, got)
	}
	if !strings.Contains(got, "full body") {
		t.Fatalf("expected message in output, got %q", got)
	}
}

func TestSetLevel(t *testing.T) {
	l, buf := newTestLogger()

	l.Debug("hidden")
	if buf.Len() != 0 {
		t.Fatalf("expected no debug output at LevelInfo, got %q", buf.String())
	}

	l.SetLevel(LevelDebug)
	l.Debug("now visible")

	got := buf.String()
	if !strings.Contains(got, "now visible") {
		t.Fatalf("expected debug message after SetLevel, got %q", got)
	}
}

func TestDebugAndTraceVisibleAtTraceLevel(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithLevel(&buf, LevelTrace)

	l.Debug("debug msg")
	l.Trace("trace msg")

	got := buf.String()
	if !strings.Contains(got, "debug msg") {
		t.Fatalf("expected debug message at trace level, got %q", got)
	}
	if !strings.Contains(got, "trace msg") {
		t.Fatalf("expected trace message at trace level, got %q", got)
	}
}
