package parser

import (
	"log/slog"
	"testing"
	"time"

	"github.com/kettari/location-bot/internal/scraper"
)

func init() {
	// Set up logger for tests
	slog.SetLogLoggerLevel(slog.LevelDebug)
}

func TestHtmlEngine_Process_SingleEvent(t *testing.T) {
	tests := []struct {
		name    string
		html    string
		url     string
		wantLen int
		wantErr bool
	}{
		{
			name: "single event with all details",
			html: `
			<html>
			<body>
				<div class="event-day">
					<div class="caption">Суббота — 19.04.2025</div>
					<div class="tabs-caption">
						<div class="tab-caption" data-timeslot="1">Утро (10:00</div>
					</div>
				</div>
				<div class="event-single" data-timeslot="1">
					<h4 class="game-title"><a href="/event/123">Test Game</a></h4>
					<table class="table-single">
						<tbody>
							<tr><td>Сеттинг:</td><td></td><td>Fantasy</td></tr>
							<tr><td>Система:</td><td></td><td>D&D 5e</td></tr>
							<tr><td>Жанр:</td><td></td><td>Adventure</td></tr>
							<tr><td>Игру проводит:</td><td></td><td><a href="/user/1">John Doe</a></td></tr>
							<tr><td>Места:</td><td></td><td>Осталось 3 мест из 6</td></tr>
						</tbody>
					</table>
				</div>
			</body>
			</html>`,
			url:     "https://rolecon.ru/event/123",
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "weekend event with multiple slots",
			html: `
			<html>
			<body>
				<div class="event-day">
					<div class="caption">Суббота — 20.04.2025</div>
					<div class="tabs-caption">
						<div class="tab-caption" data-timeslot="1">Утро (10:00</div>
						<div class="tab-caption" data-timeslot="2">День (14:00</div>
					</div>
				</div>
				<div class="event-single" data-timeslot="1">
					<h4 class="game-title"><a href="/event/456">Morning Game</a></h4>
					<table class="table-single">
						<tbody>
							<tr><td>Сеттинг:</td><td></td><td>Modern</td></tr>
							<tr><td>Места:</td><td></td><td>Осталось 4 мест из 5</td></tr>
						</tbody>
					</table>
				</div>
				<div class="event-single" data-timeslot="2">
					<h4 class="game-title"><a href="/event/789">Afternoon Game</a></h4>
					<table class="table-single">
						<tbody>
							<tr><td>Сеттинг:</td><td></td><td>Sci-Fi</td></tr>
							<tr><td>Места:</td><td></td><td>Осталось 2 мест из 4</td></tr>
						</tbody>
					</table>
				</div>
			</body>
			</html>`,
			url:     "https://rolecon.ru/weekend",
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "empty html",
			html:    "<html><body></body></html>",
			url:     "https://rolecon.ru/empty",
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "event without seat information",
			html: `
			<html>
			<body>
				<div class="event-day">
					<div class="caption">Понедельник — 21.04.2025</div>
					<div class="tabs-caption">
						<div class="tab-caption" data-timeslot="1">Вечер (20:00</div>
					</div>
				</div>
				<div class="event-single" data-timeslot="1">
					<h4 class="game-title"><a href="/event/999">No Seats Game</a></h4>
					<table class="table-single">
						<tbody>
							<tr><td>Сеттинг:</td><td></td><td>Horror</td></tr>
						</tbody>
					</table>
				</div>
			</body>
			</html>`,
			url:     "https://rolecon.ru/event/999",
			wantLen: 1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := &scraper.Page{
				URL:  tt.url,
				Html: tt.html,
			}

			engine := NewHtmlEngine()
			games, err := engine.Process(page)

			if (err != nil) != tt.wantErr {
				t.Errorf("Process() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if games == nil {
				if tt.wantLen != 0 {
					t.Error("Process() returned nil, expected games")
				}
				return
			}

			if len(*games) != tt.wantLen {
				t.Errorf("Process() returned %d games, want %d", len(*games), tt.wantLen)
			}

			// Verify first game if expected
			if tt.wantLen > 0 && len(*games) > 0 {
				game := (*games)[0]
				if game.URL == "" {
					t.Error("Process() game.URL is empty")
				}
			}
		})
	}
}

func TestHtmlEngine_Process_DateTimeParsing(t *testing.T) {
	now := time.Now()
	futureDate := now.Add(48 * time.Hour)

	tests := []struct {
		name        string
		html        string
		wantFuture  bool
		wantJoinable bool
	}{
		{
			name: "future date with free seats",
			html: `
			<html>
			<body>
				<div class="event-day">
					<div class="caption">Суббота — ` + futureDate.Format("02.01.2006") + `</div>
					<div class="tabs-caption">
						<div class="tab-caption" data-timeslot="1">Утро (10:00</div>
					</div>
				</div>
				<div class="event-single" data-timeslot="1">
					<h4>Future Game</h4>
					<table class="table-single">
						<tbody>
							<tr><td>Места:</td><td></td><td>Осталось 5 мест из 6</td></tr>
						</tbody>
					</table>
				</div>
			</body>
			</html>`,
			wantFuture:  true,
			wantJoinable: true,
		},
		{
			name: "past date",
			html: `
			<html>
			<body>
				<div class="event-day">
					<div class="caption">Понедельник — ` + now.Add(-48*time.Hour).Format("02.01.2006") + `</div>
					<div class="tabs-caption">
						<div class="tab-caption" data-timeslot="1">Вечер (20:00</div>
					</div>
				</div>
				<div class="event-single" data-timeslot="1">
					<h4>Past Game</h4>
					<table class="table-single">
						<tbody>
							<tr><td>Места:</td><td></td><td>Осталось 2 мест из 4</td></tr>
						</tbody>
					</table>
				</div>
			</body>
			</html>`,
			wantFuture:  false,
			wantJoinable: false,
		},
		{
			name: "future date no seats",
			html: `
			<html>
			<body>
				<div class="event-day">
					<div class="caption">Вторник — ` + futureDate.Format("02.01.2006") + `</div>
					<div class="tabs-caption">
						<div class="tab-caption" data-timeslot="1">День (14:00</div>
					</div>
				</div>
				<div class="event-single" data-timeslot="1">
					<h4>Full Game</h4>
					<table class="table-single">
						<tbody>
							<tr><td>Места:</td><td></td><td>Осталось 0 мест из 5</td></tr>
						</tbody>
					</table>
				</div>
			</body>
			</html>`,
			wantFuture:  true,
			wantJoinable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := &scraper.Page{
				URL:  "https://rolecon.ru/test",
				Html: tt.html,
			}

			engine := NewHtmlEngine()
			games, err := engine.Process(page)
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}

			if len(*games) == 0 {
				t.Fatal("Process() returned 0 games")
			}

			game := (*games)[0]
			if game.Date.After(time.Now()) != tt.wantFuture {
				t.Errorf("Process() game.Date.After(now) = %v, want %v", game.Date.After(time.Now()), tt.wantFuture)
			}
			if game.Joinable != tt.wantJoinable {
				t.Errorf("Process() game.Joinable = %v, want %v", game.Joinable, tt.wantJoinable)
			}
		})
	}
}

func TestNewHtmlEngine(t *testing.T) {
	engine := NewHtmlEngine()
	if engine == nil {
		t.Fatal("NewHtmlEngine() returned nil")
	}
}

