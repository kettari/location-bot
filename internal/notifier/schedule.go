package notifier

import (
	"errors"
	"fmt"
	"github.com/kettari/location-bot/internal/entity"
	"github.com/kettari/location-bot/internal/storage"
	"gorm.io/gorm"
	"log/slog"
	"strings"
	"time"
)

type Schedule struct {
	manager *storage.Manager
	Games   []entity.Game `json:"games"`
}

func NewSchedule(manager *storage.Manager) *Schedule {
	return &Schedule{manager: manager}
}

func (s *Schedule) Format() ([]string, error) {
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
		record := fmt.Sprintf("🔸 %d/%d <a href=\"%s\">%s</a> [%s; %s]",
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

// LoadJoinableEvents loads future joinable games
func (s *Schedule) LoadJoinableEvents() error {
	if s.manager == nil {
		return errors.New("manager not initialized")
	}

	if err := s.manager.Connect(); err != nil {
		return err
	}
	if result := s.manager.DB().
		Where(&entity.Game{Joinable: true}).
		Where("date > ?", time.Now()).
		Order("date ASC").
		Find(&s.Games); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			slog.Warn("No joinable future games found, exiting")
			return nil
		}
		return result.Error
	}
	slog.Debug("Found joinable future games", "games_count", len(s.Games))

	return nil
}

// LoadUnnotifiedEvents loads future joinable games which were not notified about
func (s *Schedule) LoadUnnotifiedEvents() error {
	if s.manager == nil {
		return errors.New("manager not initialized")
	}

	if err := s.manager.Connect(); err != nil {
		return err
	}
	if result := s.manager.DB().
		Where(&entity.Game{Joinable: true}).
		Where("notification_sent = ?", false).
		Where("date > ?", time.Now()).
		Order("date ASC").
		Find(&s.Games); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			slog.Info("No joinable unnotified games found, exiting")
			return nil
		}
		return result.Error
	}
	slog.Debug("Found joinable unnotified games", "games_count", len(s.Games))

	return nil
}

// MarkAsNotified marks games unchanged and notified
func (s *Schedule) MarkAsNotified() error {
	if s.manager == nil {
		return errors.New("manager not initialized")
	}

	for k, _ := range s.Games {
		s.Games[k].NotificationSent = true
		if err := s.manager.DB().Save(&s.Games[k]).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *Schedule) CheckAbsentGames() error {
	if s.manager == nil {
		return errors.New("manager not initialized")
	}

	// Check for absent games
	var storedGames []entity.Game
	if result := s.manager.DB().
		Where(&entity.Game{Joinable: true}).
		Where("date > ?", time.Now()).
		Order("date ASC").
		Find(&storedGames); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			slog.Warn("No joinable future games found, exiting")
			return nil
		}
		return result.Error
	}
	for _, sg := range storedGames {
		found := false
		for _, jg := range s.Games {
			if jg.ExternalID == sg.ExternalID {
				found = true
				break
			}
		}
		if !found {
			slog.Warn("Stored game is absent", "game_id", sg.ExternalID)
			sg.Joinable = false
			if err := s.manager.DB().Save(&sg).Error; err != nil {
				return err
			}
		}
	}
}

func (s *Schedule) SaveGames() error {
	if s.manager == nil {
		return errors.New("manager not initialized")
	}

	// Save collection
	for _, game := range s.Games {
		slog.Info("Saving the game", "game_external_id", game.ExternalID)

		storedGame := game
		result := s.manager.DB().Where(entity.Game{ExternalID: game.ExternalID}).FirstOrCreate(&storedGame)
		if result.Error != nil {
			return result.Error
		}

		game.NotificationSent = storedGame.NotificationSent
		if !game.Equal(&storedGame) {
			//game.NotificationSent = false
			slog.Warn("Event significant properties changed", "game_external_id", game.ExternalID,
				"old_date", storedGame.Date.In(time.UTC).String(), "new_date", game.Date.In(time.UTC).String(),
				"old_joinable", storedGame.Joinable, "new_joinable", game.Joinable,
			)
		}
		game.ID = storedGame.ID
		if err := s.manager.DB().Save(&game).Error; err != nil {
			return err
		}
		slog.Debug("Game internals", "game", game)
	}

	return nil
}
