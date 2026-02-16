package cmd

func Init(args []string) error {
	fs := newFlagSet("init")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return nil
}
