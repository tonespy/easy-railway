package cmd

func Deploy(args []string) error {
	fs := newFlagSet("deploy")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return nil
}

func Logs(args []string) error {
	fs := newFlagSet("logs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return nil
}

func Rollback(args []string) error {
	fs := newFlagSet("rollback")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return nil
}
