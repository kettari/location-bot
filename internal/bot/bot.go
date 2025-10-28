package bot

import (
	"log/slog"
	"strconv"
	"strings"

	"github.com/kettari/location-bot/internal/config"
	"github.com/kettari/location-bot/internal/entity"
	tele "gopkg.in/telebot.v4"
)

type Bot struct {
	bot         *tele.Bot
	destination []Recipient
}

type Recipient struct {
	User     tele.User
	ThreadID int
}

// CreateBot returns [MessageDispatcher] object to send notifications
//   - recipients is a string "chat_id1,thread_id1;chat_id2,thread_id2"
func CreateBot(recipients string) (entity.MessageDispatcher, error) {
	conf := config.GetConfig()
	pref := tele.Settings{
		Token: conf.BotToken,
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		slog.Error("unable to create bot processor object", "error", err)
		return nil, err
	}
	return &Bot{
		bot:         b,
		destination: prepareDestination(recipients),
	}, nil
}

// prepareDestination parses configuration files and prepares array with [gopkg.in/telebot.v4.User]
func prepareDestination(recipients string) []Recipient {
	result := make([]Recipient, 0)
	notificationChatIDs := strings.Split(recipients, ";")
	for _, pair := range notificationChatIDs {
		dst := strings.Split(pair, ",")
		chatID, _ := strconv.ParseInt(dst[0], 10, 0)
		threadID, _ := strconv.Atoi(dst[1])
		result = append(result, Recipient{User: tele.User{ID: chatID}, ThreadID: threadID})
	}
	slog.Debug("recipients prepared", "recipients", result)
	return result
}

// Send notification to all prepared recipients
func (b *Bot) Send(notification []string) (err error) {
	/* conf := config.GetConfig()
	if conf.DryRun {
		slog.Info("DRY RUN MODE: skipping Telegram message sending")
		for _, dest := range b.destination {
			for _, txt := range notification {
				slog.Debug("DRY RUN: would send notification", "chat_id", dest.User.ID, "thread_id", dest.ThreadID, "preview", txt[:min(100, len(txt))])
			}
		}
		return nil
	} */

	for _, dest := range b.destination {
		for _, txt := range notification {
			if _, err = b.bot.Send(&dest.User, txt, &tele.SendOptions{
				ParseMode: tele.ModeHTML, ThreadID: dest.ThreadID, DisableWebPagePreview: true}); err != nil {
				slog.Error("failed to send notification", "chat_id", dest.User.ID, "thread_id", dest.ThreadID, "notification", txt, "error", err)
				return err
			}
		}
		slog.Debug("notification sent", "chat_id", dest.User.ID, "thread_id", dest.ThreadID, "parts_count", len(notification))
	}
	return nil
}

/* func min(a, b int) int {
	if a < b {
		return a
	}
	return b
} */
