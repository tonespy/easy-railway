package prompt

import (
	"bytes"
	"strings"
	"testing"
)

func newTestPrompter(input string) (*Prompter, *bytes.Buffer) {
	var out bytes.Buffer

	return &Prompter{
		Stdin:  strings.NewReader(input),
		Stdout: &out,
	}, &out
}

func TestReadLine(t *testing.T) {
	t.Parallel()

	p, out := newTestPrompter("hello world\n")
	got, err := p.ReadLine("Enter: ")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "hello world" {
		t.Fatalf("expected %q, got %q", "hello world", got)
	}

	if !strings.Contains(out.String(), "Enter: ") {
		t.Fatalf("expected prompt in output, got %q", out.String())
	}
}

func TestReadLineEmpty(t *testing.T) {
	t.Parallel()

	p, _ := newTestPrompter("\n")
	got, err := p.ReadLine("> ")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestReadLineTrimmed(t *testing.T) {
	t.Parallel()

	p, _ := newTestPrompter("  spaced  \n")
	got, err := p.ReadLine("> ")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "spaced" {
		t.Fatalf("expected %q, got %q", "spaced", got)
	}
}

func TestReadSecretNonTerminal(t *testing.T) {
	t.Parallel()

	p, _ := newTestPrompter("secret-value\n")
	got, err := p.ReadSecret("Password: ")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "secret-value" {
		t.Fatalf("expected %q, got %q", "secret-value", got)
	}
}

func TestSelectValid(t *testing.T) {
	t.Parallel()

	p, out := newTestPrompter("2\n")
	idx, err := p.Select("Pick one:", []string{"alpha", "beta", "gamma"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if idx != 1 {
		t.Fatalf("expected index 1, got %d", idx)
	}

	output := out.String()
	if !strings.Contains(output, "1) alpha") {
		t.Fatalf("expected option list in output, got %q", output)
	}

	if !strings.Contains(output, "2) beta") {
		t.Fatalf("expected option list in output, got %q", output)
	}
}

func TestSelectFirstOption(t *testing.T) {
	t.Parallel()

	p, _ := newTestPrompter("1\n")
	idx, err := p.Select("Pick:", []string{"only"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if idx != 0 {
		t.Fatalf("expected index 0, got %d", idx)
	}
}

func TestSelectYesAlias(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"yes", "yes\n"},
		{"Yes", "Yes\n"},
		{"YES", "YES\n"},
		{"y", "y\n"},
		{"Y", "Y\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, _ := newTestPrompter(tt.input)
			idx, err := p.Select("Continue?", []string{"No", "Yes"})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if idx != 1 {
				t.Fatalf("expected index 1 (Yes), got %d", idx)
			}
		})
	}
}

func TestSelectNoAlias(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"no", "no\n"},
		{"No", "No\n"},
		{"NO", "NO\n"},
		{"n", "n\n"},
		{"N", "N\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, _ := newTestPrompter(tt.input)
			idx, err := p.Select("Continue?", []string{"No", "Yes"})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if idx != 0 {
				t.Fatalf("expected index 0 (No), got %d", idx)
			}
		})
	}
}

func TestSelectLabelMatch(t *testing.T) {
	t.Parallel()

	p, _ := newTestPrompter("Account token\n")
	idx, err := p.Select("Type:", []string{"Account token", "Project token"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if idx != 0 {
		t.Fatalf("expected index 0, got %d", idx)
	}
}

func TestSelectRetryThenSucceed(t *testing.T) {
	t.Parallel()

	// First attempt invalid, second attempt valid.
	p, out := newTestPrompter("bad\n2\n")
	idx, err := p.Select("Pick:", []string{"a", "b"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if idx != 1 {
		t.Fatalf("expected index 1, got %d", idx)
	}

	if !strings.Contains(out.String(), "Invalid choice") {
		t.Fatalf("expected retry message, got %q", out.String())
	}
}

func TestSelectRetryExhausted(t *testing.T) {
	t.Parallel()

	// Three bad attempts.
	p, _ := newTestPrompter("bad\nworse\nworst\n")
	_, err := p.Select("Pick:", []string{"a", "b"})

	if err == nil {
		t.Fatal("expected error after 3 invalid attempts")
	}

	if !strings.Contains(err.Error(), "too many invalid attempts") {
		t.Fatalf("expected 'too many invalid attempts' error, got %q", err.Error())
	}
}

func TestSelectOutOfRangeRetries(t *testing.T) {
	t.Parallel()

	// Out of range, then valid.
	p, _ := newTestPrompter("5\n1\n")
	idx, err := p.Select("Pick:", []string{"a", "b"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if idx != 0 {
		t.Fatalf("expected index 0, got %d", idx)
	}
}

func TestSelectZeroRetries(t *testing.T) {
	t.Parallel()

	// Zero is out of range, then valid.
	p, _ := newTestPrompter("0\n2\n")
	idx, err := p.Select("Pick:", []string{"a", "b"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if idx != 1 {
		t.Fatalf("expected index 1, got %d", idx)
	}
}

func TestSelectNoInput(t *testing.T) {
	t.Parallel()

	p, _ := newTestPrompter("")
	_, err := p.Select("Pick:", []string{"a", "b"})

	if err == nil {
		t.Fatal("expected error for empty input")
	}

	if !strings.Contains(err.Error(), "no input") {
		t.Fatalf("expected no input error, got %q", err.Error())
	}
}
