package entity

import (
	"log/slog"
)

type CancelledGame struct {
	bot MessageDispatcher
}

var cancelledGame *CancelledGame

func CancelledGameObserver(bot MessageDispatcher) *CancelledGame {
	if cancelledGame == nil {
		cancelledGame = &CancelledGame{
			bot: bot,
		}
	}
	return cancelledGame
}

func (g *CancelledGame) Update(game *Game, subject SubjectType) {
	if subject == SubjectTypeCancelled {
		slog.Info("cancelled game event fired", "game_id", game.ExternalID)
		notification := game.FormatCancelled()
		if err := g.bot.Send([]string{notification}); err == nil {
			slog.Error("cancelled game event error", "error", err)
		}
	}
}
