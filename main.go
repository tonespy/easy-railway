// easy-railway is a CLI for managing Railway projects with local-first workflows.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tonespy/easy-railway/internal/cmd"
	"github.com/tonespy/easy-railway/internal/log"
)

var version = "dev"

type command struct {
	name string
	desc string
	run  func(args []string) error
}

var commands = []command{
	{"login", "Log in to Railway", cmd.Login},
	{"logout", "Log out and remove stored credentials", cmd.Logout},
	{"whoami", "Display the currently authenticated user", cmd.Whoami},
	{"init", "Initialize a new .easy-railway config", cmd.Init},
	{"env", "Manage environment variables (list, get, set, push, pull)", cmd.Env},
	{"service", "Manage services (list, create, delete, link)", cmd.Service},
	{"volume", "Manage volumes (list, create, delete)", cmd.Volume},
	{"deploy", "Deploy the current service", cmd.Deploy},
	{"logs", "View deployment logs", cmd.Logs},
	{"rollback", "Rollback to a previous deployment", cmd.Rollback},
	{"sync", "Manage git hook sync (install, uninstall)", cmd.Sync},
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	level, args := extractVerbosity(args)

	outLog := log.NewWithLevel(stdout, level)
	errLog := log.NewWithLevel(stderr, level)

	// Set the default logger so all packages pick up the verbosity level.
	log.Default = errLog

	if len(args) < 1 {
		printUsage(errLog)
		return 1
	}

	arg := args[0]

	switch arg {
	case "-h", "--help", "help":
		printUsage(outLog)
		return 0
	case "-v", "--version", "version":
		if _, err := fmt.Fprintln(stdout, "easy-railway", version); err != nil {
			errLog.Error("write version output: %v", err)
			return 1
		}
		return 0
	}

	for _, c := range commands {
		if c.name == arg {
			if err := c.run(args[1:]); err != nil {
				errLog.Error("%s", err)
				return 1
			}
			return 0
		}
	}

	errLog.Error("unknown command: %s", arg)
	printUsage(errLog)
	return 1
}

// extractVerbosity scans args for -v, -vv, --verbose and removes them.
// Returns the log level and the filtered args.
func extractVerbosity(args []string) (log.Level, []string) {
	level := log.LevelInfo

	// Check env var first.
	if v := os.Getenv("EASY_RAILWAY_LOG_LEVEL"); v != "" {
		switch strings.ToLower(v) {
		case "debug":
			level = log.LevelDebug
		case "trace":
			level = log.LevelTrace
		default:
			// Warn on invalid value but don't fail; just use info level.
			fmt.Fprintf(os.Stderr, "warning: invalid log level in EASY_RAILWAY_LOG_LEVEL: %q\n", v)
		}
	}

	// Flags override env var.
	filtered := make([]string, 0, len(args))

	for _, a := range args {
		switch a {
		case "-vv":
			level = log.LevelTrace
		case "--verbose":
			level = log.LevelDebug
		default:
			// Check for -v that isn't --version.
			if a == "-v" {
				// Ambiguous: -v could be version or verbose.
				// When -v appears with other command args, treat as verbose.
				// When -v is the only arg, treat as version (handled by switch in run).
				// We pass it through; the version switch catches it first.
				filtered = append(filtered, a)
			} else {
				filtered = append(filtered, a)
			}
		}
	}

	return level, filtered
}

func printUsage(l *log.Logger) {
	l.Print("easy-railway - A better Railway CLI\n\n")
	l.Print("Usage: easy-railway <command> [arguments]\n\n")
	l.Print("Commands:\n")

	maxLen := 0
	for _, c := range commands {
		if len(c.name) > maxLen {
			maxLen = len(c.name)
		}
	}

	for _, c := range commands {
		l.Print("  %-*s  %s\n", maxLen, c.name, c.desc)
	}

	l.Print("\nFlags:\n")
	l.Print("  -h, --help      Show this help message\n")
	l.Print("  -v, --version   Show version\n")
	l.Print("  --verbose       Enable debug output\n")
	l.Print("  -vv             Enable trace output (includes HTTP bodies)\n")
	l.Print("\nEnvironment variables:\n")
	l.Print("  EASY_RAILWAY_LOG_LEVEL   Set log level (info, debug, trace)\n")
	l.Print("\nRun 'easy-railway <command> --help' for more information on a command.\n")
}
