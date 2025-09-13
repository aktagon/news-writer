package main

import "time"

// Article represents the article output with full frontmatter
type Article struct {
	Title        string    `json:"title"`
	SourceURL    string    `json:"source_url"`
	SourceDomain string    `json:"source_domain"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
	Draft        bool      `json:"draft"`
	Categories   []string  `json:"categories"`
	Tags         []string  `json:"tags"`
	PlannerModel string    `json:"planner_model"`
	WriterModel  string    `json:"writer_model"`
	Deck         string    `json:"deck"`
}

// ProcessingStatus represents the outcome status of processing an article
type ProcessingStatus string

const (
	StatusSuccess ProcessingStatus = "success"
	StatusSkipped ProcessingStatus = "skipped"
	StatusError   ProcessingStatus = "error"
)

// ProcessingResult tracks the outcome of processing each URL
type ProcessingResult struct {
	URL      string
	Status   ProcessingStatus
	Filename string
	Error    error
}
