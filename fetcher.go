package main

import (
	"fmt"
	"net/http"

	md "github.com/JohannesKaufmann/html-to-markdown"
)

// ContentResult represents the result of fetching content
type ContentResult struct {
	Text   string // Markdown text content (for HTML pages)
	FileID string // File ID (for PDFs)
}

// ContentFetcher handles fetching and processing content from URLs
type ContentFetcher struct {
	handlers []ContentHandler
	client   *http.Client
}

// NewContentFetcher creates a new content fetcher with default handlers
func NewContentFetcher(apiKey string) *ContentFetcher {
	f := &ContentFetcher{
		client: &http.Client{},
	}

	// Register handlers (most specific first)
	f.AddHandler(&YouTubeHandler{})
	f.AddHandler(&PDFHandler{apiKey: apiKey})
	f.AddHandler(&HTMLHandler{converter: md.NewConverter("", true, nil)}) // fallback

	return f
}

// AddHandler adds a content handler to the chain
func (f *ContentFetcher) AddHandler(handler ContentHandler) {
	f.handlers = append(f.handlers, handler)
}

// FetchContent fetches and processes content using handler chain
func (f *ContentFetcher) FetchContent(url string) (*ContentResult, error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{StatusCode: resp.StatusCode, URL: url}
	}

	// Find handler based on URL + response headers
	for _, handler := range f.handlers {
		if handler.CanHandle(url, resp) {
			return handler.Handle(url, resp)
		}
	}

	return nil, fmt.Errorf("no handler found for %s", url)
}
