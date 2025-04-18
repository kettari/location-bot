package console

import (
	"github.com/kettari/location-bot/internal/config"
	"github.com/kettari/location-bot/internal/notifier"
	"github.com/kettari/location-bot/internal/storage"
	"log/slog"
)

type ScheduleReportDeltaCommand struct {
}

func NewScheduleReportDeltaCommand() *ScheduleReportDeltaCommand {
	cmd := ScheduleReportDeltaCommand{}
	return &cmd
}

func (cmd *ScheduleReportDeltaCommand) Name() string {
	return "schedule:report:delta"
}

func (cmd *ScheduleReportDeltaCommand) Description() string {
	return "sends delta notification to the Telegram bot"
}

func (cmd *ScheduleReportDeltaCommand) Run() error {
	slog.Info("Running delta report")

	conf := config.GetConfig()
	manager := storage.NewManager(conf.DbConnectionString)
	schedule := notifier.NewSchedule(manager)
	if err := schedule.LoadUnnotifiedEvents(); err != nil {
		return err
	}
	report := notifier.NewReport(conf, schedule)

	return report.ExecuteDeltaReport(conf.NotificationChatID)
}
