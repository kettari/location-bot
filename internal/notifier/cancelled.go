package notifier

import (
	"github.com/kettari/location-bot/internal/bot"
	"github.com/kettari/location-bot/internal/entity"
	"log/slog"
)

type CancelledGame struct {
	bot bot.MessageDispatcher
}

var cancelledGame *CancelledGame

func CancelledGameObserver(bot bot.MessageDispatcher) *CancelledGame {
	if cancelledGame == nil {
		cancelledGame = &CancelledGame{
			bot: bot,
		}
	}
	return cancelledGame
}

func (g *CancelledGame) Update(game *entity.Game, subject entity.SubjectType) {
	if subject == entity.SubjectTypeCancelled {
		slog.Info("cancelled game event fired", "game_id", game.ExternalID)
		notification := game.FormatCancelled()
		if err := g.bot.Send([]string{notification}); err == nil {
			slog.Error("cancelled game event error", "error", err)
		}
	}
}
