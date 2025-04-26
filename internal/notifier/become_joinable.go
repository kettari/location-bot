package notifier

import (
	"github.com/kettari/location-bot/internal/bot"
	"github.com/kettari/location-bot/internal/entity"
	"log/slog"
)

type BecomeJoinableGame struct {
	bot bot.MessageDispatcher
}

var becomeJoinableGame *BecomeJoinableGame

func BecomeJoinableGameObserver(bot bot.MessageDispatcher) *BecomeJoinableGame {
	if becomeJoinableGame == nil {
		becomeJoinableGame = &BecomeJoinableGame{
			bot: bot,
		}
	}
	return becomeJoinableGame
}

func (g *BecomeJoinableGame) Update(game *entity.Game, subject entity.SubjectType) {
	if subject == entity.SubjectTypeBecomeJoinable {
		slog.Info("game become joinable event fired", "game_id", game.ExternalID)
		notification := game.FormatFreeSeatsAdded()
		if err := g.bot.Send([]string{notification}); err == nil {
			slog.Error("joinable game event error", "error", err)
		}
	}
}
