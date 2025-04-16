package console

import (
	"errors"
	"github.com/kettari/location-bot/internal/config"
	"github.com/kettari/location-bot/internal/entity"
	"github.com/kettari/location-bot/internal/storage"
	"gorm.io/gorm"
	"log/slog"
	"time"
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
	slog.Info("Requesting schedule")
	conf := config.GetConfig()

	manager := storage.NewManager(conf.DbConnectionString)
	if err := manager.Connect(); err != nil {
		return err
	}
	schedule := entity.NewSchedule()
	if result := manager.DB().Where(&entity.Game{Joinable: true}).
		Where("date > ?", time.Now()).Find(&schedule.Games); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			slog.Info("No joinable games found, exiting")
			return nil
		}
		return result.Error
	}

	slog.Info("Found joinable games", "games_count", len(schedule.Games))

	return nil
}
