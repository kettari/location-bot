package entity

import (
	"log/slog"
)

type BecomeJoinableGame struct {
	bot MessageDispatcher
}

var becomeJoinableGame *BecomeJoinableGame

func BecomeJoinableGameObserver(bot MessageDispatcher) *BecomeJoinableGame {
	if becomeJoinableGame == nil {
		becomeJoinableGame = &BecomeJoinableGame{
			bot: bot,
		}
	}
	return becomeJoinableGame
}

func (g *BecomeJoinableGame) Update(game *Game, subject SubjectType) {
	if subject == SubjectTypeBecomeJoinable {
		slog.Info("game become joinable event fired", "game_id", game.ExternalID)
		notification := game.FormatFreeSeatsAdded()
		if err := g.bot.Send([]string{notification}); err != nil {
			slog.Error("joinable game event error", "error", err)
		}
	}
}
