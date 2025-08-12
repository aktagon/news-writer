// processor.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aktagon/llmkit/anthropic/agents"
	"gopkg.in/yaml.v3"
)

// ArticleProcessor handles the main workflow
type ArticleProcessor struct {
	plannerAgent *agents.ChatAgent
	writerAgent  *agents.ChatAgent
	fetcher      *ContentFetcher
}

// NewArticleProcessor creates a new processor with configured agents
func NewArticleProcessor(apiKey string) (*ArticleProcessor, error) {
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
	}, nil
}

// ProcessArticles processes all articles from the config file
func (ap *ArticleProcessor) ProcessArticles(configPath string) ([]ProcessingResult, error) {
	config, err := ap.loadConfig(configPath)
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
			log.Printf("✗ Failed %d: %v", result.ID, result.Error)
		}
	}

	return results, nil
}

// processItem processes a single article item
func (ap *ArticleProcessor) processItem(item ArticleItem) ProcessingResult {
	filename := ap.generateFilename(item)

	// Skip if file already exists
	if ap.fileExists(filename) {
		log.Printf("Skipping %d: article exists", item.ID)
		return ProcessingResult{
			ID:       item.ID,
			Success:  true,
			Filename: filename,
		}
	}

	// Fetch source content
	sourceContent, err := ap.fetcher.FetchContent(item.SourceURL)
	if err != nil {
		return ProcessingResult{
			ID:    item.ID,
			Error: fmt.Errorf("fetching source: %w", err),
		}
	}

	// Fetch discussion content if provided
	var discussionContent string
	if item.DiscussionURL != "" {
		discussionContent, err = ap.fetcher.FetchContent(item.DiscussionURL)
		if err != nil {
			log.Printf("Warning: failed to fetch discussion for %d: %v", item.ID, err)
		}
	}

	// Generate plan
	plan, err := ap.generatePlan(sourceContent, discussionContent)
	if err != nil {
		return ProcessingResult{
			ID:    item.ID,
			Error: fmt.Errorf("generating plan: %w", err),
		}
	}

	// Generate article
	article, err := ap.generateArticle(sourceContent, plan, discussionContent, item.SourceURL)
	if err != nil {
		return ProcessingResult{
			ID:    item.ID,
			Error: fmt.Errorf("generating article: %w", err),
		}
	}

	// Save article
	err = ap.saveArticle(filename, article)
	if err != nil {
		return ProcessingResult{
			ID:    item.ID,
			Error: fmt.Errorf("saving article: %w", err),
		}
	}

	return ProcessingResult{
		ID:       item.ID,
		Success:  true,
		Filename: filename,
	}
}

// generatePlan uses the planner agent to create a plan
func (ap *ArticleProcessor) generatePlan(sourceContent, discussionContent string) (*Plan, error) {
	systemPrompt, err := ap.loadSystemPrompt("planner")
	if err != nil {
		return nil, fmt.Errorf("loading planner system prompt: %w", err)
	}

	prompt := fmt.Sprintf("Source content:\n%s", sourceContent)
	if discussionContent != "" {
		prompt += fmt.Sprintf("\n\nDiscussion content:\n%s", discussionContent)
	}

	schema, err := ap.loadSchema("planner")
	if err != nil {
		return nil, fmt.Errorf("loading planner schema: %w", err)
	}

	response, err := ap.plannerAgent.Chat(prompt, &agents.ChatOptions{
		SystemPrompt: systemPrompt,
		Schema:       schema,
		MaxTokens:    2000,
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
func (ap *ArticleProcessor) generateArticle(sourceContent string, plan *Plan, discussionContent, sourceURL string) (*Article, error) {
	systemPrompt, err := ap.loadSystemPrompt("writer")
	if err != nil {
		return nil, fmt.Errorf("loading writer system prompt: %w", err)
	}

	planJSON, _ := json.Marshal(plan)

	prompt := fmt.Sprintf(`Plan:
%s

Source content:
%s`, planJSON, sourceContent)

	if discussionContent != "" {
		prompt += fmt.Sprintf("\n\nDiscussion content:\n%s", discussionContent)
	}

	response, err := ap.writerAgent.Chat(prompt, &agents.ChatOptions{
		SystemPrompt: systemPrompt,
		MaxTokens:    3000,
	})
	if err != nil {
		return nil, fmt.Errorf("writer agent chat: %w", err)
	}

	article := &Article{
		Title:     plan.Title,
		Source:    extractDomain(sourceURL),
		SourceURL: sourceURL,
		Content:   response.Text,
		CreatedAt: time.Now(),
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

// generateFilename creates a filename from the item
func (ap *ArticleProcessor) generateFilename(item ArticleItem) string {
	slug := generateSlug(item.SourceURL)
	return fmt.Sprintf("articles/%d-%s.md", item.ID, slug)
}

// fileExists checks if a file already exists
func (ap *ArticleProcessor) fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// saveArticle saves the article to a markdown file
func (ap *ArticleProcessor) saveArticle(filename string, article *Article) error {
	// Ensure articles directory exists
	dir := filepath.Dir(filename)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	// Format as markdown
	content := fmt.Sprintf(`# %s
*Source: [%s](%s)*

%s
`, article.Title, article.Source, article.SourceURL, article.Content)

	return os.WriteFile(filename, []byte(content), 0644)
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
	filename := fmt.Sprintf("agents/%s/system.md", agentName)
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
