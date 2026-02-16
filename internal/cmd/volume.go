package cmd

// Volume handles volume management commands.
func Volume(args []string) error {
	fs := newFlagSet("volume")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return errUsage("volume", subcommandList+" | "+subcommandCreate+" | "+subcommandDelete)
	}

	switch fs.Arg(0) {
	case subcommandList:
		return volumeList(fs.Args()[1:])
	case subcommandCreate:
		return volumeCreate(fs.Args()[1:])
	case subcommandDelete:
		return volumeDelete(fs.Args()[1:])
	default:
		return errUnknown("volume", fs.Arg(0))
	}
}

func volumeList(_ []string) error   { return nil }
func volumeCreate(_ []string) error { return nil }
func volumeDelete(_ []string) error { return nil }
