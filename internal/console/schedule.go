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

	/*slog.Info("Extracting game IDs")
	if err = events.ExtractID(); err != nil {
		return err
	}*/
	conf := config.GetConfig()
	chatGPT := chatgpt.NewChatGPT(conf.OpenAIApiKey, conf.OpenAILanguageModel)
	var parsedEvents *string

	if parsedEvents, err = chatGPT.NewParseCompletion(firstN(events.Html, 50000)); err != nil {
		return err
	}
	slog.Info("Received parsed events", "size", len(*parsedEvents))
	slog.Debug("Parsed events", "events", *parsedEvents)

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
