package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

type Events struct {
	URL    string
	Csrf   *Csrf
	JSON   string
	Events []RoleconEvent
}

type RoleconEvent struct {
	ID        int      `json:"id"`
	Title     string   `json:"title"`
	URL       string   `json:"url"`
	Start     string   `json:"start"` // ISO date from calendar
	End       string   `json:"end"`   // ISO date from calendar
	AllDay    bool     `json:"allDay"`
	ClassName []string `json:"className"` // CSS class for event type (array in API)
}

func NewEvents(url string, csrf *Csrf) *Events {
	return &Events{URL: url, Csrf: csrf}
}

// LoadEvents from the Rolecon website
func (e *Events) LoadEvents() error {
	req, err := http.NewRequest("GET", e.URL, nil)
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

	resp, err := httpClient().Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			slog.Error("failed to close response body", "url", e.URL, "err", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to load events HTML page %s with HTTP code %d %s", e.URL, resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	e.JSON = string(data)

	return nil
}

func (e *Events) UnmarshalEvents() error {
	if err := json.Unmarshal([]byte(e.JSON), &e.Events); err != nil {
		return err
	}
	if len(e.Events) == 0 {
		return fmt.Errorf("no events found after unmarshal")
	}

	// Log events for debugging
	for _, event := range e.Events {
		slog.Debug("unmarshaled event",
			"id", event.ID,
			"title", event.Title,
			"url", event.URL,
			"start", event.Start,
			"end", event.End)
	}

	return nil
}
