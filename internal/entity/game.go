package entity

import (
	"gorm.io/gorm"
	"time"
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
	Changed          bool      `json:"-" gorm:"default:true"`
}

func (g *Game) Equal(game *Game) bool {
	return g.Joinable == game.Joinable &&
		g.URL == game.URL &&
		g.Title == game.Title &&
		g.Date.String() == game.Date.String() &&
		g.Setting == game.Setting &&
		g.System == game.System &&
		g.Genre == game.Genre &&
		g.MasterName == game.MasterName &&
		g.MasterLink == game.MasterLink &&
		g.Description == game.Description &&
		g.Notes == game.Notes &&
		g.SeatsTotal == game.SeatsTotal &&
		g.SeatsFree == game.SeatsFree
}
