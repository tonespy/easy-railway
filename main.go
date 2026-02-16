package main

import (
	"fmt"
	"io"
	"os"

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
	{"auth", "Authenticate with Railway (login, logout, whoami)", cmd.Auth},
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
	outLog := log.New(stdout)
	errLog := log.New(stderr)

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
		fmt.Fprintln(stdout, "easy-railway", version)
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
	l.Print("\nRun 'easy-railway <command> --help' for more information on a command.\n")
}
