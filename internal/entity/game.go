package entity

import (
	"fmt"
	"gorm.io/gorm"
	"time"
)

type CalendarEventType string

const (
	CalendarEventWeekday = "weekday"
	CalendarEventWeekend = "weekend"
)

type Game struct {
	gorm.Model
	ExternalID       string    `json:"id" gorm:"unique;not null"`
	Joinable         bool      `json:"joinable" gorm:"default:false;not null"`
	URL              string    `json:"url" gorm:"size:1024"`
	Title            string    `json:"title" gorm:"size:1024"`
	Date             time.Time `json:"date"`
	Setting          string    `json:"setting" gorm:"size:100"`
	System           string    `json:"system" gorm:"size:100"`
	Genre            string    `json:"genre" gorm:"size:100"`
	MasterName       string    `json:"master_name" gorm:"size:100"`
	MasterLink       string    `json:"master_link" gorm:"size:1024"`
	Description      string    `json:"description"`
	Notes            string    `json:"notes"`
	SeatsTotal       int       `json:"seats_total" gorm:"default:0;not null"`
	SeatsFree        int       `json:"seats_free" gorm:"default:0;not null"`
	NotificationSent bool      `json:"-" gorm:"default:false"`
	Slot             int       `json:"-" gorm:"-:all"`

	// Observers
	observerList []Observer
}

var dow = map[string]string{
	"Mon": "ПОНЕДЕЛЬНИК",
	"Tue": "ВТОРНИК",
	"Wed": "СРЕДА",
	"Thu": "ЧЕТВЕРГ",
	"Fri": "ПЯТНИЦА",
	"Sat": "СУББОТА",
	"Sun": "ВОСКРЕСЕНЬЕ",
}

func (g *Game) EqualDate(game *Game) bool {
	return g.Date.In(time.UTC).String() == game.Date.In(time.UTC).String()
}

// NewJoinable returns true if new game is in the future and joinable
func (g *Game) NewJoinable() bool {
	return g.Date.After(time.Now()) && g.Joinable && g.SeatsFree > 0
}

// FreeSeatsAdded returns true if number of free seats was zero, now game is in the future,
// is joinable and has positive free seats
func (g *Game) FreeSeatsAdded(game *Game) bool {
	return g.Date.After(time.Now()) && g.Joinable && g.SeatsFree > 0 && game.SeatsFree == 0
}

// BecomeJoinable returns true if game was not joinable, now game is in the future,
// is joinable and has positive free seats
func (g *Game) BecomeJoinable(game *Game) bool {
	return g.Date.After(time.Now()) && g.Joinable && g.SeatsFree > 0 && !game.Joinable
}

// WasJoinable returns true if game was joinable. Used for cancellation checks
func (g *Game) WasJoinable() bool {
	return g.Date.After(time.Now()) && g.SeatsTotal > 0
}

func (g *Game) FormatNew() string {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		panic(err)
	}

	result := fmt.Sprintf("<b>%s</b> (%s, %s)",
		dow[g.Date.In(moscow).Format("Mon")],
		g.Date.In(moscow).Format("02.01"),
		g.Date.In(moscow).Format("15:04"))

	result += fmt.Sprintf("\n%d/%d <a href=\"%s\">%s</a> [%s; %s]",
		g.SeatsFree,
		g.SeatsTotal,
		g.URL,
		g.Title,
		g.System,
		g.Setting)

	return result
}

func (g *Game) FormatCancelled() string {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		panic(err)
	}

	result := fmt.Sprintf("Игра отменена:\n\n<b>%s</b> (%s, %s)",
		dow[g.Date.In(moscow).Format("Mon")],
		g.Date.In(moscow).Format("02.01"),
		g.Date.In(moscow).Format("15:04"))

	result += fmt.Sprintf("\n%s [%s; %s]",
		g.Title,
		g.System,
		g.Setting)

	return result
}

func (g *Game) Register(observer Observer) {
	g.observerList = append(g.observerList, observer)
}

func (g *Game) notifyAll(subject SubjectType) {
	for _, observer := range g.observerList {
		observer.Update(g, subject)
	}
}

func (g *Game) OnNew() {
	g.notifyAll(SubjectTypeNew)
}

func (g *Game) OnBecomeJoinable() {
	g.notifyAll(SubjectTypeBecomeJoinable)
}

func (g *Game) OnCancelled() {
	g.notifyAll(SubjectTypeCancelled)
}
