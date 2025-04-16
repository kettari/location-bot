package console

import (
	"encoding/json"
	"github.com/kettari/location-bot/internal/chatgpt"
	"github.com/kettari/location-bot/internal/config"
	"github.com/kettari/location-bot/internal/entity"
	"github.com/kettari/location-bot/internal/scraper"
	"github.com/kettari/location-bot/internal/storage"
	"log/slog"
)

const (
	rootURL   = "https://rolecon.ru"
	eventsURL = "https://rolecon.ru/gamesearch"
)

type ScheduleFetchCommand struct {
}

func NewScheduleFetchCommand() *ScheduleFetchCommand {
	cmd := ScheduleFetchCommand{}
	return &cmd
}

func (cmd *ScheduleFetchCommand) Name() string {
	return "schedule:fetch"
}

func (cmd *ScheduleFetchCommand) Description() string {
	return "fetches events from the Rolecon server and parses them to the database"
}

func (cmd *ScheduleFetchCommand) Run() error {
	slog.Info("Requesting page", "url", rootURL)
	conf := config.GetConfig()

	page := scraper.NewPage(rootURL)
	if err := page.LoadHtml(); err != nil {
		return err
	}
	slog.Info("Initial page loaded", "size", len(page.Html), "cookies_count", len(page.Cookies))

	csrf := scraper.NewCsrf(page)
	var err error
	if err = csrf.ExtractCsrfToken(); err != nil {
		return err
	}
	slog.Info("Found CSRF token", "token", csrf.Token)

	if err = csrf.ExtractCsrfCookie(); err != nil {
		return err
	}
	slog.Info("Found CSRF cookie", "cookie", csrf.Cookie)

	slog.Info("Requesting events", "url", eventsURL)
	events := scraper.NewEvents(eventsURL, csrf)
	if err = events.LoadEvents(); err != nil {
		return err
	}
	slog.Info("Events page loaded", "size", len(events.Html))

	if err = events.BreakDown(); err != nil {
		return err
	}
	slog.Info("Broke down events", "events_count", len(events.Parts))
	chunks := events.Rejoin()
	slog.Info("Events split to chunks", "chunks_count", len(chunks))
	var parsedJson []string
	if len(chunks) > 0 {
		for _, chunk := range chunks {

			slog.Info("Processing events chunk", "chunk_size", len(chunk))

			// Ask ChatGPT to parse piece of the events HTML to JSON
			chatGPT := chatgpt.NewChatGPT(conf.OpenAIApiKey, conf.OpenAILanguageModel)
			var jsonBuf *string
			if jsonBuf, err = chatGPT.NewParseCompletion(chunk); err != nil {
				return err
			}
			if jsonBuf != nil {
				parsedJson = append(parsedJson, *jsonBuf)
				slog.Info("Chunk parsed to JSON", "json_size", len(*jsonBuf))
				slog.Debug("Chunk internals", "json", *jsonBuf)
			} else {
				slog.Warn("Events chunk is empty")
			}

		}
	}
	slog.Info("Finished parsing events HTML to JSON", "json_parts_count", len(parsedJson))

	// Debug-time crutch
	/*parsedJson := []string{
		"{\n\t\"games\": [\n\t\t{\n\t\t\t\"id\": \"game123\",\n\t\t\t\"joinable\": true,\n\t\t\t\"url\": \"https://rolecon.ru/path\",\n\t\t\t\"title\": \"Название игры 1\",\n\t\t\t\"date\": \"2025-04-20T11:00:00+03:00\",\n\t\t\t\"setting\": \"Eberron\",\n\t\t\t\"system\": \"D&D 2024\",\n\t\t\t\"genre\": \"Экшн, расследование.\",\n\t\t\t\"master_name\": \"kauzt\",\n\t\t\t\"master_link\": \"https://rolecon.ru/user/24001\",\n\t\t\t\"description\": \"Когда заточённые в подземелье хтонические существа из другой реальности решают объединиться, жители поверхности сначала теряются, а потом — находят самых неожиданных союзников.\",\n\t\t\t\"notes\": \"Ваншот из серии ваншотов\",\n\t\t\t\"seats_total\": 6,\n\t\t\t\"seats_free\": 0\n\t\t},\n\t\t{\n\t\t\t\"id\": \"game456\",\n\t\t\t\"joinable\": false,\n\t\t\t\"url\": \"https://rolecon.ru/path\",\n\t\t\t\"title\": \"Название игры 2\",\n\t\t\t\"date\": \"2025-04-20T11:00:00+03:00\",\n\t\t\t\"setting\": \"Авторский сеттинг\",\n\t\t\t\"system\": \"D&D 2024\",\n\t\t\t\"genre\": \"триллер на выживание\",\n\t\t\t\"master_name\": \"Tindomerel\",\n\t\t\t\"master_link\": \"https://rolecon.ru/user/3647\",\n\t\t\t\"description\": \"Партия набрана\",\n\t\t\t\"notes\": \"4+мастер\",\n\t\t\t\"seats_total\": 0,\n\t\t\t\"seats_free\": 0\n\t\t}\n\t]\n}",
	}*/

	// Store events
	manager := storage.NewManager(conf.DbConnectionString)
	if err := manager.Connect(); err != nil {
		return err
	}

	slog.Info("Unmarshalling JSON(s) to the struct")
	schedule := entity.NewSchedule()
	for _, jsonChunk := range parsedJson {
		bufSchedule := entity.NewSchedule()
		if err = json.Unmarshal([]byte(jsonChunk), bufSchedule); err != nil {
			return err
		}
		schedule.Games = append(schedule.Games, bufSchedule.Games...)
	}
	slog.Info("Unmarshalled JSON(s) to schedule struct", "schedule_length", len(schedule.Games))

	for _, game := range schedule.Games {
		slog.Info("Saving the game", "game_external_id", game.ExternalID)
		slog.Debug("Game JSON internals", "game_json", game)
		storedGame := game
		result := manager.DB().Where(entity.Game{ExternalID: game.ExternalID}).FirstOrCreate(&storedGame)
		if result.Error != nil {
			return result.Error
		}
		if !game.Equal(&storedGame) {
			game.ID = storedGame.ID
			game.NotificationSent = storedGame.NotificationSent
			game.Changed = true
			tx := manager.DB().Save(&game)
			if tx.Error != nil {
				return tx.Error
			}
			slog.Info("Event was changed, updated in the database", "game_external_id", game.ExternalID)
		}
	}

	slog.Info("Finished updating the schedule")

	return nil
}
