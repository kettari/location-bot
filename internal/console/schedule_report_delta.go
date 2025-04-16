package console

import (
	"errors"
	"github.com/kettari/location-bot/internal/config"
	"github.com/kettari/location-bot/internal/entity"
	middle "github.com/kettari/location-bot/internal/middleware"
	"github.com/kettari/location-bot/internal/storage"
	tele "gopkg.in/telebot.v4"
	"gorm.io/gorm"
	"log/slog"
	"strconv"
	"strings"
	"time"
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
	slog.Info("Requesting schedule")
	conf := config.GetConfig()

	manager := storage.NewManager(conf.DbConnectionString)
	if err := manager.Connect(); err != nil {
		return err
	}
	schedule := entity.NewSchedule()
	if result := manager.DB().
		Where(&entity.Game{Joinable: true}).
		Where("notification_sent = ?", false).
		Where("date > ?", time.Now()).
		Order("date ASC").
		Find(&schedule.Games); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			slog.Info("No joinable games found, exiting")
			return nil
		}
		return result.Error
	}

	slog.Info("Found joinable games", "games_count", len(schedule.Games))

	slog.Info("Starting the bot")
	pref := tele.Settings{
		Token: conf.BotToken,
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		slog.Error("Unable to create bot processor object", "error", err)
		return err
	}
	// Add middleware
	b.Use(middle.Logger(slog.Default()))

	notificationChatIDs := strings.Split(conf.NotificationChatID, ";")
	for _, game := range schedule.Games {
		notification, err := game.Format()
		if err != nil {
			return err
		}

		for _, pair := range notificationChatIDs {
			dst := strings.Split(pair, ",")
			chatID, _ := strconv.ParseInt(dst[0], 10, 0)
			threadID, _ := strconv.Atoi(dst[1])
			recipient := tele.User{ID: chatID}
			slog.Debug("Sending notification", "recipient", recipient)

			var message *tele.Message
			if message, err = b.Send(&recipient, notification, &tele.SendOptions{
				ParseMode: tele.ModeHTML, ThreadID: threadID, DisableWebPagePreview: true}); err != nil {
				slog.Error("Failed to send notification")
				return err
			}
			slog.Debug("Notification sent", "message", message)
		}
	}

	slog.Info("All notifications sent, update entities")

	// Mark games unchanged and notified
	for k, _ := range schedule.Games {
		schedule.Games[k].Changed = false
		schedule.Games[k].NotificationSent = true
		if err = manager.DB().Save(&schedule.Games[k]).Error; err != nil {
			return err
		}
	}

	slog.Info("Entities saved")

	return nil
}
