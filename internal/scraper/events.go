package scraper

import (
	"errors"
	"github.com/kettari/location-bot/internal/entity"
	"io"
	"log/slog"
	"net/http"
	"regexp"
)

type Events struct {
	URL      string
	Csrf     *Csrf
	Html     string
	Schedule *entity.Schedule
}

func NewEvents(url string, csrf *Csrf) *Events {
	return &Events{URL: url, Csrf: csrf}
}

func (e *Events) LoadEvents() error {
	req, err := http.NewRequest("POST", e.URL, nil)
	if err != nil {
		return err
	}

	cookie := http.Cookie{
		Name:     "_csrf",
		Value:    e.Csrf.Cookie,
		Path:     "/",
		HttpOnly: true,
	}
	req.Header.Set("Cookie", cookie.String())
	req.Header.Set("x-csrf-token", e.Csrf.Token)
	req.Header.Set("x-requested-with", "XMLHttpRequest")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			slog.Error("Failed to close response body", "url", e.URL, "err", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		slog.Error("Failed to load events HTML page", "url", e.URL, "http_code", resp.StatusCode, "status", resp.Status)
		return errors.New("failed to load events HTML page")
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	e.Html = string(data)

	return nil
}

func (e *Events) ExtractID() error {
	e.Schedule = entity.NewSchedule()
	r := regexp.MustCompile(`<div class="event-single\s+[a-z\s]+(can-join|cannot-join)\s+"\s+id="(game\d+)"[^<]+<h4 class="game-title">\s.+\s<a href="([^"]+)" class='js-game-title'>\s([^<]+)\s\(\d{2}:\d{2}\s.\s\d{2}:\d{2}\s\)`)
	matches := r.FindAllStringSubmatch(e.Html, -1)
	if len(matches) > 0 {
		for _, match := range matches {
			game := entity.NewGame(match[2], match[1], match[3], match[4])
			e.Schedule.Games = append(e.Schedule.Games, game)
		}
		return nil
	}
	return errors.New("csrf token 'csrf-token' not found in the HTML page")
}
