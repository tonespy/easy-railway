// Package prompt provides interactive CLI prompt utilities.
package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// Prompter handles interactive CLI input/output.
type Prompter struct {
	Stdin   io.Reader
	Stdout  io.Writer
	scanner *bufio.Scanner
}

// scan returns the shared scanner, creating it lazily.
func (p *Prompter) scan() *bufio.Scanner {
	if p.scanner == nil {
		p.scanner = bufio.NewScanner(p.Stdin)
	}

	return p.scanner
}

// Default is the package-level prompter using real stdin/stdout.
var Default = &Prompter{Stdin: os.Stdin, Stdout: os.Stdout}

// ReadLine prints the prompt and returns one trimmed line from Stdin.
func (p *Prompter) ReadLine(prompt string) (string, error) {
	if _, err := fmt.Fprint(p.Stdout, prompt); err != nil {
		return "", fmt.Errorf("write prompt: %w", err)
	}

	s := p.scan()
	if !s.Scan() {
		if err := s.Err(); err != nil {
			return "", fmt.Errorf("read line: %w", err)
		}

		return "", nil
	}

	return strings.TrimSpace(s.Text()), nil
}

// ReadSecret prints the prompt and reads input without terminal echo.
// Falls back to ReadLine when Stdin is not a terminal (e.g. in tests).
func (p *Prompter) ReadSecret(prompt string) (string, error) {
	f, ok := p.Stdin.(*os.File)
	if !ok {
		// Non-terminal fallback (tests) — reuse shared scanner.
		return p.ReadLine(prompt)
	}

	if _, err := fmt.Fprint(p.Stdout, prompt); err != nil {
		return "", fmt.Errorf("write prompt: %w", err)
	}

	raw, err := term.ReadPassword(int(f.Fd()))

	// Print newline after hidden input regardless of error.
	_, _ = fmt.Fprintln(p.Stdout)

	if err != nil {
		return "", fmt.Errorf("read secret: %w", err)
	}

	return strings.TrimSpace(string(raw)), nil
}

const maxSelectRetries = 3

// Select prints a numbered menu and returns the 0-based index of the chosen option.
// Accepts a 1-based number or a case-insensitive prefix/alias of an option label
// (e.g. "yes", "y", "no", "n"). Retries up to 3 times on invalid input.
func (p *Prompter) Select(prompt string, options []string) (int, error) {
	if _, err := fmt.Fprintln(p.Stdout, prompt); err != nil {
		return 0, fmt.Errorf("write prompt: %w", err)
	}

	for i, opt := range options {
		if _, err := fmt.Fprintf(p.Stdout, "  %d) %s\n", i+1, opt); err != nil {
			return 0, fmt.Errorf("write option: %w", err)
		}
	}

	s := p.scan()

	for attempt := range maxSelectRetries {
		if _, err := fmt.Fprint(p.Stdout, "Choice: "); err != nil {
			return 0, fmt.Errorf("write choice prompt: %w", err)
		}

		if !s.Scan() {
			if err := s.Err(); err != nil {
				return 0, fmt.Errorf("read choice: %w", err)
			}

			return 0, fmt.Errorf("select: no input")
		}

		text := strings.TrimSpace(s.Text())

		// Try numeric input first.
		if n, err := strconv.Atoi(text); err == nil {
			if n >= 1 && n <= len(options) {
				return n - 1, nil
			}
		}

		// Try matching option label by case-insensitive prefix.
		if idx, ok := matchOption(text, options); ok {
			return idx, nil
		}

		if attempt < maxSelectRetries-1 {
			if _, err := fmt.Fprintf(p.Stdout, "  Invalid choice %q. Please try again.\n", text); err != nil {
				return 0, fmt.Errorf("write retry prompt: %w", err)
			}
		}
	}

	return 0, fmt.Errorf("select: too many invalid attempts")
}

// matchOption matches input against option labels using case-insensitive comparison.
// It checks if the input matches the full label or a common alias (y/yes, n/no).
func matchOption(input string, options []string) (int, bool) {
	lower := strings.ToLower(input)

	// Direct label match (case-insensitive).
	for i, opt := range options {
		if strings.EqualFold(opt, input) {
			return i, true
		}
	}

	// Common aliases: match "y"/"yes" to first option starting with "Yes",
	// and "n"/"no" to first option starting with "No".
	for i, opt := range options {
		optLower := strings.ToLower(opt)
		switch lower {
		case "y", "yes":
			if strings.HasPrefix(optLower, "yes") {
				return i, true
			}
		case "n", "no":
			if strings.HasPrefix(optLower, "no") {
				return i, true
			}
		}
	}

	return 0, false
}

// Package-level convenience functions that delegate to Default.

// ReadLine prints the prompt and returns one trimmed line from stdin.
func ReadLine(prompt string) (string, error) { return Default.ReadLine(prompt) }

// ReadSecret prints the prompt and reads input without terminal echo.
func ReadSecret(prompt string) (string, error) { return Default.ReadSecret(prompt) }

// Select prints a numbered menu and returns the 0-based index of the chosen option.
func Select(prompt string, options []string) (int, error) { return Default.Select(prompt, options) }
