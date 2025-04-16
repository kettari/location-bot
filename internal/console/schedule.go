package console

import (
	"github.com/kettari/location-bot/internal/chatgpt"
	"github.com/kettari/location-bot/internal/config"
	"github.com/kettari/location-bot/internal/scraper"
	"log/slog"
)

const (
	rootURL   = "https://rolecon.ru"
	eventsURL = "https://rolecon.ru/gamesearch"
)

type ScheduleCommand struct {
}

func NewScheduleCommand() *ScheduleCommand {
	cmd := ScheduleCommand{}
	return &cmd
}

func (cmd *ScheduleCommand) Name() string {
	return "schedule:fetch"
}

func (cmd *ScheduleCommand) Description() string {
	return "Dummy command for help"
}

func (cmd *ScheduleCommand) Run() error {
	slog.Info("Requesting page", "url", rootURL)

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
		conf := config.GetConfig()
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

	return nil
}

func firstN(s string, n int) string {
	i := 0
	for j := range s {
		if i == n {
			return s[:j]
		}
		i++
	}
	return s
}
