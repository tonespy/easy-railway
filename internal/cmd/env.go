package cmd

func Env(args []string) error {
	fs := newFlagSet("env")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return errUsage("env", "list | get | set | push | pull")
	}

	switch fs.Arg(0) {
	case "list":
		return envList(fs.Args()[1:])
	case "get":
		return envGet(fs.Args()[1:])
	case "set":
		return envSet(fs.Args()[1:])
	case "push":
		return envPush(fs.Args()[1:])
	case "pull":
		return envPull(fs.Args()[1:])
	default:
		return errUnknown("env", fs.Arg(0))
	}
}

func envList(args []string) error { return nil }
func envGet(args []string) error  { return nil }
func envSet(args []string) error  { return nil }
func envPush(args []string) error { return nil }
func envPull(args []string) error { return nil }
