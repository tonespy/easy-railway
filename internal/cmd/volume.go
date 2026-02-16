package cmd

func Volume(args []string) error {
	fs := newFlagSet("volume")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return errUsage("volume", "list | create | delete")
	}

	switch fs.Arg(0) {
	case "list":
		return volumeList(fs.Args()[1:])
	case "create":
		return volumeCreate(fs.Args()[1:])
	case "delete":
		return volumeDelete(fs.Args()[1:])
	default:
		return errUnknown("volume", fs.Arg(0))
	}
}

func volumeList(args []string) error   { return nil }
func volumeCreate(args []string) error { return nil }
func volumeDelete(args []string) error { return nil }
