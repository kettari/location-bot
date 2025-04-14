package scraper

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
)

type Events struct {
	Html string
}

func (e *Events) LoadEvents(csrf *Csrf, url string) error {
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	cookie := http.Cookie{
		Name:     "_csrf",
		Value:    csrf.Cookie,
		Path:     "/",
		HttpOnly: true,
	}
	req.Header.Set("Cookie", cookie.String())
	req.Header.Set("x-csrf-token", csrf.Token)
	req.Header.Set("x-requested-with", "XMLHttpRequest")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			slog.Error("Failed to close response body", "url", url, "err", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		slog.Error("Failed to load events HTML page", "url", url, "http_code", resp.StatusCode, "status", resp.Status)
		return errors.New("failed to load events HTML page")
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	e.Html = string(data)

	return nil
}
