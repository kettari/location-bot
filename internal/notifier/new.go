package notifier

import (
	"github.com/kettari/location-bot/internal/bot"
	"github.com/kettari/location-bot/internal/entity"
	"log/slog"
)

type NewGame struct {
	bot bot.MessageDispatcher
}

var newGame *NewGame

func NewGameObserver(bot bot.MessageDispatcher) *NewGame {
	if newGame == nil {
		newGame = &NewGame{
			bot: bot,
		}
	}
	return newGame
}

func (g *NewGame) Update(game *entity.Game, subject entity.SubjectType) {
	if subject == entity.SubjectTypeNew {
		slog.Info("new game event fired", "game_id", game.ExternalID)
		notification := game.FormatNew()
		if err := g.bot.Send([]string{notification}); err == nil {
			slog.Error("new game event error", "error", err)
		}
	}
}
