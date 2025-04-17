package notifier

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

func ExecuteReport(destination string) error {
	slog.Info("Requesting schedule")
	conf := config.GetConfig()

	manager := storage.NewManager(conf.DbConnectionString)
	if err := manager.Connect(); err != nil {
		return err
	}
	schedule := entity.NewSchedule()
	if result := manager.DB().
		Where(&entity.Game{Joinable: true}).
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

	// Destination format: chat_id_1,thread_id_1;chat_id_2,thread_id_2
	notificationChatIDs := strings.Split(destination, ";")
	for _, pair := range notificationChatIDs {
		dst := strings.Split(pair, ",")
		chatID, _ := strconv.ParseInt(dst[0], 10, 0)
		threadID, _ := strconv.Atoi(dst[1])
		recipient := tele.User{ID: chatID}
		slog.Debug("Sending notification", "recipient", recipient)

		notification, err := schedule.Format()
		if err != nil {
			return err
		}

		var message *tele.Message
		for _, txt := range notification {
			if message, err = b.Send(&recipient, txt, &tele.SendOptions{
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
