package parser

import (
	"github.com/kettari/location-bot/internal/entity"
	"github.com/kettari/location-bot/internal/scraper"
)

type Parser struct {
	engine Engine
}

func NewParser(engine Engine) *Parser {
	return &Parser{
		engine: engine,
	}
}

func (p *Parser) Parse(pages *[]scraper.Page, collection entity.Collection) error {
	for _, page := range *pages {
		games, err := p.engine.Process(&page)
		if err != nil {
			return err
		}
		collection.Add(*games...)
	}
	return nil
}
