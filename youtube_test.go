// youtube_test.go
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchTranscript(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request parameters
		if r.URL.Query().Get("url") != "https://www.youtube.com/watch?v=test123" {
			t.Errorf("Expected url parameter to be 'https://www.youtube.com/watch?v=test123', got '%s'", r.URL.Query().Get("url"))
		}
		if r.URL.Query().Get("api_key") != "test-key" {
			t.Errorf("Expected api_key parameter to be 'test-key', got '%s'", r.URL.Query().Get("api_key"))
		}
		if r.URL.Query().Get("text") != "true" {
			t.Errorf("Expected text parameter to be 'true', got '%s'", r.URL.Query().Get("text"))
		}

		// Return mock transcript
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("This is a test transcript"))
	}))
	defer server.Close()

	// Test settings
	settings := YouTubeSettings{
		TranscriptAPIKey: "test-key",
		TranscriptAPIURL: server.URL,
		Retries:          1,
	}

	// Test successful fetch
	result, err := fetchTranscript("test123", settings)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result != "This is a test transcript" {
		t.Errorf("Expected 'This is a test transcript', got '%s'", result)
	}
}

func TestFetchTranscriptHTTPError(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	settings := YouTubeSettings{
		TranscriptAPIKey: "test-key",
		TranscriptAPIURL: server.URL,
		Retries:          1,
	}

	_, err := fetchTranscript("test123", settings)
	if err == nil {
		t.Fatal("Expected an error, got nil")
	}

	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("Expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code 500, got %d", httpErr.StatusCode)
	}
}

func TestGetTranscriptWithCache(t *testing.T) {
	// Create a .cache/youtube directory in current working directory for this test
	cacheDir := ".cache/youtube"
	os.MkdirAll(cacheDir, 0755)
	defer os.RemoveAll(".cache") // Clean up after test

	// Create a cached transcript
	cachePath := filepath.Join(cacheDir, "test123")
	cachedContent := "Cached transcript content"
	err := os.WriteFile(cachePath, []byte(cachedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create cache file: %v", err)
	}

	// Test with mock server (should not be called due to cache)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Server should not be called when cache exists")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Should not reach here"))
	}))
	defer server.Close()

	settings := YouTubeSettings{
		TranscriptAPIKey: "test-key",
		TranscriptAPIURL: server.URL,
		Retries:          1,
	}

	// Test GetTranscript with cache - should return cached content
	result, err := GetTranscript("https://www.youtube.com/watch?v=test123", settings)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result != cachedContent {
		t.Errorf("Expected cached content '%s', got '%s'", cachedContent, result)
	}
}

func TestExtractVideoID(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"https://www.youtube.com/watch?v=test123&t=10s", "test123"},
		{"https://youtube.com/watch?v=abc123", "abc123"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("URL_%s", test.expected), func(t *testing.T) {
			result, err := extractVideoID(test.url)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}
}
