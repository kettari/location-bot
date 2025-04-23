package notifier

import (
	"github.com/kettari/location-bot/internal/entity"
	"gopkg.in/telebot.v4"
	"log/slog"
)

type BecomeJoinableGame struct {
	bot *telebot.Bot
}

func NewBecomeJoinableGame(bot *telebot.Bot) *BecomeJoinableGame {
	return &BecomeJoinableGame{
		bot: bot,
	}
}

func (g *BecomeJoinableGame) Update(game *entity.Game, subject entity.SubjectType) {
	if subject == entity.SubjectTypeBecomeJoinable {
		slog.Warn("game become joinable event fired", "game_id", game.ID)

		recipient := telebot.User{ID: 9505498}
		notification := game.FormatNew()

		if _, err := g.bot.Send(&recipient, notification, &telebot.SendOptions{
			ParseMode: telebot.ModeHTML, ThreadID: 0, DisableWebPagePreview: true}); err != nil {
			slog.Error("failed to send notification", "error", err)
		}
		slog.Debug("notification sent")
	}
}
