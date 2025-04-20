package parser

import (
	"github.com/kettari/location-bot/internal/entity"
	"github.com/kettari/location-bot/internal/scraper"
	"golang.org/x/net/html"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type HtmlEngine struct{}

type dateSlots map[int]time.Time

var monthsMap = map[string]int{
	"января":   1,
	"февраля":  2,
	"марта":    3,
	"апреля":   4,
	"мая":      5,
	"июня":     6,
	"июля":     7,
	"августа":  8,
	"сентября": 9,
	"октября":  10,
	"ноября":   11,
	"декабря":  12,
}

func NewHtmlEngine() *HtmlEngine {
	return &HtmlEngine{}
}

// Process event with HTML tokenizer, extract relevant information and return [entity.Game] struct
func (he *HtmlEngine) Process(page *scraper.Page) (*[]entity.Game, error) {
	var games []entity.Game
	slots := make(dateSlots)
	doc, err := html.Parse(strings.NewReader(page.Html))
	if err != nil {
		return nil, err
	}

	var processAllNodes func(*html.Node)
	processAllNodes = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" && he.isDivEventDay(n) {
			if n.FirstChild != nil {
				he.parseWeekendDateNode(n.FirstChild, &slots)
			}
		}
		if n.Type == html.ElementNode && n.Data == "div" && he.isDivEvent(n) {
			// process the Event details within each <div> element
			id := he.attrValue(n.Attr, "id")
			if len(id) == 0 {
				id = "game" + page.URL[strings.LastIndex(page.URL, "/")+1:]
			}
			slotStr := he.attrValue(n.Attr, "data-timeslot")
			slot := 0
			if len(slotStr) > 0 {
				slot, _ = strconv.Atoi(slotStr)
			}
			game := entity.Game{
				ExternalID: id,
				URL:        page.URL,
				Slot:       slot,
			}
			he.processEventNode(n, &game)
			games = append(games, game)
		}
		// traverse the child nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			processAllNodes(c)
		}
	}
	// make a recursive call to your function
	processAllNodes(doc)

	// Assign dates
	for k, game := range games {
		if game.Slot == 0 {
			continue
		}
		gameDate, ok := slots[game.Slot]
		if !ok {
			continue
		}
		games[k].Date = gameDate
	}
	// Set joinable flag
	for k, game := range games {
		if games[k].Date.After(time.Now()) && game.SeatsTotal > 0 && game.SeatsFree > 0 {
			games[k].Joinable = true
		} else {
			games[k].Joinable = false
		}
	}

	slog.Debug("page processed, found games", "page_url", page.URL, "games_count", len(games))

	return &games, nil
}

func (he *HtmlEngine) isDivEvent(n *html.Node) bool {
	for _, a := range n.Attr {
		if a.Key == "class" &&
			((strings.Contains(a.Val, "event-single") && !strings.Contains(a.Val, "event-single-")) ||
				strings.Contains(a.Val, "game-single")) {
			return true
		}
	}
	return false
}

func (he *HtmlEngine) isDivEventDay(n *html.Node) bool {
	for _, a := range n.Attr {
		if a.Key == "class" && strings.Contains(a.Val, "event-day") {
			return true
		}
	}
	return false
}

// process the details of the Event within the <div class="event-single"> element
func (he *HtmlEngine) processEventNode(n *html.Node, game *entity.Game) {
	switch n.Data {
	case "h4":
		if he.attrValue(n.Attr, "class") == "game-title" && n.FirstChild != nil {
			he.findWeekendEventTitle(n.FirstChild, game)
		} else {
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				game.Title = n.FirstChild.Data
			}
		}
		game.Title = strings.Trim(game.Title, " \n\r\t")
		break
	case "p":
		if he.attrValue(n.Attr, "class") == "subcaption-h4" && n.FirstChild != nil {
			game.Date = he.findSingleEventDateTime(n.FirstChild)
		}
		break
	case "table":
		if strings.Contains(he.attrValue(n.Attr, "class"), "table-single") {
			if n.FirstChild != nil {
				he.populateTable(n.FirstChild, game)
			}
		}
		break
	}

	// Traverse child nodes
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		he.processEventNode(c, game)
	}
}

func (he *HtmlEngine) findWeekendEventTitle(n *html.Node, game *entity.Game) {
	if n.Type == html.ElementNode && n.Data == "a" {
		if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
			game.Title = n.FirstChild.Data
			game.URL = "https://rolecon.ru" + he.attrValue(n.Attr, "href")
		}
	}
	if n.NextSibling != nil {
		he.findWeekendEventTitle(n.NextSibling, game)
	}
}

func (he *HtmlEngine) findSingleEventDateTime(n *html.Node) time.Time {
	if n.Type == html.TextNode && len(strings.Trim(n.Data, " \n\t\r")) > 0 {
		eventDate := strings.Trim(n.Data, " \n\t\r")
		// 16 апреля 2025,
		//19:00 - 23:00
		r := regexp.MustCompile(`^(\d{1,2})\s([\x{0400}-\x{04FF}]+)\s(\d{4}),\s(\d{2}):(\d{2})`)
		matches := r.FindAllStringSubmatch(eventDate, -1)
		if len(matches) > 0 {
			moscow, err := time.LoadLocation("Europe/Moscow")
			if err != nil {
				panic(err)
			}
			year, _ := strconv.Atoi(matches[0][3])
			day, _ := strconv.Atoi(matches[0][1])
			hour, _ := strconv.Atoi(matches[0][4])
			minute, _ := strconv.Atoi(matches[0][5])
			return time.Date(year, time.Month(monthsMap[matches[0][2]]), day, hour, minute, 0, 0, moscow)
		}

	}
	if n.NextSibling != nil {
		he.findSingleEventDateTime(n.NextSibling)
	}
	return time.Time{}
}

func (he *HtmlEngine) populateTable(n *html.Node, game *entity.Game) {
	if n.Type == html.ElementNode && n.Data == "tbody" {
		if n.FirstChild != nil {
			he.populateTable(n.FirstChild, game)
		}
	}
	if n.Type == html.ElementNode && n.Data == "tr" {
		if n.FirstChild != nil {
			he.populateRow(n.FirstChild, game)
		}
	}
	if n.NextSibling != nil {
		he.populateTable(n.NextSibling, game)
	}
	return
}

func (he *HtmlEngine) populateRow(n *html.Node, game *entity.Game) {
	if n.Type == html.ElementNode && n.Data == "td" {
		if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
			switch n.FirstChild.Data {
			case "Сеттинг:":
				if n.NextSibling != nil && n.NextSibling.NextSibling != nil &&
					n.NextSibling.NextSibling.FirstChild != nil &&
					n.NextSibling.NextSibling.FirstChild.Type == html.TextNode {
					game.Setting = n.NextSibling.NextSibling.FirstChild.Data
					return
				}
				break
			case "Система:":
				if n.NextSibling != nil && n.NextSibling.NextSibling != nil &&
					n.NextSibling.NextSibling.FirstChild != nil &&
					n.NextSibling.NextSibling.FirstChild.Type == html.TextNode {
					game.System = n.NextSibling.NextSibling.FirstChild.Data
					return
				}
				break
			case "Жанр:":
				if n.NextSibling != nil && n.NextSibling.NextSibling != nil &&
					n.NextSibling.NextSibling.FirstChild != nil &&
					n.NextSibling.NextSibling.FirstChild.Type == html.TextNode {
					game.Genre = n.NextSibling.NextSibling.FirstChild.Data
					return
				}
				break
			case "Игру проводит:":
				if n.NextSibling != nil && n.NextSibling.NextSibling != nil &&
					n.NextSibling.NextSibling.FirstChild != nil {
					he.populateAuthor(n.NextSibling.NextSibling.FirstChild, game)
					return
				}
				break
			case "Места:":
				if n.NextSibling != nil && n.NextSibling.NextSibling != nil &&
					n.NextSibling.NextSibling.FirstChild != nil &&
					n.NextSibling.NextSibling.FirstChild.Type == html.TextNode {
					r := regexp.MustCompile(`Осталось\s+(\d+)\s+мест\s+из\s+(\d+)`)
					matches := r.FindAllStringSubmatch(n.NextSibling.NextSibling.FirstChild.Data, -1)
					for _, match := range matches {
						game.SeatsFree, _ = strconv.Atoi(match[1])
						game.SeatsTotal, _ = strconv.Atoi(match[2])
					}
					return
				}
				break
			}
		}
	}
	if n.NextSibling != nil {
		he.populateRow(n.NextSibling, game)
	}
	return
}

func (he *HtmlEngine) populateAuthor(n *html.Node, game *entity.Game) {
	if n.Type == html.ElementNode && n.Data == "a" {
		if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
			game.MasterName = n.FirstChild.Data
		}
		game.MasterLink = "https://rolecon.ru" + he.attrValue(n.Attr, "href")
	}
	if n.NextSibling != nil {
		he.populateAuthor(n.NextSibling, game)
	}
	return
}

func (he *HtmlEngine) attrValue(attrs []html.Attribute, key string) string {
	for _, a := range attrs {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func (he *HtmlEngine) parseWeekendDateNode(n *html.Node, slots *dateSlots) {
	if n.Type == html.ElementNode && he.attrValue(n.Attr, "class") == "caption" &&
		n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
		// Суббота — 19.04.2025
		r := regexp.MustCompile(`^[\x{0400}-\x{04FF}]+\s.\s(\d{1,2})\.(\d{2})\.(\d{4})`)
		matches := r.FindAllStringSubmatch(n.FirstChild.Data, -1)
		if len(matches) > 0 {
			moscow, err := time.LoadLocation("Europe/Moscow")
			if err != nil {
				panic(err)
			}
			year, _ := strconv.Atoi(matches[0][3])
			month, _ := strconv.Atoi(matches[0][2])
			day, _ := strconv.Atoi(matches[0][1])
			(*slots)[0] = time.Date(year, time.Month(month), day, 0, 0, 0, 0, moscow)
		}
	}
	if n.Type == html.ElementNode && he.attrValue(n.Attr, "class") == "tabs-caption" {
		if n.FirstChild != nil {
			he.parseWeekendDateNode(n.FirstChild, slots)
			return
		}
	}
	if n.Type == html.ElementNode && strings.Contains(he.attrValue(n.Attr, "class"), "tab-caption") {
		slotNumber, _ := strconv.Atoi(he.attrValue(n.Attr, "data-timeslot"))
		slotContent := ""
		if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
			slotContent = n.FirstChild.Data
		}
		r := regexp.MustCompile(`^[\x{0400}-\x{04FF}]+\s\((\d{2}):(\d{2})`)
		matches := r.FindAllStringSubmatch(slotContent, -1)
		if len(matches) > 0 {
			moscow, err := time.LoadLocation("Europe/Moscow")
			if err != nil {
				panic(err)
			}
			rootDate, ok := (*slots)[0]
			if !ok {
				panic("root date not set")
			}
			hour, _ := strconv.Atoi(matches[0][1])
			minute, _ := strconv.Atoi(matches[0][2])
			(*slots)[slotNumber] = time.Date(rootDate.Year(), rootDate.Month(), rootDate.Day(), hour, minute, 0, 0, moscow)
		}
	}
	if n.NextSibling != nil {
		he.parseWeekendDateNode(n.NextSibling, slots)
	}
}
