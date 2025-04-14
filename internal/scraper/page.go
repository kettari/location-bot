package scraper

import (
	"io"
	"log/slog"
	"net/http"
)

type Page struct {
	Html    string
	Cookies []*http.Cookie
}

func (p *Page) LoadHtml(url string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

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

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	p.Html = string(data)
	p.Cookies = resp.Cookies()

	return nil
}
