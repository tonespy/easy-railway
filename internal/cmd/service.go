package cmd

func Service(args []string) error {
	fs := newFlagSet("service")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return errUsage("service", "list | create | delete | link")
	}

	switch fs.Arg(0) {
	case "list":
		return serviceList(fs.Args()[1:])
	case "create":
		return serviceCreate(fs.Args()[1:])
	case "delete":
		return serviceDelete(fs.Args()[1:])
	case "link":
		return serviceLink(fs.Args()[1:])
	default:
		return errUnknown("service", fs.Arg(0))
	}
}

func serviceList(args []string) error   { return nil }
func serviceCreate(args []string) error { return nil }
func serviceDelete(args []string) error { return nil }
func serviceLink(args []string) error   { return nil }
