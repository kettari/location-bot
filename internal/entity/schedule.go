package entity

import (
	"fmt"
	"strings"
	"time"
)

type Schedule struct {
	Games []Game `json:"games"`
}

func NewSchedule() *Schedule {
	return &Schedule{}
}

func (s Schedule) Format() ([]string, error) {
	var result []string

	dow := map[string]string{
		"Mon": "ПОНЕДЕЛЬНИК",
		"Tue": "ВТОРНИК",
		"Wed": "СРЕДА",
		"Thu": "ЧЕТВЕРГ",
		"Fri": "ПЯТНИЦА",
		"Sat": "СУББОТА",
		"Sun": "ВОСКРЕСЕНЬЕ",
	}

	currentDate := ""
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return result, err
	}
	slice := "Игры, на которые можно записаться:"
	for _, game := range s.Games {
		gameDate := fmt.Sprintf("<b>%s</b> (%s, %s)",
			dow[game.Date.In(moscow).Format("Mon")],
			game.Date.In(moscow).Format("02.01"),
			game.Date.In(moscow).Format("15:04"))
		if currentDate != gameDate {
			currentDate = gameDate
			slice += "\n\n" + gameDate
		}
		record := fmt.Sprintf("🔹 %d/%d <a href=\"%s\">%s</a> [%s; %s]",
			game.SeatsFree,
			game.SeatsTotal,
			game.URL,
			game.Title,
			game.System,
			game.Setting,
		)

		slice += "\n" + record

		if len(slice) > 4000 {
			result = append(result, slice)
			slice = ""
		}
	}
	if len(strings.Trim(slice, " \n\r\t")) > 0 {
		result = append(result, slice)
	}

	return result, nil
}
