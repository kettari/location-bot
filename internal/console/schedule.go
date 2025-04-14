package console

type ScheduleCommand struct {
}

func NewScheduleCommand() *ScheduleCommand {
	cmd := ScheduleCommand{}
	return &cmd
}

func (cmd *ScheduleCommand) Name() string {
	return "schedule:fetch"
}

func (cmd *ScheduleCommand) Description() string {
	return "Dummy command for help"
}

func (cmd *ScheduleCommand) Run() error {
	return nil
}
