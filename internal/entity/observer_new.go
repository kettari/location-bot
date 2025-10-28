package entity

import (
	"log/slog"
)

type NewGame struct {
	bot MessageDispatcher
}

var newGame *NewGame

func NewGameObserver(bot MessageDispatcher) *NewGame {
	if newGame == nil {
		newGame = &NewGame{
			bot: bot,
		}
	}
	return newGame
}

func (g *NewGame) Update(game *Game, subject SubjectType) {
	if subject == SubjectTypeNew {
		slog.Info("new game event fired", "game_id", game.ExternalID)
		notification := game.FormatNew()
		if err := g.bot.Send([]string{notification}); err != nil {
			slog.Error("new game event error", "error", err)
		}
	}
}
