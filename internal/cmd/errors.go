package cmd

import "fmt"

func errUsage(command, subcommands string) error {
	return fmt.Errorf("usage: easy-railway %s <%s>", command, subcommands)
}

func errUnknown(command, subcommand string) error {
	return fmt.Errorf("unknown %s command: %s", command, subcommand)
}
