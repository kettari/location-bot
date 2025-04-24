package notifier

import (
	"github.com/kettari/location-bot/internal/entity"
	"gopkg.in/telebot.v4"
	"log/slog"
)

type CancelledGame struct {
	bot *telebot.Bot
}

var cancelledGame *CancelledGame

func CancelledGameObserver(bot *telebot.Bot) *CancelledGame {
	if cancelledGame == nil {
		cancelledGame = &CancelledGame{
			bot: bot,
		}
	}
	return cancelledGame
}

func (g *CancelledGame) Update(game *entity.Game, subject entity.SubjectType) {
	if subject == entity.SubjectTypeCancelled {
		slog.Warn("cancelled game event fired", "game_id", game.ExternalID)

		recipient := telebot.User{ID: 9505498}
		notification := game.FormatCancelled()

		if _, err := g.bot.Send(&recipient, notification, &telebot.SendOptions{
			ParseMode: telebot.ModeHTML, ThreadID: 0, DisableWebPagePreview: true}); err != nil {
			slog.Error("failed to send notification", "error", err)
		}
		slog.Debug("notification sent")
	}
}
