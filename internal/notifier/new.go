package notifier

import (
	"github.com/kettari/location-bot/internal/entity"
	"gopkg.in/telebot.v4"
	"log/slog"
)

type NewGame struct {
	bot *telebot.Bot
}

func NewNewGame(bot *telebot.Bot) *NewGame {
	return &NewGame{
		bot: bot,
	}
}

func (g *NewGame) Update(game *entity.Game, subject entity.SubjectType) {
	if subject == entity.SubjectTypeNew {
		slog.Info("new game event fired", "game_id", game.ID)

		recipient := telebot.User{ID: 9505498}
		notification := game.FormatNew()

		if _, err := g.bot.Send(&recipient, notification, &telebot.SendOptions{
			ParseMode: telebot.ModeHTML, ThreadID: 0, DisableWebPagePreview: true}); err != nil {
			slog.Error("failed to send notification", "error", err)
		}
		slog.Debug("notification sent")
	}
}
