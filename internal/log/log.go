package log

import (
	"fmt"
	"io"
	"os"
	"sync"
)

const (
	symbolInfo    = "•"
	symbolSuccess = "✓"
	symbolWarn    = "→"
	symbolError   = "✗"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
)

// Logger writes leveled, formatted messages to a writer.
type Logger struct {
	mu    sync.Mutex
	out   io.Writer
	color bool
}

// New creates a Logger that writes to out.
// Color is enabled only when out is a terminal and NO_COLOR is not set.
func New(out io.Writer) *Logger {
	return &Logger{
		out:   out,
		color: shouldColor(out),
	}
}

// Info prints a general informational message.
func (l *Logger) Info(format string, args ...any) {
	l.log(symbolInfo, colorCyan, format, args...)
}

// Success prints a success message.
func (l *Logger) Success(format string, args ...any) {
	l.log(symbolSuccess, colorGreen, format, args...)
}

// Warn prints a warning or in-progress message.
func (l *Logger) Warn(format string, args ...any) {
	l.log(symbolWarn, colorYellow, format, args...)
}

// Error prints an error message.
func (l *Logger) Error(format string, args ...any) {
	l.log(symbolError, colorRed, format, args...)
}

// Print writes plain text with no symbol or level prefix.
func (l *Logger) Print(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)

	l.mu.Lock()
	defer l.mu.Unlock()

	fmt.Fprint(l.out, msg)
}

func (l *Logger) log(symbol, color, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.color {
		fmt.Fprintf(l.out, "  %s%s%s %s\n", color, symbol, colorReset, msg)
	} else {
		fmt.Fprintf(l.out, "  %s %s\n", symbol, msg)
	}
}

func shouldColor(out io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := out.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// Default is the package-level logger writing to stderr.
var Default = New(os.Stderr)

// Package-level convenience functions that delegate to Default.

func Info(format string, args ...any)    { Default.Info(format, args...) }
func Success(format string, args ...any) { Default.Success(format, args...) }
func Warn(format string, args ...any)    { Default.Warn(format, args...) }
func Error(format string, args ...any)   { Default.Error(format, args...) }
func Print(format string, args ...any)   { Default.Print(format, args...) }
