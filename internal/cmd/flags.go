package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

// errHelp is returned when the user requests help via -h or --help.
var errHelp = errors.New("help requested")

func parseFlags(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return errHelp
		}

		return fmt.Errorf("parse %s flags: %w", fs.Name(), err)
	}

	return nil
}
