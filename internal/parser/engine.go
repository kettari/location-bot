package parser

import (
	"github.com/kettari/location-bot/internal/entity"
	"github.com/kettari/location-bot/internal/scraper"
)

type Engine interface {
	Process(*scraper.Page) (*[]entity.Game, error)
}
