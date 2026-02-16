package cmd

// Service handles service management commands.
func Service(args []string) error {
	fs := newFlagSet("service")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return errUsage("service", "list | create | delete | link")
	}

	switch fs.Arg(0) {
	case subcommandList:
		return serviceList(fs.Args()[1:])
	case subcommandCreate:
		return serviceCreate(fs.Args()[1:])
	case subcommandDelete:
		return serviceDelete(fs.Args()[1:])
	case "link":
		return serviceLink(fs.Args()[1:])
	default:
		return errUnknown("service", fs.Arg(0))
	}
}

func serviceList(_ []string) error   { return nil }
func serviceCreate(_ []string) error { return nil }
func serviceDelete(_ []string) error { return nil }
func serviceLink(_ []string) error   { return nil }
