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

func (p *Parser) ParseWithEvents(result *scraper.FetchResult, collection entity.Collection) error {
	for _, page := range result.Pages {
		games, err := p.engine.ProcessWithEvents(&page, result.EventMap)
		if err != nil {
			return err
		}
		collection.Add(*games...)
	}
	return nil
}

func (p *Parser) ParseSinglePage(page *scraper.Page) (*[]entity.Game, error) {
	return p.engine.Process(page)
}
