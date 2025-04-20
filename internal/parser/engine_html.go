package parser

import (
	"github.com/kettari/location-bot/internal/entity"
	"github.com/kettari/location-bot/internal/scraper"
	"golang.org/x/net/html"
	"io"
	"log/slog"
	"strings"
)

type TokenizerEngine struct{}

func NewTokenizerEngine() *TokenizerEngine {
	return &TokenizerEngine{}
}

// Process event with HTML tokenizer, extract relevant information and return [entity.Game] struct
func (re *TokenizerEngine) Process(page *scraper.Page) (*[]entity.Game, error) {
	var games []entity.Game
	tok := html.NewTokenizer(strings.NewReader(page.Html))

	for {
		tokenType := tok.Next()
		token := tok.Token()
		if tokenType == html.ErrorToken {
			if tok.Err() == io.EOF {
				break
			}
			slog.Error("error processing token", "token", token.String(), "token_type", tokenType)
			break
		}
		switch tokenType {
		case html.StartTagToken, html.SelfClosingTagToken:
			switch token.Data {
			case "div":
				// Create new game object
				processDIV(&token, &games, page.URL)
				break
			case "h4":
				processH4(tok, &token, &games)
				break
			}
		}

		//slog.Debug("processing token", "token_data", token.Data, "token_atom", token.DataAtom, "token_attributes", token.Attr, "token_type", tokenType)
	}

	slog.Debug("found games", "games", games)

	return &games, nil
}

func processDIV(token *html.Token, games *[]entity.Game, url string) {
	// Check for outermost block
	if !attrValueContains(token.Attr, "event-single-") {
		if attrValueContains(token.Attr, "game-single") {
			// Single event page
			game := &entity.Game{
				ExternalID:    "game" + url[strings.LastIndex(url, "/")+1:],
				CalendarEvent: entity.CalendarEventWeekday,
			}
			*games = append(*games, *game)
		} else if attrValueContains(token.Attr, "event-single") {
			// Weekend events page
			game := &entity.Game{
				ExternalID:    attrValue(token.Attr, "id"),
				CalendarEvent: entity.CalendarEventWeekend,
			}
			*games = append(*games, *game)
		}
	}
}

func processH4(tok *html.Tokenizer, curToken *html.Token, games *[]entity.Game) error {
	if len(*games) == 0 {
		return nil
	}

	nextTokenType := tok.Next()
	nextToken := tok.Token()
	if nextTokenType == html.ErrorToken {
		if tok.Err() == io.EOF {
			return nil
		}
		return tok.Err()
	}
	if nextTokenType == html.TextToken {
		game := &(*games)[len(*games)-1]
		switch game.CalendarEvent {
		case entity.CalendarEventWeekday:
			game.Title = nextToken.Data
			break
		case entity.CalendarEventWeekend:
			break
		}
	}

	return nil
}

func attrValueContains(attrs []html.Attribute, value string) bool {
	for _, a := range attrs {
		if strings.Contains(a.Val, value) {
			return true
		}
	}
	return false
}

func attrValue(attrs []html.Attribute, key string) string {
	for _, a := range attrs {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
