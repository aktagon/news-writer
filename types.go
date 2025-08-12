// types.go
package main

import "time"

// ArticleItem represents a single article to process
type ArticleItem struct {
	ID            int    `yaml:"id"`
	SourceURL     string `yaml:"source_url"`
	DiscussionURL string `yaml:"discussion_url,omitempty"`
}

// Config represents the YAML configuration structure
type Config struct {
	Items []ArticleItem `yaml:"items"`
}

// Plan represents the planner agent output
type Plan struct {
	Title     string   `json:"title"`
	KeyPoints []string `json:"key_points"`
	Structure []string `json:"structure"`
	Target    Target   `json:"target"`
}

// Target represents the target specifications for the article
type Target struct {
	WordCount int    `json:"word_count"`
	Tone      string `json:"tone"`
}

// Article represents the final article output
type Article struct {
	Title     string    `json:"title"`
	Source    string    `json:"source"`
	SourceURL string    `json:"source_url"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// ProcessingResult tracks the outcome of processing each item
type ProcessingResult struct {
	ID       int
	Success  bool
	Filename string
	Error    error
}
