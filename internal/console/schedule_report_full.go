package console

import (
	"github.com/kettari/location-bot/internal/config"
	"github.com/kettari/location-bot/internal/schedule"
	"github.com/kettari/location-bot/internal/storage"
	"log/slog"
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
	slog.Info("running full report")

	conf := config.GetConfig()
	manager := storage.NewManager(conf.DbConnectionString)
	sch := schedule.NewSchedule(manager)
	if err := sch.LoadJoinableEvents(); err != nil {
		return err
	}

	return sch.ExecuteFullReport(conf.NotificationChatID)
}
