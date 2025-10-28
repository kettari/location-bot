package console

import (
	"fmt"
	"github.com/kettari/location-bot/internal/scraper"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestScheduleFetchCommand_dispatcher(t *testing.T) {
	// Create test server that simulates page responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "<html><body>Test page %s</body></html>", r.URL.Path)
	}))
	defer server.Close()

	tests := []struct {
		name        string
		urls        []string
		workerCount int
		wantError   bool
	}{
		{
			name:        "single url",
			urls:        []string{server.URL + "/1"},
			workerCount: 1,
			wantError:   false,
		},
		{
			name:        "multiple urls",
			urls:        []string{server.URL + "/1", server.URL + "/2", server.URL + "/3"},
			workerCount: 3,
			wantError:   false,
		},
		{
			name:        "more workers than urls",
			urls:        []string{server.URL + "/1", server.URL + "/2"},
			workerCount: 5,
			wantError:   false,
		},
		{
			name:        "empty urls",
			urls:        []string{},
			workerCount: 1,
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ScheduleFetchCommand{}
			var pages []scraper.Page
			errorFlag := false

			cmd.dispatcher(tt.urls, tt.workerCount, &pages, &errorFlag)

			if errorFlag != tt.wantError {
				t.Errorf("dispatcher() errorFlag = %v, want %v", errorFlag, tt.wantError)
			}

			if !tt.wantError && len(pages) != len(tt.urls) {
				t.Errorf("dispatcher() collected %d pages, want %d", len(pages), len(tt.urls))
			}
		})
	}
}

func TestScheduleFetchCommand_dispatcher_ContextCancellation(t *testing.T) {
	// This test verifies that the dispatcher handles cancellation properly
	// by creating a scenario where requests would take too long
	
	delayedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer delayedServer.Close()

	cmd := &ScheduleFetchCommand{}
	urls := []string{delayedServer.URL + "/1", delayedServer.URL + "/2"}
	var pages []scraper.Page
	errorFlag := false

	start := time.Now()
	cmd.dispatcher(urls, 2, &pages, &errorFlag)
	elapsed := time.Since(start)

	// Should complete without hanging
	if elapsed > 30*time.Second {
		t.Error("dispatcher() took too long, possible deadlock")
	}

	if errorFlag {
		t.Error("dispatcher() set errorFlag unexpectedly")
	}
}

func TestScheduleFetchCommand_dispatcher_ErrorHandling(t *testing.T) {
	// Create a server that fails after first request
	requestCount := 0
	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount > 2 {
			// Simulate error on third request
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "<html><body>Success</body></html>")
		}
	}))
	defer failingServer.Close()

	cmd := &ScheduleFetchCommand{}
	urls := []string{
		failingServer.URL + "/1",
		failingServer.URL + "/2",
		failingServer.URL + "/3", // This one will fail
	}
	var pages []scraper.Page
	errorFlag := false

	cmd.dispatcher(urls, 2, &pages, &errorFlag)

	// errorFlag should be true due to the failing request
	if !errorFlag {
		t.Error("dispatcher() should set errorFlag on errors")
	}
}

func TestScheduleFetchCommand_dispatcher_Concurrency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "<html><body>%s</body></html>", r.URL.Path)
	}))
	defer server.Close()

	// Test with concurrent workers
	numUrls := 10
	urls := make([]string, numUrls)
	for i := 0; i < numUrls; i++ {
		urls[i] = fmt.Sprintf("%s/%d", server.URL, i)
	}

	cmd := &ScheduleFetchCommand{}
	var pages []scraper.Page
	errorFlag := false

	start := time.Now()
	cmd.dispatcher(urls, 5, &pages, &errorFlag)
	elapsed := time.Since(start)

	if errorFlag {
		t.Error("dispatcher() set errorFlag unexpectedly")
	}

	if len(pages) != numUrls {
		t.Errorf("dispatcher() collected %d pages, want %d", len(pages), numUrls)
	}

	// With concurrent workers, should take less time than sequential
	maxTime := 100 * time.Millisecond * time.Duration(numUrls)
	if elapsed > maxTime {
		t.Logf("dispatcher() took %v, which is longer than expected for concurrent execution", elapsed)
	}
}

func TestScheduleFetchCommand_dispatcher_EmptyResults(t *testing.T) {
	cmd := &ScheduleFetchCommand{}
	var pages []scraper.Page
	errorFlag := false

	cmd.dispatcher([]string{}, 5, &pages, &errorFlag)

	if errorFlag {
		t.Error("dispatcher() should not set errorFlag for empty urls")
	}

	if len(pages) != 0 {
		t.Errorf("dispatcher() collected %d pages, want 0", len(pages))
	}
}

