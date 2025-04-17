package notifier

import (
	"github.com/kettari/location-bot/internal/config"
	middle "github.com/kettari/location-bot/internal/middleware"
	tele "gopkg.in/telebot.v4"
	"log/slog"
	"strconv"
	"strings"
)

type Report struct {
	conf     *config.Config
	schedule *Schedule
}

func NewReport(conf *config.Config, schedule *Schedule) *Report {
	return &Report{conf, schedule}
}

// ExecuteFullReport and send notification to recipients
//
// Destination format: chat_id_1,thread_id_1;chat_id_2,thread_id_2
func (r *Report) ExecuteFullReport(destination string) error {
	slog.Info("Executing joinable games full report")

	slog.Debug("Starting the bot")
	pref := tele.Settings{
		Token: r.conf.BotToken,
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		slog.Error("Unable to create bot processor object", "error", err)
		return err
	}
	// Add middleware
	b.Use(middle.Logger(slog.Default()))

	notificationChatIDs := strings.Split(destination, ";")
	for _, pair := range notificationChatIDs {
		dst := strings.Split(pair, ",")
		chatID, _ := strconv.ParseInt(dst[0], 10, 0)
		threadID, _ := strconv.Atoi(dst[1])
		recipient := tele.User{ID: chatID}
		slog.Debug("Sending notification", "recipient", recipient)

		notification, err := r.schedule.Format()
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

	slog.Debug("All notifications sent, update entities", "games_count", len(r.schedule.Games))
	if err = r.schedule.MarkAsNotified(); err != nil {
		return err
	}

	slog.Info("Full report sent")

	return nil
}

// ExecuteDeltaReport and send notification to recipients
//
// Destination format: chat_id_1,thread_id_1;chat_id_2,thread_id_2
func (r *Report) ExecuteDeltaReport(destination string) error {
	slog.Info("Executing joinable games delta report")

	slog.Debug("Starting the bot")
	pref := tele.Settings{
		Token: r.conf.BotToken,
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		slog.Error("Unable to create bot processor object", "error", err)
		return err
	}
	// Add middleware
	b.Use(middle.Logger(slog.Default()))

	notificationChatIDs := strings.Split(destination, ";")
	for _, game := range r.schedule.Games {
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

	slog.Debug("All notifications sent, update entities", "games_count", len(r.schedule.Games))
	if err = r.schedule.MarkAsNotified(); err != nil {
		return err
	}

	slog.Info("Delta report sent")

	return nil
}
