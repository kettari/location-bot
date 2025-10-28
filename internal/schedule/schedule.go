package schedule

import (
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/kettari/location-bot/internal/bot"
	"github.com/kettari/location-bot/internal/config"
	"github.com/kettari/location-bot/internal/entity"
	"github.com/kettari/location-bot/internal/storage"
	"gorm.io/gorm"
)

type Schedule struct {
	manager *storage.Manager
	Games   []entity.Game `json:"games"`
}

func NewSchedule(manager *storage.Manager) *Schedule {
	return &Schedule{manager: manager}
}

func (s *Schedule) Add(games ...entity.Game) {
	s.Games = append(s.Games, games...)
}

// Format returns a formatted message list for games.
// If no games are available, returns ["Открытых игр для записи на сайте нет."]
func (s *Schedule) Format() ([]string, error) {
	var result []string

	// If no games, return specific message
	if len(s.Games) == 0 {
		return []string{"Открытых игр для записи на сайте нет."}, nil
	}

	// Sort games: by date (ascending), then by free seats (descending), then by title (ascending)
	sort.Slice(s.Games, func(i, j int) bool {
		// First: sort by date ascending
		if s.Games[i].Date.Before(s.Games[j].Date) {
			return true
		}
		if s.Games[i].Date.After(s.Games[j].Date) {
			return false
		}

		// Second: if same time, sort by free seats descending (most free first)
		if s.Games[i].SeatsFree != s.Games[j].SeatsFree {
			return s.Games[i].SeatsFree > s.Games[j].SeatsFree
		}

		// Third: if same free seats, sort by title ascending
		return s.Games[i].Title < s.Games[j].Title
	})

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
			slog.Warn("no joinable future games found, exiting")
			return nil
		}
		return result.Error
	}
	slog.Debug("found joinable future games", "games_count", len(s.Games))

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
			slog.Info("no joinable unnotified games found, exiting")
			return nil
		}
		return result.Error
	}
	slog.Debug("found joinable unnotified games", "games_count", len(s.Games))

	return nil
}

func (s *Schedule) CheckAbsentGames() error {
	conf := config.GetConfig()

	if conf.DryRun && s.manager == nil {
		slog.Info("DRY RUN MODE: skipping check for absent games")
		return nil
	}

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
			slog.Warn("no joinable future games found, exiting")
			return nil
		}
		return result.Error
	}
	// Register observers
	b, err := bot.CreateBot(conf.BotToken, conf.NotificationChatID)
	if err != nil {
		slog.Error("unable to create bot processor object", "error", err)
		return err
	}
	for k := range storedGames {
		storedGames[k].Register(entity.CancelledGameObserver(b))
	}

	if conf.DryRun {
		slog.Info("DRY RUN MODE: skipping database updates for absent games")
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
			slog.Warn("stored game is absent", "game_id", sg.ExternalID)
			sg.Joinable = false
			if !conf.DryRun {
				if err := s.manager.DB().Save(&sg).Error; err != nil {
					return err
				}
			}
			slog.Debug("cancelled game internals", "game", sg)
			if sg.WasJoinable() {
				sg.OnCancelled()
			}
		}
	}

	return nil
}

func (s *Schedule) SaveGames() error {
	conf := config.GetConfig()
	if conf.DryRun {
		slog.Info("DRY RUN MODE: skipping database saves")
		if s.manager == nil {
			// DryRun mode without DB - just simulate events
			for _, game := range s.Games {
				if game.NewJoinable() {
					game.OnNew()
				} else if game.SeatsFree > 0 {
					game.OnBecomeJoinable()
				}
			}
			return nil
		}
		// Still trigger observers for logging, but they won't send messages in DryRun
		for _, game := range s.Games {
			storedGame := game
			result := s.manager.DB().Where(entity.Game{ExternalID: game.ExternalID}).First(&storedGame)
			if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return result.Error
			}
			freshGame := errors.Is(result.Error, gorm.ErrRecordNotFound)

			if freshGame && game.NewJoinable() {
				game.OnNew()
			} else if game.FreeSeatsAdded(&storedGame) || game.BecomeJoinable(&storedGame) {
				game.OnBecomeJoinable()
			}
		}
		return nil
	}

	if s.manager == nil {
		return errors.New("manager not initialized")
	}

	// Save collection
	for _, game := range s.Games {
		slog.Debug("saving the game", "game_external_id", game.ExternalID)

		// Identify new games to fire event later
		storedGame := game
		result := s.manager.DB().Where(entity.Game{ExternalID: game.ExternalID}).First(&storedGame)
		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}
		freshGame := errors.Is(result.Error, gorm.ErrRecordNotFound)

		// Find or create record for the game in the DB
		result = s.manager.DB().Where(entity.Game{ExternalID: game.ExternalID}).FirstOrCreate(&storedGame)
		if result.Error != nil {
			return result.Error
		}

		game.ID = storedGame.ID
		if err := s.manager.DB().Save(&game).Error; err != nil {
			return err
		}

		// Select event
		if freshGame && game.NewJoinable() {
			game.OnNew()
		} else if game.FreeSeatsAdded(&storedGame) || game.BecomeJoinable(&storedGame) {
			game.OnBecomeJoinable()
		}
	}

	return nil
}
