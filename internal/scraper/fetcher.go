package scraper

import (
	"fmt"
	"log/slog"
	"time"
)

const (
	rootURL   = "https://rolecon.ru"
	eventsURL = "https://rolecon.ru/event/json-calendar?start=%s&end=%s"
	twoWeeks  = 24 * time.Hour * 14
)

// Fetcher encapsulates the logic for fetching CSRF, loading events JSON, and collecting individual event pages.
type Fetcher struct {
	rootURL   string
	eventsURL string
}

// FetchResult contains all fetched pages and metadata.
type FetchResult struct {
	Pages    []Page
	Events   []RoleconEvent
	EventMap map[string]RoleconEvent // Maps URL to event metadata
	TotalURL int
}

// NewFetcher creates a new fetcher with default URLs.
func NewFetcher() *Fetcher {
	return &Fetcher{
		rootURL:   rootURL,
		eventsURL: eventsURL,
	}
}

// FetchAll performs the full fetch workflow: CSRF extraction, events JSON loading, and individual pages collection.
func (f *Fetcher) FetchAll(fetchPages func([]string) ([]Page, error)) (*FetchResult, error) {
	// Get the root page for CSRF
	slog.Debug("requesting page", "url", f.rootURL)
	page := NewPage(f.rootURL)
	if err := page.LoadHtml(); err != nil {
		return nil, fmt.Errorf("failed to load root page: %w", err)
	}
	slog.Debug("initial page loaded", "size", len(page.Html), "cookies_count", len(page.Cookies))

	// Extract CSRF token and cookie
	csrf := NewCsrf(page)
	if err := csrf.ExtractCsrfToken(); err != nil {
		return nil, fmt.Errorf("failed to extract CSRF token: %w", err)
	}
	slog.Debug("found CSRF token", "token", csrf.Token)
	if err := csrf.ExtractCsrfCookie(); err != nil {
		return nil, fmt.Errorf("failed to extract CSRF cookie: %w", err)
	}
	slog.Debug("found CSRF cookie", "cookie", csrf.Cookie)

	// Load events JSON
	url := fmt.Sprintf(f.eventsURL, time.Now().Format("2006-01-02"), time.Now().Add(twoWeeks).Format("2006-01-02"))
	slog.Debug("requesting events", "url", url)
	events := NewEvents(url, csrf)
	if err := events.LoadEvents(); err != nil {
		return nil, fmt.Errorf("failed to load events: %w", err)
	}
	slog.Debug("events page loaded", "size", len(events.JSON))
	if err := events.UnmarshalEvents(); err != nil {
		return nil, fmt.Errorf("failed to unmarshal events: %w", err)
	}
	slog.Debug("events unmarshalled", "events_count", len(events.Events))

	// Collect URLs for individual event pages and build URL->event mapping
	var urls []string
	eventMap := make(map[string]RoleconEvent)
	for _, event := range events.Events {
		fullURL := f.rootURL + event.URL
		urls = append(urls, fullURL)
		eventMap[fullURL] = event
	}

	// Fetch individual pages using the provided function
	slog.Debug("requesting events pages")
	pages, err := fetchPages(urls)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch event pages: %w", err)
	}
	slog.Debug("collected events pages", "pages_count", len(pages))

	return &FetchResult{
		Pages:    pages,
		Events:   events.Events,
		EventMap: eventMap,
		TotalURL: len(urls),
	}, nil
}
