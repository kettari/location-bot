package console

type HelpCommand struct {
}

func NewHelpCommand() *HelpCommand {
	cmd := HelpCommand{}
	return &cmd
}

func (cmd *HelpCommand) Name() string {
	return "help"
}

func (cmd *HelpCommand) Description() string {
	return "dummy command for help"
}

func (cmd *HelpCommand) Run() error {
	return nil
}
