package cmd

// Init initializes easy-railway configuration in the current repository.
func Init(args []string) error {
	fs := newFlagSet("init")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	return nil
}
