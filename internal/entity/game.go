package entity

import (
	"gorm.io/gorm"
	"time"
)

type Game struct {
	gorm.Model
	ExternalID  string    `json:"id" gorm:"unique;not null"`
	Joinable    bool      `json:"joinable;not null"`
	URL         string    `json:"url" gorm:"size:1024"`
	Title       string    `json:"title" gorm:"size:1024"`
	Date        time.Time `json:"date"`
	Setting     string    `json:"setting" gorm:"size:100"`
	System      string    `json:"system" gorm:"size:100"`
	Genre       string    `json:"genre" gorm:"size:100"`
	MasterName  string    `json:"master_name" gorm:"size:100"`
	MasterLink  string    `json:"master_link" gorm:"size:1024"`
	Description string    `json:"description"`
	Notes       string    `json:"notes"`
	SeatsTotal  int       `json:"seats_total" gorm:"default:0;not null"`
	SeatsFree   int       `json:"seats_free" gorm:"default:0;not null"`
}
