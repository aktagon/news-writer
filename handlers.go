package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/aktagon/llmkit/anthropic"
)

// HTTPError represents an HTTP error with status code
type HTTPError struct {
	StatusCode int
	URL        string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d for %s", e.StatusCode, e.URL)
}

// ContentHandler processes URLs based on response inspection
type ContentHandler interface {
	CanHandle(url string, resp *http.Response) bool
	Handle(url string, resp *http.Response) (*ContentResult, error)
}

// Global rate limiter for YouTube API calls
var (
	youtubeMutex     sync.Mutex
	lastYouTubeCall  time.Time
	youtubeCallDelay = 2 * time.Second // Minimum delay between API calls
	debugEnabled     bool
)

// SetDebugMode enables or disables debug logging
func SetDebugMode(enabled bool) {
	debugEnabled = enabled
}

func debugLog(format string, args ...interface{}) {
	if debugEnabled {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// YouTubeHandler handles YouTube videos
type YouTubeHandler struct{}

func (h *YouTubeHandler) CanHandle(url string, resp *http.Response) bool {
	return strings.Contains(url, "youtube.com/watch") ||
		strings.Contains(url, "youtu.be/")
}

func (h *YouTubeHandler) Handle(url string, resp *http.Response) (*ContentResult, error) {
	// Load settings from environment
	apiKey := os.Getenv("YOUTUBE_TRANSCRIPT_API_KEY")
	apiURL := os.Getenv("YOUTUBE_TRANSCRIPT_API_URL")

	if apiKey == "" || apiURL == "" {
		return nil, fmt.Errorf("YouTube API configuration missing: set YOUTUBE_TRANSCRIPT_API_KEY and YOUTUBE_TRANSCRIPT_API_URL")
	}

	transcript, err := getTranscript(url, apiKey, apiURL)
	if err != nil {
		return nil, fmt.Errorf("fetching YouTube transcript: %w", err)
	}

	return &ContentResult{Text: transcript}, nil
}

// PDFHandler handles PDF content
type PDFHandler struct {
	apiKey string
}

func (h *PDFHandler) CanHandle(url string, resp *http.Response) bool {
	// Check URL extension first
	if strings.HasSuffix(strings.ToLower(url), ".pdf") {
		return true
	}

	// Check content-type header
	contentType := resp.Header.Get("Content-Type")
	return strings.Contains(contentType, "application/pdf")
}

func (h *PDFHandler) Handle(url string, resp *http.Response) (*ContentResult, error) {
	// Download PDF content to a temporary file
	tempFile, err := os.CreateTemp("", "pdf-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("creating temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up temp file
	defer tempFile.Close()

	// Copy PDF content from response to temp file
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("downloading PDF content: %w", err)
	}

	// Close the file so it can be opened by UploadFile
	tempFile.Close()

	// Upload PDF file to Anthropic for processing
	file, err := anthropic.UploadFile(tempFile.Name(), h.apiKey)
	if err != nil {
		return nil, fmt.Errorf("uploading PDF file: %w", err)
	}

	return &ContentResult{FileID: file.ID}, nil
}

// HTMLHandler handles regular HTML content (fallback)
type HTMLHandler struct {
	converter *md.Converter
}

func (h *HTMLHandler) CanHandle(url string, resp *http.Response) bool {
	return true // Always handles as fallback
}

func (h *HTMLHandler) Handle(url string, resp *http.Response) (*ContentResult, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	markdown, err := h.converter.ConvertString(string(body))
	if err != nil {
		return nil, fmt.Errorf("converting HTML to markdown: %w", err)
	}

	return &ContentResult{Text: markdown}, nil
}

// YouTube transcript functions

func getTranscript(videoURL, apiKey, apiURL string) (string, error) {
	videoID, err := extractVideoID(videoURL)
	if err != nil {
		return "", fmt.Errorf("extracting video ID: %w", err)
	}

	// Check cache
	cachePath := filepath.Join(".cache", "youtube", videoID)
	if content, err := os.ReadFile(cachePath); err == nil {
		return string(content), nil
	}

	// Fetch with retries (increased from 3 to 5 for rate limit handling)
	transcript, err := fetchTranscriptWithRetries(videoID, apiKey, apiURL, 5)
	if err != nil {
		return "", err
	}

	// Cache result
	cacheDir := filepath.Dir(cachePath)
	os.MkdirAll(cacheDir, 0755)
	os.WriteFile(cachePath, []byte(transcript), 0644)

	return transcript, nil
}

func extractVideoID(videoURL string) (string, error) {
	parsedURL, err := url.Parse(videoURL)
	if err != nil {
		return "", err
	}

	// Validate YouTube domain
	if !strings.Contains(parsedURL.Host, "youtube.com") && !strings.Contains(parsedURL.Host, "youtu.be") {
		return "", fmt.Errorf("not a YouTube URL")
	}

	// Handle youtu.be URLs
	if strings.Contains(parsedURL.Host, "youtu.be") {
		return strings.TrimPrefix(parsedURL.Path, "/"), nil
	}

	// Handle youtube.com URLs
	videoID := parsedURL.Query().Get("v")
	if videoID == "" {
		return "", fmt.Errorf("no video ID found in URL")
	}
	return videoID, nil
}

func fetchTranscriptWithRetries(videoID, apiKey, apiURL string, retries int) (string, error) {
	var lastErr error
	for i := 0; i < retries; i++ {
		transcript, err := fetchTranscript(videoID, apiKey, apiURL)
		if err == nil {
			return transcript, nil
		}
		lastErr = err

		// Check for rate limit errors (either HTTP 429 or API service 429)
		isRateLimit := false
		if httpErr, ok := err.(*HTTPError); ok && httpErr.StatusCode == http.StatusTooManyRequests {
			isRateLimit = true
		}
		// Also check for 429 errors reported by the transcript service
		if strings.Contains(err.Error(), "too many 429 error responses") ||
			strings.Contains(err.Error(), "429") {
			isRateLimit = true
		}

		if isRateLimit && i < retries-1 {
			// Exponential backoff with jitter: 2^i + random(0-1) seconds
			backoff := time.Duration(1<<uint(i)) * time.Second
			jitter := time.Duration(float64(time.Second) * 0.5 * (1.0 + float64(i)))
			time.Sleep(backoff + jitter)
			continue
		}

		// For non-rate-limit errors, don't retry
		if !isRateLimit {
			return "", err
		}
	}
	return "", fmt.Errorf("exceeded max retries after %d attempts: %w", retries, lastErr)
}

func fetchTranscript(videoID, apiKey, apiURL string) (string, error) {
	// Rate limit YouTube API calls
	youtubeMutex.Lock()
	timeSinceLastCall := time.Since(lastYouTubeCall)
	if timeSinceLastCall < youtubeCallDelay {
		time.Sleep(youtubeCallDelay - timeSinceLastCall)
	}
	lastYouTubeCall = time.Now()
	youtubeMutex.Unlock()

	videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	q.Add("url", videoURL)
	q.Add("api_key", apiKey)
	q.Add("text", "true")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{
		Timeout: 30 * time.Second, // Add timeout to prevent hanging
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Debug logging for response
	debugLog("YouTube transcript API response: status=%d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return "", &HTTPError{StatusCode: resp.StatusCode, URL: videoURL}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Debug logging for first 100 chars of body
	bodyStr := string(body)
	preview := bodyStr
	if len(preview) > 100 {
		preview = preview[:100]
	}
	debugLog("YouTube transcript API body (first 100 chars): %q", preview)

	return bodyStr, nil
}
