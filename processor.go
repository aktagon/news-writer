package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

// ArticleProcessor handles the main workflow
type ArticleProcessor struct {
	agents  *AgentManager
	fetcher *ContentFetcher
	config  *Config
	apiKey  string
}

// NewArticleProcessor creates a new processor with agent manager and config
func NewArticleProcessor(apiKey string, overrides *ConfigOverrides) (*ArticleProcessor, error) {
	config, err := NewConfig(overrides)
	if err != nil {
		return nil, fmt.Errorf("creating config: %w", err)
	}

	agents, err := NewAgentManager(apiKey, config)
	if err != nil {
		return nil, fmt.Errorf("creating agent manager: %w", err)
	}

	fetcher := NewContentFetcher(apiKey)

	return &ArticleProcessor{
		agents:  agents,
		fetcher: fetcher,
		config:  config,
		apiKey:  apiKey,
	}, nil
}

// ProcessURLsFromFile processes all URLs from a config file
func (p *ArticleProcessor) ProcessURLsFromFile(configPath string) error {
	urls, err := p.loadURLsFromFile(configPath)
	if err != nil {
		return fmt.Errorf("loading URLs: %w", err)
	}

	log.Printf("Processing %d URLs from %s", len(urls), configPath)

	successful := 0
	failed := 0
	skipped := 0

	for _, url := range urls {
		filename, err := p.ProcessURL(url, false)
		if err != nil {
			log.Printf("✗ Failed: %s - %v", url, err)
			failed++
		} else {
			log.Printf("✓ %s -> %s", url, filename)
			successful++
		}
	}

	log.Printf("Complete: %d successful, %d failed, %d skipped", successful, failed, skipped)
	return nil
}

// ProcessURL processes a single URL
func (p *ArticleProcessor) ProcessURL(url string, rewrite bool) (string, error) {
	// Check if article already exists
	existingFile := p.findExistingFile(url)
	if existingFile != "" && !rewrite {
		log.Printf("→ Skipping existing: %s", existingFile)
		return existingFile, nil
	}

	// Fetch content
	content, err := p.fetcher.FetchContent(url)
	if err != nil {
		return "", fmt.Errorf("fetching content: %w", err)
	}

	// Generate metadata using planner agent
	metadata, err := p.agents.PlanMetadata(url, content)
	if err != nil {
		return "", fmt.Errorf("generating metadata: %w", err)
	}

	// Generate article with single AI call
	article, err := p.generateArticle(url, content, metadata)
	if err != nil {
		return "", fmt.Errorf("generating article: %w", err)
	}

	// Generate filename
	filename := existingFile
	if filename == "" {
		filename = p.generateFilename(url, article.Title)
	}

	// Save article
	err = p.saveArticle(filename, article)
	if err != nil {
		return "", fmt.Errorf("saving article: %w", err)
	}

	log.Printf("✓ Saved: %s", filename)
	return filename, nil
}

// loadURLsFromFile loads URLs from YAML file
func (p *ArticleProcessor) loadURLsFromFile(configPath string) ([]string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	type Source struct {
		URL string `yaml:"url"`
	}
	type Config struct {
		Sources []Source `yaml:"sources"`
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	var urls []string
	for _, source := range config.Sources {
		if source.URL != "" && (strings.HasPrefix(source.URL, "http://") || strings.HasPrefix(source.URL, "https://")) {
			urls = append(urls, source.URL)
		}
	}

	return urls, nil
}

// generateArticle creates an article using the AgentManager
func (p *ArticleProcessor) generateArticle(url string, content *ContentResult, metadata *FrontmatterMetadata) (*Article, error) {
	// Use AgentManager to write the article with configured prompts
	articleContent, err := p.agents.Write(content, metadata)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	// Get model info from agents
	plannerModel, writerModel := p.agents.GetModelInfo()

	// Extract domain from URL
	sourceDomain := p.extractDomain(url)

	return &Article{
		Title:        metadata.Title,
		SourceURL:    url,
		SourceDomain: sourceDomain,
		Content:      articleContent,
		CreatedAt:    time.Now(),
		Draft:        false,
		Categories:   metadata.Categories,
		Tags:         metadata.Tags,
		PlannerModel: plannerModel,
		WriterModel:  writerModel,
		Deck:         metadata.Deck,
	}, nil
}

// extractTitle extracts the first # heading from markdown content
func (p *ArticleProcessor) extractTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "# ") {
			return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "#"))
		}
	}
	return ""
}

// extractDomain extracts the domain from a URL
func (p *ArticleProcessor) extractDomain(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsedURL.Host
}

// generateFilename creates a hash-based filename
func (p *ArticleProcessor) generateFilename(url, title string) string {
	slug := p.generateSlug(title)
	hash := p.generateURLHash(url)

	// Ensure output directory exists
	outputDir := p.config.Settings.OutputDirectory
	os.MkdirAll(outputDir, 0755)

	return filepath.Join(outputDir, fmt.Sprintf("%s-%s.md", slug, hash))
}

// generateSlug creates a URL-safe slug from title
func (p *ArticleProcessor) generateSlug(title string) string {
	// Convert to lowercase and replace spaces/special chars with hyphens
	slug := strings.ToLower(title)
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")

	// Limit length
	if len(slug) > 50 {
		slug = slug[:50]
	}

	return slug
}

// generateURLHash creates a short hash of the URL
func (p *ArticleProcessor) generateURLHash(url string) string {
	hash := sha256.Sum256([]byte(url))
	return fmt.Sprintf("%x", hash)[:8]
}

// findExistingFile finds an existing article file by URL
func (p *ArticleProcessor) findExistingFile(url string) string {
	outputDir := p.config.Settings.OutputDirectory
	urlHash := p.generateURLHash(url)

	// Check if output directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return ""
	}

	// Look for files with matching hash
	pattern := filepath.Join(outputDir, fmt.Sprintf("*-%s.md", urlHash))
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return ""
	}

	return matches[0]
}

// saveArticle saves the article to a file
func (p *ArticleProcessor) saveArticle(filename string, article *Article) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Create file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	// Template with full frontmatter
	tmplStr := `---
title: "{{.Title}}"
date: {{.CreatedAt.Format "2006-01-02T15:04:05Z07:00"}}
draft: {{.Draft}}
categories: [{{range $i, $cat := .Categories}}{{if $i}}, {{end}}"{{$cat}}"{{end}}]
tags: [{{range $i, $tag := .Tags}}{{if $i}}, {{end}}"{{$tag}}"{{end}}]
planner_model: "{{.PlannerModel}}"
writer_model: "{{.WriterModel}}"
deck: "{{.Deck}}"
source_url: "{{.SourceURL}}"
source_domain: "{{.SourceDomain}}"
---

{{.Content}}`

	tmpl, err := template.New("article").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	return tmpl.Execute(file, article)
}
