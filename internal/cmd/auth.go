// Package cmd contains top-level command handlers for easy-railway.
package cmd

// Auth handles authentication commands.
func Auth(args []string) error {
	fs := newFlagSet("auth")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return errUsage("auth", "login | logout | whoami")
	}

	switch fs.Arg(0) {
	case "login":
		return authLogin(fs.Args()[1:])
	case "logout":
		return authLogout()
	case "whoami":
		return authWhoami()
	default:
		return errUnknown("auth", fs.Arg(0))
	}
}

func authLogin(_ []string) error { return nil }
func authLogout() error          { return nil }
func authWhoami() error          { return nil }
