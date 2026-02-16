package cmd

// Deploy handles deployment actions for the current service.
func Deploy(args []string) error {
	fs := newFlagSet("deploy")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	return nil
}

// Logs handles deployment log commands.
func Logs(args []string) error {
	fs := newFlagSet("logs")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	return nil
}

// Rollback handles rolling back a deployment.
func Rollback(args []string) error {
	fs := newFlagSet("rollback")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	return nil
}
