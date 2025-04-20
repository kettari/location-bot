package scraper

import (
	"io"
	"log/slog"
	"net/http"
)

type Page struct {
	URL     string
	Html    string
	Cookies []*http.Cookie
}

func NewPage(url string) *Page {
	return &Page{URL: url}
}

func (p *Page) LoadHtml() error {
	req, err := http.NewRequest("GET", p.URL, nil)
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
			slog.Error("failed to close response body", "url", p.URL, "err", err)
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
