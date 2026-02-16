package cmd

// Sync handles git hook synchronization commands.
func Sync(args []string) error {
	fs := newFlagSet("sync")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return errUsage("sync", "install | uninstall")
	}

	switch fs.Arg(0) {
	case "install":
		return syncInstall(fs.Args()[1:])
	case "uninstall":
		return syncUninstall(fs.Args()[1:])
	default:
		return errUnknown("sync", fs.Arg(0))
	}
}

func syncInstall(_ []string) error   { return nil }
func syncUninstall(_ []string) error { return nil }
