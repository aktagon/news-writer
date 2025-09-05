// types.go
package main

import "time"

// ArticleItem represents a single article to process
type ArticleItem struct {
	URL string `yaml:"url"`
}

// Config represents the YAML configuration structure
type Config struct {
	Items []ArticleItem `yaml:"items"`
}

// Plan represents the planner agent output
type Plan struct {
	Title      string   `json:"title"`
	Deck       string   `json:"deck"`
	KeyPoints  []string `json:"key_points"`
	Structure  []string `json:"structure"`
	Categories []string `json:"categories"`
	Tags       []string `json:"tags"`
	Target     Target   `json:"target"`
}

// Target represents the target specifications for the article
type Target struct {
	WordCount int    `json:"word_count"`
	Tone      string `json:"tone"`
	Audience  string `json:"audience"`
}

// Article represents the final article output
type Article struct {
	Title          string    `json:"title"`
	Source         string    `json:"source"`
	SourceURL      string    `json:"source_url"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
	Deck           string    `json:"deck"`
	Categories     []string  `json:"categories"`
	Tags           []string  `json:"tags"`
	Author         string    `json:"author"`
	AuthorTitle    string    `json:"author_title"`
	SourceDomain   string    `json:"source_domain"`
	TargetAudience string    `json:"target_audience"`
}

// ProcessingStatus represents the outcome status of processing an article
type ProcessingStatus string

const (
	StatusSuccess ProcessingStatus = "success"
	StatusSkipped ProcessingStatus = "skipped"
	StatusError   ProcessingStatus = "error"
)

// ProcessingResult tracks the outcome of processing each item
type ProcessingResult struct {
	URL      string
	Status   ProcessingStatus
	Filename string
	Error    error
}

// AgentSettings represents settings for a single agent
type AgentSettings struct {
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
}

// Settings represents the application settings
type Settings struct {
	OutputDirectory string   `yaml:"output_directory"`
	TemplatePath    string   `yaml:"template_path"`
	Categories      []string `yaml:"categories"`
	Agents          struct {
		Planner AgentSettings `yaml:"planner"`
		Writer  AgentSettings `yaml:"writer"`
	} `yaml:"agents"`
	YouTube YouTubeSettings `yaml:"youtube"`
}

// YouTubeSettings represents the YouTube-specific settings
type YouTubeSettings struct {
	TranscriptAPIKey string // Read from YOUTUBE_TRANSCRIPT_API_KEY env var
	TranscriptAPIURL string // Read from YOUTUBE_TRANSCRIPT_API_URL env var
	Retries          int    `yaml:"retries"`
}
