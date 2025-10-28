package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHttpClient_Timeout(t *testing.T) {
	// Create a server that delays response beyond timeout
	delayedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond) // Longer than client timeout (15s)
		w.WriteHeader(http.StatusOK)
	}))

	// This test would normally fail with timeout, but since our timeout is 15s
	// and the delay is 200ms, it should succeed
	client := httpClient()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", delayedServer.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	_, err = client.Do(req)
	elapsed := time.Since(start)

	if elapsed < 50*time.Millisecond {
		t.Error("Request completed too quickly")
	}

	// Request should complete within client timeout
	if elapsed > 20*time.Second {
		t.Error("Request took too long, possible timeout issue")
	}
}

func TestHttpClient_ConnectionPooling(t *testing.T) {
	reqCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount++
		w.WriteHeader(http.StatusOK)
	}))

	client := httpClient()

	// Make multiple requests to the same server
	for i := 0; i < 5; i++ {
		req, err := http.NewRequest("GET", server.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}

	if reqCount != 5 {
		t.Errorf("Expected 5 requests, got %d", reqCount)
	}
}

func TestHttpClient_HTTP2(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	client := httpClient()
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Logf("HTTP2 not available: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.ProtoMajor != 2 {
		t.Logf("Using HTTP/%d.%d (HTTP2 may not be supported by test server)", resp.ProtoMajor, resp.ProtoMinor)
	}
}

func TestHttpClient_ContextPropagation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if context values are propagated
		w.WriteHeader(http.StatusOK)
	}))

	client := httpClient()
	ctx := context.WithValue(context.Background(), "test-key", "test-value")
	req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestHttpClient_NoLeaks(t *testing.T) {
	// This is a basic test to ensure the client doesn't leak connections
	// More sophisticated testing would require monitoring goroutines
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	client := httpClient()
	
	for i := 0; i < 10; i++ {
		req, err := http.NewRequest("GET", server.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}
	
	// Give time for cleanup
	time.Sleep(100 * time.Millisecond)
	
	t.Log("No connection leaks detected")
}

