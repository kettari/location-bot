package scraper

import (
	"errors"
	"github.com/kettari/location-bot/internal/notifier"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
)

type Events struct {
	URL      string
	Csrf     *Csrf
	Html     string
	Parts    []string
	Schedule *notifier.Schedule
}

func NewEvents(url string, csrf *Csrf) *Events {
	return &Events{URL: url, Csrf: csrf}
}

// LoadEvents from the Rolecon website
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

// BreakDown long HTML to well-formed HTML chunks for each event
func (e *Events) BreakDown() error {
	r := regexp.MustCompile(`<div class="event-single[^-][^>]+">`)
	matches := r.FindAllStringIndex(e.Html, -1)
	previousIndex := 0
	for _, match := range matches {
		chunk := e.Html[previousIndex:match[0]]
		if len(strings.Trim(chunk, " \n\r\t")) > 0 {
			e.Parts = append(e.Parts, chunk)
		}
		previousIndex = match[0]
	}
	return nil
}

// Rejoin builds array of well-formed HTML with events where each part is shorter
// than maximum OpenAI API allowed request length
func (e *Events) Rejoin() (result []string) {
	/*buf := ""
	for _, part := range e.Parts {
		if len(buf+part) < chatgpt.MaxInputLength {
			buf += part
		} else {
			result = append(result, buf)
			buf = ""
		}
	}
	if len(buf) > 0 {
		result = append(result, buf)
	}*/
	var buf []string
	for _, part := range e.Parts {
		if len(buf) < 2 {
			buf = append(buf, part)
		} else {
			result = append(result, buf...)
			buf = []string{}
		}
	}
	if len(buf) > 0 {
		result = append(result, buf...)
	}
	return result
}
