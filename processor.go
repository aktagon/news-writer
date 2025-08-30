// processor.go
package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/aktagon/llmkit/anthropic/agents"
	"gopkg.in/yaml.v3"
)

// ArticleProcessor handles the main workflow
type ArticleProcessor struct {
	plannerAgent *agents.ChatAgent
	writerAgent  *agents.ChatAgent
	fetcher      *ContentFetcher
	settings     *Settings
}

// NewArticleProcessor creates a new processor with configured agents
func NewArticleProcessor(apiKey string) (*ArticleProcessor, error) {
	// Load settings
	settings, err := loadSettings("config/settings.yaml")
	if err != nil {
		return nil, fmt.Errorf("loading settings: %w", err)
	}

	// Create planner agent
	plannerAgent, err := agents.New(apiKey)
	if err != nil {
		return nil, fmt.Errorf("creating planner agent: %w", err)
	}

	// Create writer agent
	writerAgent, err := agents.New(apiKey)
	if err != nil {
		return nil, fmt.Errorf("creating writer agent: %w", err)
	}

	// Create content fetcher
	fetcher := NewContentFetcher()

	return &ArticleProcessor{
		plannerAgent: plannerAgent,
		writerAgent:  writerAgent,
		fetcher:      fetcher,
		settings:     settings,
	}, nil
}

// ProcessArticles processes all articles from the config file or URL
func (ap *ArticleProcessor) ProcessArticles(configSource string) ([]ProcessingResult, error) {
	var config *Config
	var err error

	if strings.HasPrefix(configSource, "http://") || strings.HasPrefix(configSource, "https://") {
		config, err = ap.loadConfigFromURL(configSource)
	} else {
		config, err = ap.loadConfig(configSource)
	}

	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	results := make([]ProcessingResult, 0, len(config.Items))

	for _, item := range config.Items {
		result := ap.processItem(item)
		results = append(results, result)

		if result.Success {
			log.Printf("✓ Generated: %s", result.Filename)
		} else {
			log.Printf("✗ Failed %s: %v", result.URL, result.Error)
		}
	}

	return results, nil
}

// processItem processes a single article item
func (ap *ArticleProcessor) processItem(item ArticleItem) ProcessingResult {
	filename := ap.generateFilename(item)

	// Skip if file already exists
	if ap.fileExists(filename) {
		log.Printf("Skipping %s: article exists", item.URL)
		return ProcessingResult{
			URL:      item.URL,
			Success:  true,
			Filename: filename,
		}
	}

	// Fetch source content
	sourceContent, err := ap.fetcher.FetchContent(item.URL)
	if err != nil {
		return ProcessingResult{
			URL:   item.URL,
			Error: fmt.Errorf("fetching source: %w", err),
		}
	}

	// Generate plan
	plan, err := ap.generatePlan(sourceContent)
	if err != nil {
		return ProcessingResult{
			URL:   item.URL,
			Error: fmt.Errorf("generating plan: %w", err),
		}
	}

	// Generate article
	article, err := ap.generateArticle(sourceContent, plan, item.URL)
	if err != nil {
		return ProcessingResult{
			URL:   item.URL,
			Error: fmt.Errorf("generating article: %w", err),
		}
	}

	// Save article
	err = ap.saveArticle(filename, article)
	if err != nil {
		return ProcessingResult{
			URL:   item.URL,
			Error: fmt.Errorf("saving article: %w", err),
		}
	}

	return ProcessingResult{
		URL:      item.URL,
		Success:  true,
		Filename: filename,
	}
}

// generatePlan uses the planner agent to create a plan
func (ap *ArticleProcessor) generatePlan(sourceContent string) (*Plan, error) {
	systemPrompt, err := ap.loadSystemPrompt("planner")
	if err != nil {
		return nil, fmt.Errorf("loading planner system prompt: %w", err)
	}

	prompt := fmt.Sprintf("Source content:\n%s", sourceContent)

	schema, err := ap.loadSchema("planner")
	if err != nil {
		return nil, fmt.Errorf("loading planner schema: %w", err)
	}

	response, err := ap.plannerAgent.Chat(prompt, &agents.ChatOptions{
		SystemPrompt: systemPrompt,
		Schema:       schema,
		MaxTokens:    ap.settings.Agents.Planner.MaxTokens,
		Temperature:  ap.settings.Agents.Planner.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("planner agent chat: %w", err)
	}

	var plan Plan
	err = json.Unmarshal([]byte(response.Text), &plan)
	if err != nil {
		return nil, fmt.Errorf("parsing plan JSON: %w", err)
	}

	return &plan, nil
}

// generateArticle uses the writer agent to create the final article
func (ap *ArticleProcessor) generateArticle(sourceContent string, plan *Plan, sourceURL string) (*Article, error) {
	systemPrompt, err := ap.loadSystemPrompt("writer")
	if err != nil {
		return nil, fmt.Errorf("loading writer system prompt: %w", err)
	}

	planJSON, _ := json.Marshal(plan)

	prompt := fmt.Sprintf(`Plan:
%s

Source content:
%s`, planJSON, sourceContent)

	response, err := ap.writerAgent.Chat(prompt, &agents.ChatOptions{
		SystemPrompt: systemPrompt,
		MaxTokens:    ap.settings.Agents.Writer.MaxTokens,
		Temperature:  ap.settings.Agents.Writer.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("writer agent chat: %w", err)
	}

	article := &Article{
		Title:        plan.Title,
		Source:       extractDomain(sourceURL),
		SourceURL:    sourceURL,
		Content:      response.Text,
		CreatedAt:    time.Now(),
		Deck:         plan.Deck,
		Categories:   plan.Categories,
		Tags:         plan.Tags,
		Author:       "Signal Editorial Team",
		AuthorTitle:  "AI-generated content, human-reviewed",
		SourceDomain: extractDomain(sourceURL),
	}

	return article, nil
}

// loadConfig loads the YAML configuration file
func (ap *ArticleProcessor) loadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// loadConfigFromURL loads configuration from a CSV URL
func (ap *ArticleProcessor) loadConfigFromURL(url string) (*Config, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Fetch the CSV content
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching CSV from URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d when fetching CSV", resp.StatusCode)
	}

	// Parse CSV
	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	// Skip header row if it exists (check if first row contains "url" header)
	startIdx := 0
	if len(records) > 0 && len(records[0]) > 0 && strings.ToLower(records[0][0]) == "url" {
		startIdx = 1
	}

	// Convert CSV rows to Config struct
	config := &Config{
		Items: make([]ArticleItem, 0, len(records)-startIdx),
	}

	for i := startIdx; i < len(records); i++ {
		row := records[i]
		if len(row) == 0 || strings.TrimSpace(row[0]) == "" {
			continue // Skip empty rows
		}

		url := strings.TrimSpace(row[0])
		if url != "" {
			config.Items = append(config.Items, ArticleItem{
				URL: url,
			})
		}
	}

	return config, nil
}

// generateFilename creates a filename from the item
func (ap *ArticleProcessor) generateFilename(item ArticleItem) string {
	slug := generateSlug(item.URL)
	currentDate := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s/%s-%s.md", ap.settings.OutputDirectory, currentDate, slug)
}

// fileExists checks if a file already exists
func (ap *ArticleProcessor) fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// saveArticle saves the article to a markdown file using the template
func (ap *ArticleProcessor) saveArticle(filename string, article *Article) error {
	// Ensure output directory exists
	dir := filepath.Dir(filename)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	// Load template
	templateData, err := os.ReadFile(ap.settings.TemplatePath)
	if err != nil {
		return fmt.Errorf("reading template %s: %w", ap.settings.TemplatePath, err)
	}

	// Parse template
	tmpl, err := template.New("article").Parse(string(templateData))
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, article)
	if err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	return os.WriteFile(filename, buf.Bytes(), 0644)
}

// generateSlug creates a URL slug from a URL
func generateSlug(url string) string {
	// Extract domain/path parts
	re := regexp.MustCompile(`https?://(?:www\.)?([^/]+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) < 2 {
		return "article"
	}

	domain := matches[1]
	parts := strings.Split(domain, ".")
	if len(parts) > 0 {
		slug := parts[0]
		// Clean the slug
		slug = strings.ToLower(slug)
		slug = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(slug, "-")
		slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")
		slug = strings.Trim(slug, "-")
		if slug == "" {
			return "article"
		}
		return slug
	}

	return "article"
}

// loadSystemPrompt loads a system prompt from a file
func (ap *ArticleProcessor) loadSystemPrompt(agentName string) (string, error) {
	filename := fmt.Sprintf("agents/%s/system-prompt.md", agentName)
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("reading system prompt %s: %w", filename, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// loadSchema loads a JSON schema from a file
func (ap *ArticleProcessor) loadSchema(agentName string) (string, error) {
	filename := fmt.Sprintf("agents/%s/output-schema.json", agentName)
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("reading schema %s: %w", filename, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// extractDomain extracts the domain name from a URL
func extractDomain(url string) string {
	re := regexp.MustCompile(`https?://(?:www\.)?([^/]+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) >= 2 {
		return matches[1]
	}
	return url
}

// loadSettings loads settings from YAML file
func loadSettings(settingsPath string) (*Settings, error) {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		// Return default settings if file doesn't exist
		return &Settings{
			OutputDirectory: "articles",
			TemplatePath:    "config/news-article-template.md",
			Agents: struct {
				Planner AgentSettings `yaml:"planner"`
				Writer  AgentSettings `yaml:"writer"`
			}{
				Planner: AgentSettings{MaxTokens: 2000, Temperature: 0.0},
				Writer:  AgentSettings{MaxTokens: 3000, Temperature: 0.3},
			},
		}, nil
	}

	var settings Settings
	err = yaml.Unmarshal(data, &settings)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}
