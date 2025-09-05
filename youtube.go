// youtube.go
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// HTTPError represents an HTTP error with status code
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return e.Message
}

// GetTranscript fetches a YouTube transcript, using a local cache if available.
func GetTranscript(videoURL string, settings YouTubeSettings) (string, error) {
	videoID, err := extractVideoID(videoURL)
	if err != nil {
		return "", fmt.Errorf("could not extract video ID: %w", err)
	}

	cachePath := filepath.Join(".cache", "youtube", videoID)
	if content, err := os.ReadFile(cachePath); err == nil {
		return string(content), nil
	}

	transcript, err := fetchTranscriptWithRetries(videoID, settings)
	if err != nil {
		return "", fmt.Errorf("failed to fetch transcript: %w", err)
	}

	// Create cache directory if it doesn't exist
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	if err := os.WriteFile(cachePath, []byte(transcript), 0644); err != nil {
		return "", fmt.Errorf("failed to cache transcript: %w", err)
	}

	return transcript, nil
}

func extractVideoID(videoURL string) (string, error) {
	parsedURL, err := url.Parse(videoURL)
	if err != nil {
		return "", err
	}
	return parsedURL.Query().Get("v"), nil
}

func fetchTranscriptWithRetries(videoID string, settings YouTubeSettings) (string, error) {
	var lastErr error
	for i := 0; i < settings.Retries; i++ {
		transcript, err := fetchTranscript(videoID, settings)
		if err == nil {
			return transcript, nil
		}
		lastErr = err
		if httpErr, ok := err.(*HTTPError); ok && httpErr.StatusCode == http.StatusTooManyRequests {
			time.Sleep(time.Second * time.Duration(i+1)) // Exponential backoff
			continue
		}
		return "", err
	}
	return "", fmt.Errorf("exceeded max retries: %w", lastErr)
}

func fetchTranscript(videoID string, settings YouTubeSettings) (string, error) {
	videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)

	req, err := http.NewRequest("GET", settings.TranscriptAPIURL, nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	q.Add("url", videoURL)
	q.Add("api_key", settings.TranscriptAPIKey)
	q.Add("text", "true")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", &HTTPError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("bad status code: %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
