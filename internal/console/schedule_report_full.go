package console

import (
	"github.com/kettari/location-bot/internal/config"
	"github.com/kettari/location-bot/internal/notifier"
)

type ScheduleReportFullCommand struct {
}

func NewScheduleReportFullCommand() *ScheduleReportFullCommand {
	cmd := ScheduleReportFullCommand{}
	return &cmd
}

func (cmd *ScheduleReportFullCommand) Name() string {
	return "schedule:report:full"
}

func (cmd *ScheduleReportFullCommand) Description() string {
	return "sends full notification to the Telegram bot"
}

func (cmd *ScheduleReportFullCommand) Run() error {
	conf := config.GetConfig()
	return notifier.ExecuteReport(conf.NotificationChatID)
}
