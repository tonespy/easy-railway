// Package log provides lightweight structured-style CLI output helpers.
package log

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// Level controls which messages are printed.
type Level int

const (
	// LevelInfo is the default level — Info, Success, Warn, Error are printed.
	LevelInfo Level = iota
	// LevelDebug adds debug messages (-v).
	LevelDebug
	// LevelTrace adds trace messages with full HTTP bodies (-vv).
	LevelTrace
)

const (
	symbolInfo    = "•"
	symbolSuccess = "✓"
	symbolWarn    = "→"
	symbolError   = "✗"
	symbolDebug   = "⋯"
	symbolTrace   = "…"
)

const (
	colorReset   = "\033[0m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorRed     = "\033[31m"
	colorCyan    = "\033[36m"
	colorMagenta = "\033[35m"
	colorGray    = "\033[90m"
)

// Logger writes leveled, formatted messages to a writer.
type Logger struct {
	mu    sync.Mutex
	out   io.Writer
	color bool
	level Level
}

// New creates a Logger that writes to out at LevelInfo.
// Color is enabled only when out is a terminal and NO_COLOR is not set.
func New(out io.Writer) *Logger {
	return &Logger{
		out:   out,
		color: shouldColor(out),
		level: LevelInfo,
	}
}

// NewWithLevel creates a Logger that writes to out at the given level.
func NewWithLevel(out io.Writer, level Level) *Logger {
	return &Logger{
		out:   out,
		color: shouldColor(out),
		level: level,
	}
}

// SetLevel changes the logger's verbosity level.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.level = level
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

// Debug prints a debug message, visible only at LevelDebug or higher.
func (l *Logger) Debug(format string, args ...any) {
	l.mu.Lock()
	lvl := l.level
	l.mu.Unlock()

	if lvl < LevelDebug {
		return
	}

	l.log(symbolDebug, colorMagenta, format, args...)
}

// Trace prints a trace message, visible only at LevelTrace or higher.
func (l *Logger) Trace(format string, args ...any) {
	l.mu.Lock()
	lvl := l.level
	l.mu.Unlock()

	if lvl < LevelTrace {
		return
	}

	l.log(symbolTrace, colorGray, format, args...)
}

// Print writes plain text with no symbol or level prefix.
func (l *Logger) Print(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)

	l.mu.Lock()
	defer l.mu.Unlock()

	if _, err := fmt.Fprint(l.out, msg); err != nil {
		return
	}
}

func (l *Logger) log(symbol, color, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.color {
		if _, err := fmt.Fprintf(l.out, "  %s%s%s %s\n", color, symbol, colorReset, msg); err != nil {
			return
		}
	} else {
		if _, err := fmt.Fprintf(l.out, "  %s %s\n", symbol, msg); err != nil {
			return
		}
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

// Info writes an informational message via the default logger.
func Info(format string, args ...any) { Default.Info(format, args...) }

// Success writes a success message via the default logger.
func Success(format string, args ...any) { Default.Success(format, args...) }

// Warn writes a warning message via the default logger.
func Warn(format string, args ...any) { Default.Warn(format, args...) }

// Error writes an error message via the default logger.
func Error(format string, args ...any) { Default.Error(format, args...) }

// Debug writes a debug message via the default logger.
func Debug(format string, args ...any) { Default.Debug(format, args...) }

// Trace writes a trace message via the default logger.
func Trace(format string, args ...any) { Default.Trace(format, args...) }

// Print writes plain text via the default logger.
func Print(format string, args ...any) { Default.Print(format, args...) }
