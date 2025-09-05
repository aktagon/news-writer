// fetcher.go
package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ContentFetcher handles web content retrieval
type ContentFetcher struct {
	client          *http.Client
	youtubeSettings YouTubeSettings
}

// NewContentFetcher creates a new content fetcher
func NewContentFetcher(youtubeSettings YouTubeSettings) *ContentFetcher {
	return &ContentFetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		youtubeSettings: youtubeSettings,
	}
}

// FetchContent retrieves and processes content from a URL
func (cf *ContentFetcher) FetchContent(url string) (string, error) {
	if strings.Contains(url, "youtube.com") {
		if cf.youtubeSettings.TranscriptAPIKey == "" || cf.youtubeSettings.TranscriptAPIURL == "" {
			return "", fmt.Errorf("YouTube transcript settings missing. Please set the following environment variables:\n\nYOUTUBE_TRANSCRIPT_API_KEY=\"api-key-12345\"\nYOUTUBE_TRANSCRIPT_API_URL=\"https://us-central1-aktagon.cloudfunctions.net/get_youtube_transcript\"")
		}
		return GetTranscript(url, cf.youtubeSettings)
	}

	// Fetch raw HTML
	html, err := cf.fetchHTML(url)
	if err != nil {
		return "", fmt.Errorf("fetching HTML: %w", err)
	}

	// Convert to markdown-like text
	text := cf.htmlToText(html)

	// Clean and truncate if needed
	cleaned := cf.cleanText(text)

	return cleaned, nil
}

// fetchHTML retrieves the HTML content from a URL
func (cf *ContentFetcher) fetchHTML(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Set user agent to avoid blocks
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ArticleDistiller/1.0)")

	resp, err := cf.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// htmlToText converts HTML to readable text
func (cf *ContentFetcher) htmlToText(html string) string {
	// Remove script and style tags
	html = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`(?i)<style[^>]*>.*?</style>`).ReplaceAllString(html, "")

	// Convert headers to markdown-style
	html = regexp.MustCompile(`(?i)<h1[^>]*>(.*?)</h1>`).ReplaceAllString(html, "\n# $1\n")
	html = regexp.MustCompile(`(?i)<h2[^>]*>(.*?)</h2>`).ReplaceAllString(html, "\n## $1\n")
	html = regexp.MustCompile(`(?i)<h3[^>]*>(.*?)</h3>`).ReplaceAllString(html, "\n### $1\n")

	// Convert paragraphs
	html = regexp.MustCompile(`(?i)<p[^>]*>(.*?)</p>`).ReplaceAllString(html, "\n$1\n")

	// Convert line breaks
	html = regexp.MustCompile(`(?i)<br[^>]*/?>`).ReplaceAllString(html, "\n")

	// Convert lists
	html = regexp.MustCompile(`(?i)<li[^>]*>(.*?)</li>`).ReplaceAllString(html, "- $1\n")
	html = regexp.MustCompile(`(?i)<ul[^>]*>|</ul>`).ReplaceAllString(html, "\n")
	html = regexp.MustCompile(`(?i)<ol[^>]*>|</ol>`).ReplaceAllString(html, "\n")

	// Convert emphasis
	html = regexp.MustCompile(`(?i)<strong[^>]*>(.*?)</strong>`).ReplaceAllString(html, "**$1**")
	html = regexp.MustCompile(`(?i)<em[^>]*>(.*?)</em>`).ReplaceAllString(html, "*$1*")
	html = regexp.MustCompile(`(?i)<b[^>]*>(.*?)</b>`).ReplaceAllString(html, "**$1**")
	html = regexp.MustCompile(`(?i)<i[^>]*>(.*?)</i>`).ReplaceAllString(html, "*$1*")

	// Convert links
	html = regexp.MustCompile(`(?i)<a[^>]*href="([^"]*)"[^>]*>(.*?)</a>`).ReplaceAllString(html, "[$2]($1)")

	// Remove all remaining HTML tags
	html = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(html, "")

	// Decode HTML entities
	html = strings.ReplaceAll(html, "&nbsp;", " ")
	html = strings.ReplaceAll(html, "&amp;", "&")
	html = strings.ReplaceAll(html, "&lt;", "<")
	html = strings.ReplaceAll(html, "&gt;", ">")
	html = strings.ReplaceAll(html, "&quot;", "\"")
	html = strings.ReplaceAll(html, "&#39;", "'")

	return html
}

// cleanText cleans and normalizes the extracted text
func (cf *ContentFetcher) cleanText(text string) string {
	// Normalize whitespace
	text = regexp.MustCompile(`\r\n|\r`).ReplaceAllString(text, "\n")
	text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`\n\s*\n\s*\n`).ReplaceAllString(text, "\n\n")

	// Remove leading/trailing whitespace from lines
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	text = strings.Join(lines, "\n")

	// Remove empty lines at start/end
	text = strings.TrimSpace(text)

	// Truncate if too long (keep under ~10k chars for API limits)
	if len(text) > 10000 {
		text = text[:10000] + "\n\n[Content truncated for processing...]"
	}

	return text
}
