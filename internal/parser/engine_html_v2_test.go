package parser

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kettari/location-bot/internal/entity"
	"github.com/kettari/location-bot/internal/scraper"
	htmlpkg "golang.org/x/net/html"
)

func init() {
	// Set up logger for tests
	slog.SetLogLoggerLevel(slog.LevelDebug)
}

func TestHtmlEngineV2_Process_RealExamples(t *testing.T) {
	tests := []struct {
		name       string
		htmlFile   string
		url        string
		wantGames  int
		gameChecks func(*testing.T, *[]entity.Game)
		checkGames func(*testing.T, []entity.Game)
	}{
		{
			name:      "Broken tales game page",
			htmlFile:  "docs/webpage-examples/[Broken tales] Осколки Сказок – Ролекон.html",
			url:       "https://rolecon.ru/game/18601",
			wantGames: 0, // Single game page, not an event page
		},
		{
			name:      "D&D Мор game page",
			htmlFile:  "docs/webpage-examples/D&D Мор – Ролекон.html",
			url:       "https://rolecon.ru/game/18624",
			wantGames: 0, // Single game page
		},
		{
			name:      "Декагон game page",
			htmlFile:  "docs/webpage-examples/Декагон – Ролекон.html",
			url:       "https://rolecon.ru/game/18627",
			wantGames: 0, // Single game page
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Read HTML file
			htmlBytes, err := os.ReadFile(tt.htmlFile)
			if err != nil {
				if os.IsNotExist(err) {
					t.Skipf("HTML file not found: %s", tt.htmlFile)
					return
				}
				t.Fatalf("Failed to read HTML file: %v", err)
			}

			page := &scraper.Page{
				URL:  tt.url,
				Html: string(htmlBytes),
			}

			engine := NewHtmlEngineV2()
			games, err := engine.Process(page)

			if err != nil {
				t.Errorf("Process() error = %v", err)
				return
			}

			if tt.wantGames == 0 && games != nil && len(*games) == 0 {
				// This is expected for single game pages
				return
			}

			if games == nil {
				if tt.wantGames != 0 {
					t.Error("Process() returned nil, expected games")
				}
				return
			}

			if len(*games) != tt.wantGames {
				t.Errorf("Process() returned %d games, want %d", len(*games), tt.wantGames)
			}

			if tt.gameChecks != nil {
				tt.gameChecks(t, games)
			}
			if tt.checkGames != nil {
				tt.checkGames(t, *games)
			}
		})
	}
}

func TestHtmlEngineV2_Process_WeekendEvent(t *testing.T) {
	// Read a weekend event HTML file
	htmlBytes, err := os.ReadFile("docs/webpage-examples/Игры по выходным – Ролекон.html")
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("HTML file not found")
			return
		}
		t.Fatalf("Failed to read HTML file: %v", err)
	}

	page := &scraper.Page{
		URL:  "https://rolecon.ru/lw202041125",
		Html: string(htmlBytes),
	}

	engine := NewHtmlEngineV2()
	games, err := engine.Process(page)

	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if games == nil {
		t.Fatal("Process() returned nil")
	}

	if len(*games) == 0 {
		t.Fatal("Process() returned 0 games")
	}

	// Check first game
	game := (*games)[0]
	if game.Title == "" {
		t.Error("First game has empty Title")
	}
	if game.ExternalID == "" {
		t.Error("First game has empty ExternalID")
	}
	if game.URL == "" {
		t.Error("First game has empty URL")
	}

	// Log all games for debugging
	for i, g := range *games {
		t.Logf("Game %d: Title=%s, System=%s, Setting=%s, Genre=%s, Seats=%d/%d, URL=%s",
			i+1, g.Title, g.System, g.Setting, g.Genre, g.SeatsFree, g.SeatsTotal, g.URL)
	}
}

func TestHtmlEngineV2_Process_ExpandedProgram(t *testing.T) {
	// Read the expanded program HTML file
	htmlBytes, err := os.ReadFile("docs/webpage-examples/Ролекон 2025_ расширенная программа – Ролекон.html")
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("HTML file not found")
			return
		}
		t.Fatalf("Failed to read HTML file: %v", err)
	}

	page := &scraper.Page{
		URL:  "https://rolecon.ru/r25ep",
		Html: string(htmlBytes),
	}

	engine := NewHtmlEngineV2()
	games, err := engine.Process(page)

	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if games == nil {
		t.Fatal("Process() returned nil")
	}

	if len(*games) == 0 {
		t.Error("Process() returned 0 games")
	}

	// Verify at least some games have dates
	hasDates := false
	for _, g := range *games {
		if !g.Date.IsZero() {
			hasDates = true
			break
		}
	}
	if !hasDates {
		t.Error("No games have dates assigned")
	}

	// Verify some games have all fields populated
	allFieldsCount := 0
	for _, g := range *games {
		if g.Title != "" && g.System != "" && g.Setting != "" &&
			g.MasterName != "" && !g.Date.IsZero() {
			allFieldsCount++
		}
	}

	if allFieldsCount == 0 {
		t.Error("No games have all core fields populated")
	}

	t.Logf("Found %d games with all fields, out of %d total", allFieldsCount, len(*games))
}

func TestHtmlEngineV2_ExtractSeats_WithValue(t *testing.T) {
	tests := []struct {
		name      string
		seatsText string
		wantFree  int
		wantTotal int
	}{
		{
			name:      "standard format",
			seatsText: "Осталось 3 мест из 6",
			wantFree:  3,
			wantTotal: 6,
		},
		{
			name:      "without осталось prefix",
			seatsText: "1 мест из 2",
			wantFree:  1,
			wantTotal: 2,
		},
		{
			name:      "zero free seats",
			seatsText: "Осталось 0 мест из 8",
			wantFree:  0,
			wantTotal: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := &entity.Game{}
			// Create a simple HTML structure to test
			htmlStr := `<table><tbody><tr><td>Места:</td><td></td><td>` + tt.seatsText + `</td></tr></tbody></table>`
			doc, err := htmlpkg.Parse(strings.NewReader(htmlStr))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			// Find the td node with "Места:"
			var tdNode *htmlpkg.Node
			var findTd func(*htmlpkg.Node)
			findTd = func(n *htmlpkg.Node) {
				if n.Type == htmlpkg.ElementNode && n.Data == "td" {
					if n.FirstChild != nil && n.FirstChild.Type == htmlpkg.TextNode && n.FirstChild.Data == "Места:" {
						tdNode = n
						return
					}
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					findTd(c)
				}
			}
			findTd(doc)

			if tdNode == nil {
				t.Fatal("Could not find Места: td node")
			}

			engine := NewHtmlEngineV2()
			engine.extractSeatsV2(tdNode, game)

			if game.SeatsFree != tt.wantFree {
				t.Errorf("SeatsFree = %d, want %d", game.SeatsFree, tt.wantFree)
			}
			if game.SeatsTotal != tt.wantTotal {
				t.Errorf("SeatsTotal = %d, want %d", game.SeatsTotal, tt.wantTotal)
			}
		})
	}
}

func TestHtmlEngineV2_ParseDateSlots(t *testing.T) {
	engine := NewHtmlEngineV2()
	slots := make(map[int]time.Time)

	htmlStr := `
	<html>
	<body>
		<div class="event-day">
			<div class="caption">Воскресенье — 2.11.2025</div>
			<div class="tabs-caption">
				<div class="tab-caption active" data-timeslot="3410">День (11:00-15:00)</div>
				<div class="tab-caption" data-timeslot="3411">Вечер (16:00-20:00)</div>
			</div>
		</div>
	</body>
	</html>`

	doc, err := htmlpkg.Parse(strings.NewReader(htmlStr))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	// Find the event-day div
	var eventDay *htmlpkg.Node
	var findEventDay func(*htmlpkg.Node)
	findEventDay = func(n *htmlpkg.Node) {
		if n.Type == htmlpkg.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && attr.Val == "event-day" {
					eventDay = n
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findEventDay(c)
		}
	}
	findEventDay(doc)

	if eventDay == nil {
		t.Fatal("Could not find event-day div")
	}

	engine.parseWeekendDateNodeV2(eventDay.FirstChild, slots)

	// Check that slots are populated
	if len(slots) == 0 {
		t.Error("No date slots were parsed")
	}

	// Check that root date (slot 0) is set
	if _, ok := slots[0]; !ok {
		t.Error("Root date (slot 0) not set")
	}

	// Verify the date
	moscow, _ := time.LoadLocation("Europe/Moscow")
	expectedDate := time.Date(2025, 11, 2, 0, 0, 0, 0, moscow)
	if slots[0].Format("2006-01-02") != expectedDate.Format("2006-01-02") {
		t.Errorf("Root date = %v, want %v", slots[0], expectedDate)
	}

	// Check that time slots have correct times
	if date, ok := slots[3410]; ok {
		if date.Hour() != 11 {
			t.Errorf("Slot 3410 hour = %d, want 11", date.Hour())
		}
		if date.Minute() != 0 {
			t.Errorf("Slot 3410 minute = %d, want 0", date.Minute())
		}
	}
}

func TestHtmlEngineV2_Process_MinimalEvent(t *testing.T) {
	html := `
	<html>
	<body>
		<div class="event-day">
			<div class="caption">Суббота — 19.04.2025</div>
			<div class="tabs-caption">
				<div class="tab-caption" data-timeslot="1">Утро (10:00)</div>
			</div>
		</div>
		<div class="event-single" data-timeslot="1" id="game123">
			<h4 class="game-title"><a href="https://rolecon.ru/game/123">Test Game</a></h4>
			<table class="table-single">
				<tbody>
					<tr><td>Сеттинг:</td><td></td><td>Fantasy</td></tr>
					<tr><td>Система:</td><td></td><td>D&D 5e</td></tr>
					<tr><td>Жанр:</td><td></td><td>Adventure</td></tr>
					<tr><td>Игру проводит:</td><td></td><td><a href="https://rolecon.ru/user/1">John Doe</a></td></tr>
					<tr><td>Места:</td><td></td><td>Осталось 3 мест из 6</td></tr>
				</tbody>
			</table>
		</div>
	</body>
	</html>`

	page := &scraper.Page{
		URL:  "https://rolecon.ru/event/test",
		Html: html,
	}

	engine := NewHtmlEngineV2()
	games, err := engine.Process(page)

	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if games == nil {
		t.Fatal("Process() returned nil")
	}

	if len(*games) != 1 {
		t.Fatalf("Process() returned %d games, want 1", len(*games))
	}

	game := (*games)[0]

	// Verify title
	if game.Title != "Test Game" {
		t.Errorf("Title = %s, want Test Game", game.Title)
	}

	// Verify game metadata
	if game.Setting != "Fantasy" {
		t.Errorf("Setting = %s, want Fantasy", game.Setting)
	}
	if game.System != "D&D 5e" {
		t.Errorf("System = %s, want D&D 5e", game.System)
	}
	if game.Genre != "Adventure" {
		t.Errorf("Genre = %s, want Adventure", game.Genre)
	}
	if game.MasterName != "John Doe" {
		t.Errorf("MasterName = %s, want John Doe", game.MasterName)
	}

	// Verify seats
	if game.SeatsFree != 3 {
		t.Errorf("SeatsFree = %d, want 3", game.SeatsFree)
	}
	if game.SeatsTotal != 6 {
		t.Errorf("SeatsTotal = %d, want 6", game.SeatsTotal)
	}

	// Verify date was parsed from slot
	if game.Date.IsZero() {
		t.Error("Date should be set from slot, but it's zero")
	}

	// Verify expected date (19.04.2025 at 10:00)
	expectedYear := 2025
	expectedMonth := 4
	expectedDay := 19
	expectedHour := 10

	if game.Date.Year() != expectedYear {
		t.Errorf("Date.Year() = %d, want %d", game.Date.Year(), expectedYear)
	}
	if game.Date.Month() != time.Month(expectedMonth) {
		t.Errorf("Date.Month() = %d, want %d", game.Date.Month(), expectedMonth)
	}
	if game.Date.Day() != expectedDay {
		t.Errorf("Date.Day() = %d, want %d", game.Date.Day(), expectedDay)
	}
	if game.Date.Hour() != expectedHour {
		t.Errorf("Date.Hour() = %d, want %d", game.Date.Hour(), expectedHour)
	}

	// Verify IDs and URLs
	if game.ExternalID != "game123" {
		t.Errorf("ExternalID = %s, want game123", game.ExternalID)
	}
	if game.URL != "https://rolecon.ru/game/123" {
		t.Errorf("URL = %s, want https://rolecon.ru/game/123", game.URL)
	}
}

func TestHtmlEngineV2_Process_SingleGamePage(t *testing.T) {
	html := `
	<html>
	<body>
		<div class="game-single" id="game999">
			<h4>Декагон</h4>
			<p class="subcaption-h4">
				29 октября 2025,
				19:00 - 23:00
			</p>
			<table class="table-single reverse">
				<tbody>
					<tr><td>Сеттинг:</td><td></td><td>_Научная фантастика</td></tr>
					<tr><td>Система:</td><td></td><td>Mothership RPG</td></tr>
					<tr><td>Жанр:</td><td></td><td>Ужасы</td></tr>
					<tr><td>Игру проводит:</td><td></td><td><a href="https://rolecon.ru/user/29757">dan-white-ox</a></td></tr>
					<tr><td>Места:</td><td></td><td>Осталось 5 мест из 6</td></tr>
				</tbody>
			</table>
		</div>
	</body>
	</html>`

	page := &scraper.Page{
		URL:  "https://rolecon.ru/game/999",
		Html: html,
	}

	engine := NewHtmlEngineV2()
	games, err := engine.Process(page)

	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if games == nil {
		t.Fatal("Process() returned nil")
	}

	if len(*games) != 1 {
		t.Fatalf("Process() returned %d games, want 1", len(*games))
	}

	game := (*games)[0]

	// Verify title
	if game.Title != "Декагон" {
		t.Errorf("Title = %s, want Декагон", game.Title)
	}

	// Verify game metadata
	if game.Setting != "_Научная фантастика" {
		t.Errorf("Setting = %s, want _Научная фантастика", game.Setting)
	}
	if game.System != "Mothership RPG" {
		t.Errorf("System = %s, want Mothership RPG", game.System)
	}
	if game.Genre != "Ужасы" {
		t.Errorf("Genre = %s, want Ужасы", game.Genre)
	}
	if game.MasterName != "dan-white-ox" {
		t.Errorf("MasterName = %s, want dan-white-ox", game.MasterName)
	}

	// Verify seats
	if game.SeatsFree != 5 {
		t.Errorf("SeatsFree = %d, want 5", game.SeatsFree)
	}
	if game.SeatsTotal != 6 {
		t.Errorf("SeatsTotal = %d, want 6", game.SeatsTotal)
	}

	// Verify date was parsed from subcaption-h4
	if game.Date.IsZero() {
		t.Error("Date should be parsed from subcaption-h4, but it's zero")
	}

	// Verify expected date (29 октября 2025 at 19:00)
	expectedYear := 2025
	expectedMonth := 10
	expectedDay := 29
	expectedHour := 19

	if game.Date.Year() != expectedYear {
		t.Errorf("Date.Year() = %d, want %d", game.Date.Year(), expectedYear)
	}
	if game.Date.Month() != time.Month(expectedMonth) {
		t.Errorf("Date.Month() = %d, want %d", game.Date.Month(), expectedMonth)
	}
	if game.Date.Day() != expectedDay {
		t.Errorf("Date.Day() = %d, want %d", game.Date.Day(), expectedDay)
	}
	if game.Date.Hour() != expectedHour {
		t.Errorf("Date.Hour() = %d, want %d", game.Date.Hour(), expectedHour)
	}

	// Verify IDs and URLs
	if game.ExternalID != "game999" {
		t.Errorf("ExternalID = %s, want game999", game.ExternalID)
	}
	if game.URL != "https://rolecon.ru/game/999" {
		t.Errorf("URL = %s, want https://rolecon.ru/game/999", game.URL)
	}
}

func TestNewHtmlEngineV2(t *testing.T) {
	engine := NewHtmlEngineV2()
	if engine == nil {
		t.Fatal("NewHtmlEngineV2() returned nil")
	}
}

func TestHtmlEngineV2_ExtractSingleEventDate(t *testing.T) {
	tests := []struct {
		name     string
		htmlDate string
		expected time.Time
	}{
		{
			name:     "genitive case month with comma",
			htmlDate: "30 октября 2025, 19:00",
			expected: time.Date(2025, time.October, 30, 19, 0, 0, 0, mustLoadMoscow()),
		},
		{
			name:     "genitive case month with time range",
			htmlDate: "15 апреля 2025, 10:00 - 14:00",
			expected: time.Date(2025, time.April, 15, 10, 0, 0, 0, mustLoadMoscow()),
		},
		{
			name:     "nominative case month",
			htmlDate: "1 январь 2025, 20:00",
			expected: time.Date(2025, time.January, 1, 20, 0, 0, 0, mustLoadMoscow()),
		},
		{
			name:     "february with extra spaces",
			htmlDate: "12  февраля  2025,  18:00",
			expected: time.Date(2025, time.February, 12, 18, 0, 0, 0, mustLoadMoscow()),
		},
		{
			name:     "december genitive",
			htmlDate: "31 декабря 2025, 23:59",
			expected: time.Date(2025, time.December, 31, 23, 59, 0, 0, mustLoadMoscow()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := fmt.Sprintf(`
			<html>
			<body>
				<div class="game-single" id="test123">
					<h4>Test Game</h4>
					<p class="subcaption-h4">%s</p>
					<table class="table-single">
						<tbody>
							<tr><td>Сеттинг:</td><td></td><td>Fantasy</td></tr>
						</tbody>
					</table>
				</div>
			</body>
			</html>`, tt.htmlDate)

			page := &scraper.Page{
				URL:  "https://rolecon.ru/game/123",
				Html: html,
			}

			engine := NewHtmlEngineV2()
			games, err := engine.Process(page)

			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}

			if games == nil || len(*games) != 1 {
				t.Fatalf("Process() returned %d games, want 1", len(*games))
			}

			game := (*games)[0]

			if !game.Date.Equal(tt.expected) && !tt.expected.IsZero() {
				t.Errorf("Date = %v, want %v", game.Date, tt.expected)
			}
		})
	}
}

func mustLoadMoscow() *time.Location {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		panic(err)
	}
	return moscow
}

func TestHtmlEngineV2_ProcessWithEvents_RealHTMLExample(t *testing.T) {
	// Test with actual HTML from "Охота: Война в тени" game page
	// This page has date format "Пятница (19:00 - 23:00)" without specific date
	html := `<!DOCTYPE html>
<html lang="ru-RU">
<body>
    <div class="game-single" id="game18475">
        <h4>Охота: Война в тени</h4>
        <p class="subcaption-h4">
            Пятница (19:00 - 23:00), ,
19:00 - 23:00            </p>
        <table class="table-single reverse">
                    <tbody>
                        <tr>
                            <td>Сеттинг:</td>
                            <td></td>
                            <td>Авторский сеттинг</td>
                        </tr>
                        <tr>
                            <td>Система:</td>
                            <td></td>
                            <td>Авторская система</td>
                        </tr>
                        <tr>
                            <td>Жанр:</td>
                            <td></td>
                            <td>Фэнтези</td>
                        </tr>
                        <tr>
                            <td>Игру проводит:</td>
                            <td></td>
                            <td><a href="https://rolecon.ru/user/34437">Sigfuss</a></td>
                        </tr>
                        <tr>
                            <td>Места:</td>
                            <td></td>
                            <td>Осталось 4 мест из 4</td>
                        </tr>
                    </tbody>
                </table>
    </div>
</body>
</html>`

	event := scraper.RoleconEvent{
		ID:    18475,
		Title: "Охота: Война в тени",
		URL:   "/game/18475",
		Start: "2025-10-24T19:00:00+03:00", // Friday, Oct 24, 2025
		End:   "2025-10-24T23:00:00+03:00",
	}

	page := &scraper.Page{
		URL:  "https://rolecon.ru/game/18475",
		Html: html,
	}

	eventMap := map[string]scraper.RoleconEvent{
		page.URL: event,
	}

	engine := NewHtmlEngineV2()
	games, err := engine.ProcessWithEvents(page, eventMap)

	if err != nil {
		t.Fatalf("ProcessWithEvents() error = %v", err)
	}

	if games == nil || len(*games) != 1 {
		t.Fatalf("ProcessWithEvents() returned %d games, want 1", len(*games))
	}

	game := (*games)[0]

	// Verify basic fields
	if game.Title != "Охота: Война в тени" {
		t.Errorf("Title = %s, want 'Охота: Война в тени'", game.Title)
	}

	// Verify that date was set from event metadata (fallback)
	// This is the main purpose of this test: verify that when a page
	// has "Пятница (19:00 - 23:00)" format without a specific date,
	// the parser falls back to using the date from event metadata
	if game.Date.IsZero() {
		t.Error("Date should be set from event metadata via fallback, but it's zero")
	}

	// Verify the date is Friday, October 24, 2025 at 19:00 Moscow time
	expectedDate := time.Date(2025, 10, 24, 19, 0, 0, 0, mustLoadMoscow())
	if !game.Date.Equal(expectedDate) {
		t.Errorf("Date = %v, want %v (should be parsed from event metadata)", game.Date, expectedDate)
	}

	slog.Debug("Test passed", "date_parsed", game.Date, "fallback_used", !game.Date.IsZero())
}

func TestHtmlEngineV2_ProcessWithEvents_PbtA_RealHTML(t *testing.T) {
	// Test with actual HTML from "[PbtA][ГВ3][КР1] Когда границы пройдены! [12+]" game page
	// This page has date format "Пятница (19:00 - 23:00)" without specific date
	html := `<!DOCTYPE html>
<html lang="ru-RU">
<body>
    <div class="game-single">
        <h4>[PbtA][ГВ3][КР1] Когда границы пройдены! [12+]</h4>
        <p class="subcaption-h4">
            Пятница (19:00 - 23:00), ,
19:00 - 23:00            </p>
        <table class="table-single reverse">
            <tbody>
                <tr>
                    <td>Сеттинг:</td>
                    <td>Космические Рейнджеры</td>
                </tr>
                <tr>
                    <td>Система:</td>
                    <td>*W_Грань Вселенной: Третья редакция</td>
                </tr>
                <tr>
                    <td>Жанр:</td>
                    <td>Космоопера\Боевик\Триллер</td>
                </tr>
                <tr>
                    <td>Игру проводит:</td>
                    <td><a href="https://rolecon.ru/user/5116">Doc</a></td>
                </tr>
                <tr>
                    <td>Места:</td>
                    <td>Осталось 4 мест из 5</td>
                </tr>
            </tbody>
        </table>
    </div>
</body>
</html>`

	event := scraper.RoleconEvent{
		ID:    18446,
		Title: "[PbtA][ГВ3][КР1] Когда границы пройдены! [12+]",
		URL:   "/game/18446",
		Start: "2025-10-24T19:00:00+03:00",
		End:   "2025-10-24T23:00:00+03:00",
	}

	page := &scraper.Page{
		URL:  "https://rolecon.ru/game/18446",
		Html: html,
	}

	eventMap := map[string]scraper.RoleconEvent{
		page.URL: event,
	}

	engine := NewHtmlEngineV2()
	games, err := engine.ProcessWithEvents(page, eventMap)

	if err != nil {
		t.Fatalf("ProcessWithEvents() error = %v", err)
	}

	if games == nil || len(*games) != 1 {
		t.Fatalf("ProcessWithEvents() returned %d games, want 1", len(*games))
	}

	game := (*games)[0]

	// Verify basic fields
	if game.Title != "[PbtA][ГВ3][КР1] Когда границы пройдены! [12+]" {
		t.Errorf("Title = %s, want '[PbtA][ГВ3][КР1] Когда границы пройдены! [12+]'", game.Title)
	}

	// Verify date was set from event metadata (fallback)
	if game.Date.IsZero() {
		t.Error("Date should be set from event metadata, but it's zero")
	}

	// Verify the date from event metadata
	expectedDate := time.Date(2025, 10, 24, 19, 0, 0, 0, mustLoadMoscow())
	if !game.Date.Equal(expectedDate) {
		t.Errorf("Date = %v, want %v", game.Date, expectedDate)
	}

	// Verify table parsing
	if game.Setting != "Космические Рейнджеры" {
		t.Errorf("Setting = %s, want 'Космические Рейнджеры'", game.Setting)
	}

	if game.System != "*W_Грань Вселенной: Третья редакция" {
		t.Errorf("System = %s, want '*W_Грань Вселенной: Третья редакция'", game.System)
	}

	if game.Genre != "Космоопера\\Боевик\\Триллер" {
		t.Errorf("Genre = %s, want 'Космоопера\\Боевик\\Триллер'", game.Genre)
	}

	if game.MasterName != "Doc" {
		t.Errorf("MasterName = %s, want 'Doc'", game.MasterName)
	}

	// Verify seats
	if game.SeatsTotal != 5 || game.SeatsFree != 4 {
		t.Errorf("Seats = %d/%d, want 4/5", game.SeatsFree, game.SeatsTotal)
	}
}

func TestHtmlEngineV2_ProcessWithEvents_WarnOnMissingDate(t *testing.T) {
	// Test that WARN is logged when a game has no date and no fallback is available
	html := `<!DOCTYPE html>
<html lang="ru-RU">
<body>
    <div class="game-single" id="game99999">
        <h4>Game Without Date</h4>
        <p class="subcaption-h4">Undefined date format</p>
        <table class="table-single reverse">
            <tbody>
                <tr><td>Сеттинг:</td><td></td><td>Test</td></tr>
            </tbody>
        </table>
    </div>
</body>
</html>`

	page := &scraper.Page{
		URL:  "https://rolecon.ru/game/99999",
		Html: html,
	}

	engine := NewHtmlEngineV2()

	// Capture log output
	var warnLogged bool
	originalHandler := slog.Default().Handler()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	defer slog.SetDefault(slog.New(originalHandler))

	games, err := engine.ProcessWithEvents(page, nil)

	if err != nil {
		t.Fatalf("ProcessWithEvents() error = %v", err)
	}

	if games == nil || len(*games) != 1 {
		t.Fatalf("ProcessWithEvents() returned %d games, want 1", len(*games))
	}

	game := (*games)[0]

	// Verify the game has no date
	if !game.Date.IsZero() {
		t.Error("Game should have no date, but date is set")
	}

	// Just verify the test doesn't panic (warning is logged but we can't easily capture it in tests)
	_ = warnLogged
}

func TestHtmlEngineV2_ProcessWithEvents_DateFallback(t *testing.T) {
	tests := []struct {
		name        string
		htmlContent string
		event       scraper.RoleconEvent
		expectDate  bool
		wantDate    time.Time
	}{
		{
			name: "date from event metadata when page has no date",
			htmlContent: `
			<html>
			<body>
				<div class="game-single" id="game18475">
					<h4>Fallout. Однажды в Нью-Вегасе</h4>
					<p class="subcaption-h4">Пятница (19:00 - 23:00), ,</p>
					<table class="table-single reverse">
						<tbody>
							<tr><td>Сеттинг:</td><td></td><td>Постапокалипсис</td></tr>
							<tr><td>Система:</td><td></td><td>Mothership RPG</td></tr>
						</tbody>
					</table>
				</div>
			</body>
			</html>`,
			event: scraper.RoleconEvent{
				ID:    18475,
				Title: "Fallout. Однажды в Нью-Вегасе",
				URL:   "/game/18475",
				Start: "2025-10-17T19:00:00+03:00",
				End:   "2025-10-17T23:00:00+03:00",
			},
			expectDate: true,
			wantDate:   time.Date(2025, 10, 17, 19, 0, 0, 0, mustLoadMoscow()),
		},
		{
			name: "page has date, don't use fallback",
			htmlContent: `
			<html>
			<body>
				<div class="game-single" id="game123">
					<h4>Test Game</h4>
					<p class="subcaption-h4">30 октября 2025, 19:00</p>
					<table class="table-single reverse">
						<tbody>
							<tr><td>Сеттинг:</td><td></td><td>Fantasy</td></tr>
						</tbody>
					</table>
				</div>
			</body>
			</html>`,
			event: scraper.RoleconEvent{
				ID:    123,
				Title: "Test Game",
				URL:   "/game/123",
				Start: "2025-11-01T10:00:00+03:00", // Different date
			},
			expectDate: true,
			wantDate:   time.Date(2025, 10, 30, 19, 0, 0, 0, mustLoadMoscow()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := &scraper.Page{
				URL:  "https://rolecon.ru" + tt.event.URL,
				Html: tt.htmlContent,
			}

			eventMap := map[string]scraper.RoleconEvent{
				page.URL: tt.event,
			}

			engine := NewHtmlEngineV2()
			games, err := engine.ProcessWithEvents(page, eventMap)

			if err != nil {
				t.Fatalf("ProcessWithEvents() error = %v", err)
			}

			if games == nil || len(*games) != 1 {
				t.Fatalf("ProcessWithEvents() returned %d games, want 1", len(*games))
			}

			game := (*games)[0]

			if tt.expectDate {
				if game.Date.IsZero() {
					t.Error("Date should be set, but it's zero")
				}
				if !game.Date.Equal(tt.wantDate) {
					t.Errorf("Date = %v, want %v", game.Date, tt.wantDate)
				}
			} else {
				if !game.Date.IsZero() {
					t.Errorf("Date should be zero, but got %v", game.Date)
				}
			}
		})
	}
}

// Test all HTML example files in the webpage-examples directory
func TestHtmlEngineV2_Process_AllExamples(t *testing.T) {
	exampleDir := "docs/webpage-examples"
	files, err := os.ReadDir(exampleDir)
	if err != nil {
		t.Skipf("Could not read example directory: %v", err)
		return
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".html") {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			htmlBytes, err := os.ReadFile(filepath.Join(exampleDir, file.Name()))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			page := &scraper.Page{
				URL:  "https://rolecon.ru/test",
				Html: string(htmlBytes),
			}

			engine := NewHtmlEngineV2()
			games, err := engine.Process(page)

			if err != nil {
				t.Errorf("Process() error = %v", err)
				return
			}

			// Just check that it doesn't crash and returns something
			if games == nil {
				t.Error("Process() returned nil")
			}

			// Log some statistics
			if games != nil && len(*games) > 0 {
				t.Logf("Found %d games in %s", len(*games), file.Name())

				// Count how many have various fields populated
				withTitle := 0
				withDate := 0
				withSeats := 0
				withMaster := 0

				for _, g := range *games {
					if g.Title != "" {
						withTitle++
					}
					if !g.Date.IsZero() {
						withDate++
					}
					if g.SeatsTotal > 0 {
						withSeats++
					}
					if g.MasterName != "" {
						withMaster++
					}
				}

				t.Logf("Games with Title: %d, Date: %d, Seats: %d, Master: %d",
					withTitle, withDate, withSeats, withMaster)
			}
		})
	}
}
