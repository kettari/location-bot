package scraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetcher_FetchAll(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			// Root page with CSRF token
			w.Header().Set("Set-Cookie", "_csrf=test-csrf-cookie")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `<html><head><meta name="csrf-token" content="test-csrf-token"></head></html>`)
		case "/event/json-calendar":
			// Events JSON endpoint
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `[{"id":1,"title":"Test Event","url":"/event/1"}]`)
		case "/event/1":
			// Individual event page
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `<div class="event-single"><h4>Test Event</h4></div>`)
		default:
			t.Logf("unexpected request: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Adjust fetcher to use test server
	fetcher := &Fetcher{
		rootURL:   server.URL,
		eventsURL: server.URL + "/event/json-calendar?start=%s&end=%s",
	}

	// Mock fetchPages function
	fetchPages := func(urls []string) ([]Page, error) {
		var pages []Page
		for _, url := range urls {
			p := NewPage(url)
			if err := p.LoadHtml(); err != nil {
				return nil, err
			}
			pages = append(pages, *p)
		}
		return pages, nil
	}

	// Execute fetch
	result, err := fetcher.FetchAll(fetchPages)
	if err != nil {
		t.Fatalf("FetchAll() error = %v, want nil", err)
	}

	// Assertions
	if result == nil {
		t.Fatal("FetchAll() result is nil")
	}
	if len(result.Pages) != 1 {
		t.Errorf("FetchAll() pages count = %d, want 1", len(result.Pages))
	}
	if len(result.Events) != 1 {
		t.Errorf("FetchAll() events count = %d, want 1", len(result.Events))
	}
	if result.Events[0].Title != "Test Event" {
		t.Errorf("FetchAll() event title = %s, want 'Test Event'", result.Events[0].Title)
	}
}

func TestFetcher_FetchAll_ErrorCases(t *testing.T) {
	// Test server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	fetcher := &Fetcher{
		rootURL:   server.URL,
		eventsURL: server.URL + "/event/json-calendar?start=%s&end=%s",
	}

	fetchPages := func(urls []string) ([]Page, error) {
		return []Page{}, nil
	}

	_, err := fetcher.FetchAll(fetchPages)
	if err == nil {
		t.Error("FetchAll() expected error, got nil")
	}
}

func TestNewFetcher(t *testing.T) {
	fetcher := NewFetcher()
	if fetcher == nil {
		t.Fatal("NewFetcher() returned nil")
	}
	if fetcher.rootURL != rootURL {
		t.Errorf("NewFetcher() rootURL = %s, want %s", fetcher.rootURL, rootURL)
	}
	if fetcher.eventsURL != eventsURL {
		t.Errorf("NewFetcher() eventsURL = %s, want %s", fetcher.eventsURL, eventsURL)
	}
}

func TestFetcher_FetchAll_EmptyEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Set-Cookie", "_csrf=test")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `<html><head><meta name="csrf-token" content="token"></head></html>`)
		case "/event/json-calendar":
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `[]`)
		}
	}))
	defer server.Close()

	fetcher := &Fetcher{
		rootURL:   server.URL,
		eventsURL: server.URL + "/event/json-calendar?start=%s&end=%s",
	}

	fetchPages := func(urls []string) ([]Page, error) {
		return []Page{}, nil
	}

	result, err := fetcher.FetchAll(fetchPages)
	if err == nil {
		t.Fatal("FetchAll() expected error for empty events, got nil")
	}
	if result != nil {
		t.Error("FetchAll() result should be nil on error")
	}
}

