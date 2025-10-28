package parser

import (
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kettari/location-bot/internal/entity"
	"github.com/kettari/location-bot/internal/scraper"
	html "golang.org/x/net/html"
)

type HtmlEngineV2 struct{}

var monthsMapV2 = map[string]int{
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
	"январь":   1, // именительный падеж
	"февраль":  2,
	"март":     3,
	"апрель":   4,
	"май":      5,
	"июнь":     6,
	"июль":     7,
	"август":   8,
	"сентябрь": 9,
	"октябрь":  10,
	"ноябрь":   11,
	"декабрь":  12,
}

func NewHtmlEngineV2() *HtmlEngineV2 {
	return &HtmlEngineV2{}
}

// Process parses HTML page and extracts game information into entity.Game structs
func (he *HtmlEngineV2) Process(page *scraper.Page) (*[]entity.Game, error) {
	return he.ProcessWithEvents(page, nil)
}

// ProcessWithEvents parses HTML page with optional event metadata for date fallback
func (he *HtmlEngineV2) ProcessWithEvents(page *scraper.Page, eventMap map[string]scraper.RoleconEvent) (*[]entity.Game, error) {
	doc, err := html.Parse(strings.NewReader(page.Html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	games, slots := he.extractGamesFromPage(doc, page)
	he.assignDatesFromSlots(games, slots)

	// Store the HTML for extracting time from pages without dates
	// If page parsing didn't find dates, try to use event metadata
	if eventMap != nil {
		if event, ok := eventMap[page.URL]; ok {
			he.fallbackToEventDates(games, event, page.Html)
		}
	}

	he.setJoinableFlags(games)

	// Log warning for games without dates
	for _, game := range games {
		if game.Date.IsZero() {
			slog.Warn("game has no date",
				"game_id", game.ExternalID,
				"title", game.Title,
				"url", page.URL)
		}
	}

	slog.Debug("page processed", "page_url", page.URL, "games_count", len(games))
	return &games, nil
}

func (he *HtmlEngineV2) extractGamesFromPage(doc *html.Node, page *scraper.Page) ([]entity.Game, map[int]time.Time) {
	var games []entity.Game
	slots := make(map[int]time.Time)

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			if he.isDivEventDay(n) {
				he.parseWeekendDateNodeV2(n.FirstChild, slots)
			}
			if he.isDivEvent(n) {
				game := he.createGameFromDiv(n, page)
				he.processEventNodeV2(n, &game, page.URL)
				games = append(games, game)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)

	return games, slots
}

func (he *HtmlEngineV2) createGameFromDiv(n *html.Node, page *scraper.Page) entity.Game {
	id := he.attrValue(n.Attr, "id")
	if id == "" {
		id = "game" + page.URL[strings.LastIndex(page.URL, "/")+1:]
	}

	slot, _ := strconv.Atoi(he.attrValue(n.Attr, "data-timeslot"))

	return entity.Game{
		ExternalID: id,
		URL:        page.URL,
		Slot:       slot,
	}
}

func (he *HtmlEngineV2) assignDatesFromSlots(games []entity.Game, slots map[int]time.Time) {
	for k := range games {
		if games[k].Slot == 0 {
			continue
		}
		if gameDate, ok := slots[games[k].Slot]; ok {
			games[k].Date = gameDate
		}
	}
}

func (he *HtmlEngineV2) setJoinableFlags(games []entity.Game) {
	for k := range games {
		games[k].Joinable = games[k].Date.After(time.Now()) &&
			games[k].SeatsTotal > 0 && games[k].SeatsFree > 0
	}
}

// fallbackToEventDates sets dates from event metadata if game date is not set.
// The date is taken from event metadata, but the time is extracted from the game's HTML page.
func (he *HtmlEngineV2) fallbackToEventDates(games []entity.Game, event scraper.RoleconEvent, htmlContent string) {
	if event.Start == "" {
		return
	}

	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		slog.Warn("failed to load Moscow timezone", "err", err)
		return
	}

	// Parse ISO date from event metadata (this gives us the date)
	var eventDate time.Time

	// Try ISO format with timezone first
	if eventDate, err = time.Parse("2006-01-02T15:04:05-07:00", event.Start); err == nil {
		// Already has timezone, just convert to Moscow
		eventDate = eventDate.In(moscow)
	} else if eventDate, err = time.ParseInLocation("2006-01-02T15:04:05", event.Start, moscow); err == nil {
		// ISO format without timezone - assume Moscow time
	} else if eventDate, err = time.ParseInLocation("2006-01-02 15:04:05", event.Start, moscow); err == nil {
		// Format "2025-10-30 19:00:00" - assume Moscow time
	} else if eventDate, err = time.ParseInLocation("2006-01-02", event.Start, moscow); err != nil {
		slog.Debug("failed to parse event date", "start", event.Start, "err", err)
		return
	}

	// Try to extract the start time from the HTML page
	var timeFromHTML map[int][2]int // maps slot number to [hour, minute]

	if len(games) == 1 {
		// Single game pages: extract time from the page directly
		hour, minute := he.extractTimeFromHTML(htmlContent)
		if hour > 0 || minute > 0 {
			timeFromHTML = map[int][2]int{
				games[0].Slot: [2]int{hour, minute},
			}
		}
	} else {
		// Summary pages: extract times from tab-caption elements
		timeFromHTML = he.extractTimesFromSummaryPage(htmlContent)
	}

	slog.Debug("extracted time from HTML",
		"time_map", timeFromHTML,
		"games_count", len(games),
		"event_start", event.Start,
		"event_date", eventDate,
		"event_date_time", eventDate.Format("15:04"))

	// Set dates for games that don't have them
	for k := range games {
		if games[k].Date.IsZero() {
			var finalDate time.Time
			var timeHour, timeMinute int

			// Check if we have a time for this game's slot
			if slotTime, ok := timeFromHTML[games[k].Slot]; ok {
				timeHour, timeMinute = slotTime[0], slotTime[1]
			}

			// If we successfully extracted time from HTML, use it with date from event metadata
			// Otherwise use both date and time from event metadata
			if timeHour > 0 || timeMinute > 0 {
				// Only use extracted time if it's actually set (not 00:00)
				finalDate = time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(),
					timeHour, timeMinute, 0, 0, moscow)
				slog.Debug("using time from HTML with date from event metadata",
					"game_id", games[k].ExternalID,
					"slot", games[k].Slot,
					"time_from_html", fmt.Sprintf("%02d:%02d", timeHour, timeMinute),
					"event_date", eventDate,
					"event_date_time", eventDate.Format("15:04"),
					"final_date", finalDate,
					"final_date_time", finalDate.Format("15:04"))
			} else {
				// Use time from event metadata
				finalDate = eventDate
				slog.Debug("using both date and time from event metadata",
					"game_id", games[k].ExternalID,
					"slot", games[k].Slot,
					"event_date_time", eventDate.Format("15:04"),
					"final_date", finalDate,
					"final_date_time", finalDate.Format("15:04"))
			}

			games[k].Date = finalDate
		}
	}
}

// extractTimeFromHTML extracts the start time from HTML content
// Looks for patterns like "Пятница (19:00 - 23:00)" and extracts "19:00"
func (he *HtmlEngineV2) extractTimeFromHTML(htmlContent string) (hour int, minute int) {
	// Pattern to match time ranges like "(19:00 - 23:00)"
	re := regexp.MustCompile(`\((\d{2}):(\d{2})\s*-\s*\d{2}:\d{2}\)`)
	matches := re.FindStringSubmatch(htmlContent)

	if len(matches) >= 3 {
		if h, err := strconv.Atoi(matches[1]); err == nil {
			hour = h
		}
		if m, err := strconv.Atoi(matches[2]); err == nil {
			minute = m
		}
		slog.Debug("extractTimeFromHTML found match",
			"pattern", re.String(),
			"matches", matches,
			"extracted", fmt.Sprintf("%02d:%02d", hour, minute))
	}

	return hour, minute
}

// extractTimesFromSummaryPage extracts times from summary page tab-caption elements
// Returns a map from timeslot number to [hour, minute]
// Example: finds "Пятница (19:00 - 23:00)" in a tab-caption with data-timeslot="3361"
func (he *HtmlEngineV2) extractTimesFromSummaryPage(htmlContent string) map[int][2]int {
	result := make(map[int][2]int)

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		slog.Debug("failed to parse HTML in extractTimesFromSummaryPage", "err", err)
		return result
	}

	// Find all elements with class="tab-caption"
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			class := he.attrValue(n.Attr, "class")
			if strings.Contains(class, "tab-caption") {
				// Extract data-timeslot
				timeslotStr := he.attrValue(n.Attr, "data-timeslot")
				timeslot, err := strconv.Atoi(timeslotStr)
				if err != nil {
					slog.Debug("failed to parse timeslot", "timeslot", timeslotStr, "err", err)
				} else {
					// Extract text content
					text := he.extractTextContentV2(n)

					// Extract time from text (e.g., "Пятница (19:00 - 23:00)")
					re := regexp.MustCompile(`\((\d{2}):(\d{2})\s*-\s*\d{2}:\d{2}\)`)
					matches := re.FindStringSubmatch(text)

					if len(matches) >= 3 {
						if h, err := strconv.Atoi(matches[1]); err == nil {
							if m, err := strconv.Atoi(matches[2]); err == nil {
								result[timeslot] = [2]int{h, m}
								slog.Debug("extracted time from tab-caption",
									"timeslot", timeslot,
									"text", text,
									"hour", h,
									"minute", m)
							}
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	return result
}

func (he *HtmlEngineV2) isDivEvent(n *html.Node) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			classVal := a.Val
			// Match event-single or game-single classes
			// Check for exact match first
			if classVal == "game-single" {
				return true
			}
			// Then check for event-single (but not event-single-content, event-single-about, etc.)
			if strings.Contains(classVal, "event-single") {
				// Make sure it's not a modifier class like "event-single-content"
				if !strings.Contains(classVal, "event-single-") {
					return true
				}
			}
		}
	}
	return false
}

func (he *HtmlEngineV2) isDivEventDay(n *html.Node) bool {
	for _, a := range n.Attr {
		if a.Key == "class" && strings.Contains(a.Val, "event-day") {
			return true
		}
	}
	return false
}

func (he *HtmlEngineV2) processEventNodeV2(n *html.Node, game *entity.Game, baseURL string) {
	switch n.Data {
	case "h4":
		// Handle game title - check if it has class "game-title"
		class := he.attrValue(n.Attr, "class")
		if class == "game-title" {
			// Weekend event with link
			he.extractTitleV2(n.FirstChild, game, baseURL)
		} else {
			// Single game page with direct text
			he.extractTitleV2(n, game, baseURL)
		}
	case "p":
		// Handle date/time
		if he.attrValue(n.Attr, "class") == "subcaption-h4" {
			game.Date = he.extractSingleEventDateV2(n.FirstChild)
		}
		// Handle notes section
		if strings.Contains(he.attrValue(n.Attr, "class"), "game-description") &&
			he.hasParentWithClass(n, "i-notes") {
			game.Notes = he.extractTextContentV2(n)
		}
	case "div":
		// Handle description section
		if he.attrValue(n.Attr, "class") == "caption" {
			// Check if this is the "Описание" caption
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode &&
				strings.TrimSpace(n.FirstChild.Data) == "Описание" {
				// Look for the next sibling with game-description class
				if n.NextSibling != nil {
					game.Description = he.extractDescriptionFromCaptionSiblingV2(n.NextSibling)
				}
			}
		}
	case "table":
		if strings.Contains(he.attrValue(n.Attr, "class"), "table-single") {
			he.populateTableV2(n.FirstChild, game, baseURL)
		}
	}

	// Traverse child nodes
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		he.processEventNodeV2(c, game, baseURL)
	}
}

func (he *HtmlEngineV2) extractTitleV2(n *html.Node, game *entity.Game, baseURL string) {
	// Handle h4 with game-title class (weekend events with links)
	if n.Type == html.ElementNode && n.Data == "h4" && he.attrValue(n.Attr, "class") == "game-title" {
		if n.FirstChild != nil {
			he.extractTitleV2(n.FirstChild, game, baseURL)
		}
		return
	}

	// Handle h4 with direct text (single game pages)
	// Skip if this h4 is inside event-xs or info divs (those are nested event info, not the game title)
	if n.Type == html.ElementNode && n.Data == "h4" {
		// Check parent to avoid nested event info
		parent := n.Parent
		for parent != nil {
			if parent.Type == html.ElementNode {
				parentClass := he.attrValue(parent.Attr, "class")
				if parentClass == "event-xs" || parentClass == "info" {
					// This h4 is inside event-xs or info, skip it
					return
				}
			}
			parent = parent.Parent
		}

		if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
			// Text is directly in h4, not in an <a> tag
			game.Title = strings.TrimSpace(n.FirstChild.Data)
		}
		// Also check for links inside h4
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			he.extractTitleV2(c, game, baseURL)
		}
		return
	}

	// Handle link with title
	if n.Type == html.ElementNode && n.Data == "a" {
		if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
			game.Title = strings.TrimSpace(n.FirstChild.Data)
			href := he.attrValue(n.Attr, "href")
			if len(href) > 0 {
				if strings.HasPrefix(href, "http") {
					game.URL = href
				} else {
					game.URL = "https://rolecon.ru" + href
				}
			}
		}
		return
	}

	// Continue traversing
	if n.NextSibling != nil {
		he.extractTitleV2(n.NextSibling, game, baseURL)
	}
	if n.FirstChild != nil {
		he.extractTitleV2(n.FirstChild, game, baseURL)
	}
}

func (he *HtmlEngineV2) extractSingleEventDateV2(n *html.Node) time.Time {
	if n == nil {
		return time.Time{}
	}

	if n.Type == html.TextNode && len(strings.Trim(n.Data, " \n\t\r")) > 0 {
		eventDate := strings.Trim(n.Data, " \n\t\r")

		// Remove possible leading/trailing non-word characters
		eventDate = strings.Trim(eventDate, " \n\t\r")

		// Try multiple regex patterns
		patterns := []string{
			`(\d{1,2})\s+([\p{Cyrillic}]+)\s+(\d{4}),\s*(\d{2}):(\d{2})`,                       // "30 октября 2025, 19:00"
			`(\d{1,2})\s+([\p{Cyrillic}]+)\s+(\d{4}),\s*(\d{2}):(\d{2})\s*-\s*(\d{2}):(\d{2})`, // with end time
			`[\p{Cyrillic}]+\s+\((\d{2}):(\d{2})\s*-\s*\d{2}:\d{2}\)`,                          // "Пятница (19:00 - 23:00)" - no date, only time
		}

		for i, pattern := range patterns {
			r := regexp.MustCompile(pattern)
			matches := r.FindStringSubmatch(eventDate)

			// Pattern "Пятница (19:00 - 23:00)" returns only time, no date
			if i == 2 && len(matches) >= 3 {
				// This pattern doesn't have date, return zero time to trigger fallback
				slog.Debug("found time-only pattern without date", "pattern", pattern, "matches", matches)
				return time.Time{}
			}

			if len(matches) >= 6 {
				moscow, err := time.LoadLocation("Europe/Moscow")
				if err != nil {
					slog.Warn("failed to load Moscow timezone", "err", err)
					continue
				}

				year, _ := strconv.Atoi(matches[3])
				day, _ := strconv.Atoi(matches[1])
				hour, _ := strconv.Atoi(matches[4])
				minute, _ := strconv.Atoi(matches[5])

				if month, ok := monthsMapV2[matches[2]]; ok {
					slog.Debug("parsed single event date", "date", eventDate, "parsed", time.Date(year, time.Month(month), day, hour, minute, 0, 0, moscow))
					return time.Date(year, time.Month(month), day, hour, minute, 0, 0, moscow)
				} else {
					slog.Warn("month not found in map", "month", matches[2], "full_date", eventDate)
				}
			}
		}

		slog.Debug("failed to parse date", "raw_date", eventDate)
	}

	if n.NextSibling != nil {
		return he.extractSingleEventDateV2(n.NextSibling)
	}
	if n.FirstChild != nil {
		return he.extractSingleEventDateV2(n.FirstChild)
	}

	return time.Time{}
}

func (he *HtmlEngineV2) populateTableV2(n *html.Node, game *entity.Game, baseURL string) {
	if n.Type == html.ElementNode && n.Data == "tbody" {
		if n.FirstChild != nil {
			he.populateTableV2(n.FirstChild, game, baseURL)
		}
	}
	if n.Type == html.ElementNode && n.Data == "tr" {
		he.populateRowV2(n.FirstChild, game, baseURL)
	}
	if n.NextSibling != nil {
		he.populateTableV2(n.NextSibling, game, baseURL)
	}
}

func (he *HtmlEngineV2) populateRowV2(n *html.Node, game *entity.Game, baseURL string) {
	if n.Type == html.ElementNode && n.Data == "td" {
		if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
			key := n.FirstChild.Data
			switch key {
			case "Сеттинг:":
				he.extractTableCellValueV2(n, &game.Setting)
			case "Система:":
				he.extractTableCellValueV2(n, &game.System)
			case "Жанр:":
				he.extractTableCellValueV2(n, &game.Genre)
			case "Игру проводит:":
				he.populateAuthorV2(n, game, baseURL)
			case "Места:":
				he.extractSeatsV2(n, game)
			}
		}
	}
	if n.NextSibling != nil {
		he.populateRowV2(n.NextSibling, game, baseURL)
	}
}

func (he *HtmlEngineV2) extractTableCellValueV2(n *html.Node, value *string) {
	// Look for the value in next sibling's next sibling (skip the blank td)
	if n.NextSibling != nil && n.NextSibling.NextSibling != nil {
		valueNode := n.NextSibling.NextSibling
		if valueNode.FirstChild != nil && valueNode.FirstChild.Type == html.TextNode {
			*value = strings.TrimSpace(valueNode.FirstChild.Data)
		}
	}
}

func (he *HtmlEngineV2) extractSeatsV2(n *html.Node, game *entity.Game) {
	if n.NextSibling != nil && n.NextSibling.NextSibling != nil {
		valueNode := n.NextSibling.NextSibling
		if valueNode.FirstChild != nil && valueNode.FirstChild.Type == html.TextNode {
			seatsText := valueNode.FirstChild.Data
			// Parse "Осталось X мест из Y" or just "X мест из Y"
			r := regexp.MustCompile(`(\d+)\s+мест\s+из\s+(\d+)`)
			matches := r.FindAllStringSubmatch(seatsText, -1)
			if len(matches) > 0 {
				game.SeatsFree, _ = strconv.Atoi(matches[0][1])
				game.SeatsTotal, _ = strconv.Atoi(matches[0][2])
			}
		}
	}
}

func (he *HtmlEngineV2) populateAuthorV2(n *html.Node, game *entity.Game, baseURL string) {
	// Find the link in the next sibling's next sibling
	if n.NextSibling != nil && n.NextSibling.NextSibling != nil {
		linkNode := n.NextSibling.NextSibling.FirstChild
		if linkNode != nil && linkNode.Type == html.ElementNode && linkNode.Data == "a" {
			if linkNode.FirstChild != nil && linkNode.FirstChild.Type == html.TextNode {
				game.MasterName = strings.TrimSpace(linkNode.FirstChild.Data)
			}
			href := he.attrValue(linkNode.Attr, "href")
			if len(href) > 0 {
				if strings.HasPrefix(href, "http") {
					game.MasterLink = href
				} else {
					game.MasterLink = "https://rolecon.ru" + href
				}
			}
		}
	}
}

func (he *HtmlEngineV2) extractTextContentV2(n *html.Node) string {
	var result strings.Builder
	he.writeNodeText(n, &result)
	return strings.TrimSpace(result.String())
}

func (he *HtmlEngineV2) writeNodeText(n *html.Node, result *strings.Builder) {
	if n.Type == html.TextNode {
		result.WriteString(n.Data)
		return
	}

	if n.Type == html.ElementNode {
		switch n.Data {
		case "p":
			if result.Len() > 0 {
				result.WriteString("\n\n")
			}
		case "br":
			result.WriteString("\n")
		case "ul", "ol":
			for li := n.FirstChild; li != nil; li = li.NextSibling {
				if li.Data == "li" {
					result.WriteString("• ")
					for c := li.FirstChild; c != nil; c = c.NextSibling {
						he.writeNodeText(c, result)
					}
					result.WriteString("\n")
				}
			}
			return
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		he.writeNodeText(c, result)
	}
}

func (he *HtmlEngineV2) extractDescriptionFromCaptionSiblingV2(n *html.Node) string {
	if n.Type == html.ElementNode && he.attrValue(n.Attr, "class") == "game-description" {
		return he.extractTextContentV2(n)
	}
	if n.FirstChild != nil {
		return he.extractDescriptionFromCaptionSiblingV2(n.FirstChild)
	}
	return ""
}

func (he *HtmlEngineV2) hasParentWithClass(n *html.Node, className string) bool {
	if n.Parent == nil {
		return false
	}
	if n.Parent.Type == html.ElementNode && n.Parent.Data == "div" {
		classAttr := he.attrValue(n.Parent.Attr, "class")
		if strings.Contains(classAttr, className) {
			return true
		}
	}
	return he.hasParentWithClass(n.Parent, className)
}

func (he *HtmlEngineV2) parseWeekendDateNodeV2(n *html.Node, slots map[int]time.Time) {
	if n == nil {
		return
	}

	switch {
	case he.isCaptionNode(n):
		he.parseCaptionDate(n, slots)
	case he.isTabsCaptionNode(n):
		he.parseWeekendDateNodeV2(n.FirstChild, slots)
		return
	case he.isTabCaptionNode(n):
		he.parseTabCaptionSlot(n, slots)
	}

	he.parseWeekendDateNodeV2(n.NextSibling, slots)
}

func (he *HtmlEngineV2) isCaptionNode(n *html.Node) bool {
	return n.Type == html.ElementNode && he.attrValue(n.Attr, "class") == "caption" &&
		n.FirstChild != nil && n.FirstChild.Type == html.TextNode
}

func (he *HtmlEngineV2) isTabsCaptionNode(n *html.Node) bool {
	return n.Type == html.ElementNode && he.attrValue(n.Attr, "class") == "tabs-caption"
}

func (he *HtmlEngineV2) isTabCaptionNode(n *html.Node) bool {
	return n.Type == html.ElementNode && strings.Contains(he.attrValue(n.Attr, "class"), "tab-caption")
}

func (he *HtmlEngineV2) parseCaptionDate(n *html.Node, slots map[int]time.Time) {
	dateText := ""
	if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
		dateText = n.FirstChild.Data
	}

	// Pattern matches formats like:
	// - "Пятница — 7.11.2025" (without parentheses)
	// - "Воскресенье (09.11) — 9.11.2025" (with parentheses)
	// The actual date is always after the em dash (—)
	r := regexp.MustCompile(`[\p{Cyrillic}]+(?:\s+\([^\)]+\))?\s—\s(\d{1,2})\.(\d{2})\.(\d{4})`)
	matches := r.FindAllStringSubmatch(dateText, -1)
	if len(matches) == 0 {
		slog.Debug("parseCaptionDate: no matches found", "date_text", dateText, "pattern", r.String())
		return
	}

	slog.Debug("parseCaptionDate: matches found", "date_text", dateText, "matches", matches)

	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		slog.Warn("failed to load Moscow timezone", "err", err)
		return
	}

	year, _ := strconv.Atoi(matches[0][3])
	month, _ := strconv.Atoi(matches[0][2])
	day, _ := strconv.Atoi(matches[0][1])
	slots[0] = time.Date(year, time.Month(month), day, 0, 0, 0, 0, moscow)
}

func (he *HtmlEngineV2) parseTabCaptionSlot(n *html.Node, slots map[int]time.Time) {
	slotNumber, _ := strconv.Atoi(he.attrValue(n.Attr, "data-timeslot"))
	slotContent := ""
	if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
		slotContent = n.FirstChild.Data
	}

	r := regexp.MustCompile(`[\p{Cyrillic}]+\s*\((\d{2}):(\d{2})`)
	matches := r.FindAllStringSubmatch(slotContent, -1)
	if len(matches) == 0 {
		return
	}

	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		slog.Warn("failed to load Moscow timezone", "err", err)
		return
	}

	rootDate, ok := slots[0]
	if !ok {
		slog.Warn("root date not set for slot", "slot", slotNumber)
		return
	}

	hour, _ := strconv.Atoi(matches[0][1])
	minute, _ := strconv.Atoi(matches[0][2])
	slots[slotNumber] = time.Date(rootDate.Year(), rootDate.Month(), rootDate.Day(), hour, minute, 0, 0, moscow)
}

func (he *HtmlEngineV2) attrValue(attrs []html.Attribute, key string) string {
	for _, a := range attrs {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
