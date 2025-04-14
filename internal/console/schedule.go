package console

import (
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

	page := &scraper.Page{}
	if err := page.LoadHtml(rootURL); err != nil {
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
	events := &scraper.Events{}
	if err = events.LoadEvents(csrf, eventsURL); err != nil {
		return err
	}
	slog.Info("Events page loaded", "size", len(events.Html))

	return nil
}
