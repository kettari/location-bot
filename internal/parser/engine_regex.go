package parser

import (
	"errors"
	"github.com/kettari/location-bot/internal/entity"
	"github.com/kettari/location-bot/internal/scraper"
	"regexp"
)

type RegexEngine struct{}

type eventType string

const (
	eventTypeOpen  eventType = "can-join"
	eventTypeClose eventType = "cannot-join"
)

func NewRegexEngine() *RegexEngine {
	return &RegexEngine{}
}

// Process event with regexp, extract relevant information and return [entity.Game] struct
func (re *RegexEngine) Process(page *scraper.Page) (*[]entity.Game, error) {
	/*gameType, err := re.parseEventType(event)
	if err != nil {
		return nil, err
	}

	game := entity.Game{}
	if gameType == eventTypeOpen {
		game.Joinable = true
	} else {
		game.Joinable = false
	}*/

	return nil, nil
}

func (re *RegexEngine) breakDown(html string) (eventsHtml []string, err error) {
	r := regexp.MustCompile(`<div class="event-single[^-]`)
	match := r.FindAllStringSubmatchIndex(html, -1)
	previousIndex := 0
	for _, v := range match {
		if previousIndex == 0 {
			previousIndex = v[0]
		} else {
			eventsHtml = append(eventsHtml, html[previousIndex:v[0]])
		}
	}

	return []string{}, nil
}

func (re *RegexEngine) parseEventType(event string) (eventType, error) {
	r := regexp.MustCompile(`<div class="event-single\s*[a-z]*\s*(can-join|cannot-join)`)
	match := r.FindString(event)
	if match == string(eventTypeOpen) {
		return eventTypeOpen, nil
	} else if match == string(eventTypeClose) {
		return eventTypeClose, nil
	}
	return "", errors.New("invalid event type")
}
