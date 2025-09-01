// processor.go
package main

import (
	"bytes"
	_ "embed"
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

const defaultConfigDir = ".news-writer/"

// GetConfigPath returns the full path to a config file
func GetConfigPath(filename string) string {
	return filepath.Join(defaultConfigDir, filename)
}

// ConfigOverrides holds file path overrides for embedded configurations
type ConfigOverrides struct {
	PlannerPromptPath *string
	WriterPromptPath  *string
	PlannerSchemaPath *string
	TemplatePath      *string
	SettingsPath      *string
}

//go:embed config/planner-system-prompt.md
var plannerSystemPrompt string

//go:embed config/writer-system-prompt.md
var writerSystemPrompt string

//go:embed config/planner-output-schema.json
var plannerSchema string

//go:embed config/settings.yaml
var defaultSettings string

//go:embed config/news-article-template.md
var defaultTemplate string

//go:embed config/news-articles.yaml
var defaultNewsArticles string

// ArticleProcessor handles the main workflow
type ArticleProcessor struct {
	plannerAgent *agents.ChatAgent
	writerAgent  *agents.ChatAgent
	fetcher      *ContentFetcher
	settings     *Settings
	overwrite    bool
	overrides    *ConfigOverrides
}

// NewArticleProcessor creates a new processor with configured agents
func NewArticleProcessor(apiKey string, overrides *ConfigOverrides) (*ArticleProcessor, error) {
	// Ensure embedded config files are written to config/ on first run
	err := ensureConfigExists()
	if err != nil {
		return nil, fmt.Errorf("ensuring config files exist: %w", err)
	}

	// Load settings with override
	var settings *Settings
	if overrides != nil && overrides.SettingsPath != nil {
		// Explicit settings file must exist
		settings, err = loadSettingsRequired(*overrides.SettingsPath)
		if err != nil {
			log.Fatalf("Critical error: settings file missing: %s - %v", *overrides.SettingsPath, err)
		}
	} else {
		// Default settings path - use fallback if missing
		settingsPath := filepath.Join(defaultConfigDir, "settings.yaml")
		settings, err = loadSettings(settingsPath)
		if err != nil {
			return nil, fmt.Errorf("loading settings: %w", err)
		}
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
		overwrite:    false,
		overrides:    overrides,
	}, nil
}

// SetOverwrite sets the overwrite flag
func (ap *ArticleProcessor) SetOverwrite(overwrite bool) {
	ap.overwrite = overwrite
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

	log.Printf("Processing %d articles...", len(config.Items))

	for i, item := range config.Items {
		log.Printf("[%d/%d] Processing: %s", i+1, len(config.Items), item.URL)
		result := ap.ProcessItem(item)
		results = append(results, result)

		if result.Success {
			log.Printf("✓ Generated: %s", result.Filename)
		} else {
			log.Printf("✗ Failed %s: %v", result.URL, result.Error)
		}
	}

	return results, nil
}

// ProcessItem processes a single article item
func (ap *ArticleProcessor) ProcessItem(item ArticleItem) ProcessingResult {
	return ap.ProcessItemWithFilename(item, "")
}

// ProcessItemWithFilename processes a single article item with an optional existing filename
func (ap *ArticleProcessor) ProcessItemWithFilename(item ArticleItem, existingFilename string) ProcessingResult {
	// Skip if article for this URL already exists and overwrite is false
	if existingFile := ap.FindExistingArticle(item.URL); existingFile != "" && !ap.overwrite {
		log.Printf("Skipping %s: article exists (%s)", item.URL, existingFile)
		return ProcessingResult{
			URL:      item.URL,
			Success:  true,
			Filename: existingFile,
		}
	}

	// Fetch source content
	log.Printf("  → Fetching content...")
	sourceContent, err := ap.fetcher.FetchContent(item.URL)
	if err != nil {
		return ProcessingResult{
			URL:   item.URL,
			Error: fmt.Errorf("fetching source: %w", err),
		}
	}

	// Generate plan
	log.Printf("  → Generating plan...")
	plan, err := ap.generatePlan(sourceContent)
	if err != nil {
		return ProcessingResult{
			URL:   item.URL,
			Error: fmt.Errorf("generating plan: %w", err),
		}
	}

	// Generate article first to get the date
	log.Printf("  → Writing article: %s", plan.Title)
	article, err := ap.generateArticle(sourceContent, plan, item.URL)
	if err != nil {
		return ProcessingResult{
			URL:   item.URL,
			Error: fmt.Errorf("generating article: %w", err),
		}
	}

	// Generate filename using article date or existing filename
	var filename string
	if existingFilename != "" {
		filename = existingFilename
	} else {
		filename = ap.generateFilenameFromArticle(article)
	}

	// Check if file exists and skip if overwrite is false
	if !ap.overwrite && ap.fileExists(filename) {
		log.Printf("Skipping %s: file exists (%s)", item.URL, filename)
		return ProcessingResult{
			URL:      item.URL,
			Success:  true,
			Filename: filename,
		}
	}

	// Save article
	log.Printf("  → Saving to: %s", filename)
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
	systemPrompt, err := ap.loadPlannerSystemPrompt()
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
	systemPrompt, err := ap.loadWriterSystemPrompt()
	if err != nil {
		return nil, fmt.Errorf("loading writer system prompt: %w", err)
	}

	// Marshal plan as JSON for system prompt
	planJSON, _ := json.Marshal(plan)

	// Always append plan context and instructions to system prompt (user cannot override)
	enhancedSystemPrompt := fmt.Sprintf(`%s

<plan>
%s
</plan>

MANDATORY INSTRUCTIONS:
- Follow the structure outlined in the plan exactly
- Match your writing style and emphasis to the plan's category: %s
- Use the specified tone: %s
- Include all key points from the plan
- Target word count: %d words
- IMPORTANT: Do NOT include the <plan> tags or plan content in your response. Only use the plan for guidance.`,
		systemPrompt, string(planJSON), strings.Join(plan.Categories, ", "), plan.Target.Tone, plan.Target.WordCount)

	// User prompt contains only the source content
	userPrompt := fmt.Sprintf("Source content:\n%s", sourceContent)

	response, err := ap.writerAgent.Chat(userPrompt, &agents.ChatOptions{
		SystemPrompt: enhancedSystemPrompt,
		MaxTokens:    ap.settings.Agents.Writer.MaxTokens,
		Temperature:  ap.settings.Agents.Writer.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("writer agent chat: %w", err)
	}

	// Clean up response: remove any plan tags that might have been included
	cleanedContent := cleanPlanTags(response.Text)

	article := &Article{
		Title:          plan.Title,
		Source:         extractDomain(sourceURL),
		SourceURL:      sourceURL,
		Content:        cleanedContent,
		CreatedAt:      time.Now(),
		Deck:           plan.Deck,
		Categories:     plan.Categories,
		Tags:           plan.Tags,
		Author:         "Signal Editorial Team",
		AuthorTitle:    "AI generated and human reviewed news meta-commentary.",
		SourceDomain:   extractDomain(sourceURL),
		TargetAudience: plan.Target.Audience,
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

// generateFilenameFromTitle creates a filename from the article title
func (ap *ArticleProcessor) generateFilenameFromTitle(title string) string {
	slug := generateSlugFromTitle(title)
	currentDate := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s/%s-%s.md", ap.settings.OutputDirectory, currentDate, slug)
}

// generateFilenameFromArticle creates a filename using the date from article frontmatter
func (ap *ArticleProcessor) generateFilenameFromArticle(article *Article) string {
	slug := generateSlugFromTitle(article.Title)
	// Use article date in YYYY/MM/slug.md format
	year := article.CreatedAt.Format("2006")
	month := article.CreatedAt.Format("01")
	return fmt.Sprintf("%s/%s/%s/%s.md", ap.settings.OutputDirectory, year, month, slug)
}

// fileExists checks if a file already exists
func (ap *ArticleProcessor) fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// FindExistingArticle checks if an article for the given URL already exists by checking frontmatter
func (ap *ArticleProcessor) FindExistingArticle(url string) string {
	files, err := filepath.Glob(filepath.Join(ap.settings.OutputDirectory, "*.md"))
	if err != nil {
		return ""
	}

	searchPattern := fmt.Sprintf(`source_url: "%s"`, url)

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		if strings.Contains(string(content), searchPattern) {
			return file
		}
	}

	return ""
}

// saveArticle saves the article to a markdown file using the template
func (ap *ArticleProcessor) saveArticle(filename string, article *Article) error {
	// Ensure output directory exists
	dir := filepath.Dir(filename)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	// Load template with override
	var templateData []byte
	if ap.overrides != nil && ap.overrides.TemplatePath != nil {
		// Only read from file if explicitly overridden
		data, err := os.ReadFile(*ap.overrides.TemplatePath)
		if err != nil {
			log.Fatalf("Critical error: template file missing: %s - %v", *ap.overrides.TemplatePath, err)
		}
		templateData = data
	} else {
		// Use embedded template by default
		templateData = []byte(defaultTemplate)
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

// generateSlugFromTitle creates a URL slug from an article title
func generateSlugFromTitle(title string) string {
	if title == "" {
		return "article"
	}

	// Convert to lowercase and replace spaces/special chars with hyphens
	slug := strings.ToLower(title)
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")

	// Limit length to avoid filesystem issues
	if len(slug) > 50 {
		slug = slug[:50]
		slug = strings.Trim(slug, "-")
	}

	if slug == "" {
		return "article"
	}

	return slug
}

// loadPlannerSystemPrompt loads planner prompt and appends categories
func (ap *ArticleProcessor) loadPlannerSystemPrompt() (string, error) {
	var basePrompt string

	if ap.overrides != nil && ap.overrides.PlannerPromptPath != nil {
		data, err := os.ReadFile(*ap.overrides.PlannerPromptPath)
		if err != nil {
			log.Fatalf("Critical error: planner prompt file missing: %s - %v", *ap.overrides.PlannerPromptPath, err)
		}
		basePrompt = strings.TrimSpace(string(data))
	} else {
		basePrompt = strings.TrimSpace(plannerSystemPrompt)
	}

	categoriesJSON, _ := json.MarshalIndent(ap.settings.Categories, "", "  ")
	return basePrompt + "\n\nAvailable categories:\n" + string(categoriesJSON), nil
}

// loadWriterSystemPrompt loads writer prompt
func (ap *ArticleProcessor) loadWriterSystemPrompt() (string, error) {
	if ap.overrides != nil && ap.overrides.WriterPromptPath != nil {
		data, err := os.ReadFile(*ap.overrides.WriterPromptPath)
		if err != nil {
			log.Fatalf("Critical error: writer prompt file missing: %s - %v", *ap.overrides.WriterPromptPath, err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	return strings.TrimSpace(writerSystemPrompt), nil
}

// loadSchema loads a JSON schema from config directory or embedded data
func (ap *ArticleProcessor) loadSchema(agentName string) (string, error) {
	if ap.overrides != nil && ap.overrides.PlannerSchemaPath != nil && agentName == "planner" {
		data, err := os.ReadFile(*ap.overrides.PlannerSchemaPath)
		if err != nil {
			log.Fatalf("Critical error: planner schema file missing: %s - %v", *ap.overrides.PlannerSchemaPath, err)
		}
		return strings.TrimSpace(string(data)), nil
	}

	// Use embedded data
	switch agentName {
	case "planner":
		return strings.TrimSpace(plannerSchema), nil
	case "writer":
		return "", fmt.Errorf("writer agent does not use schema")
	default:
		return "", fmt.Errorf("unknown agent: %s", agentName)
	}
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

// loadSettings loads settings from YAML file with fallback to defaults
func loadSettings(settingsPath string) (*Settings, error) {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		// Return default settings if file doesn't exist
		return &Settings{
			OutputDirectory: "articles",
			TemplatePath:    filepath.Join(defaultConfigDir, "news-article-template.md"),
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

// loadSettingsRequired loads settings from YAML file, failing if file doesn't exist
func loadSettingsRequired(settingsPath string) (*Settings, error) {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, err
	}

	var settings Settings
	err = yaml.Unmarshal(data, &settings)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

// ensureConfigExists creates config directory and writes settings.yaml if needed
func ensureConfigExists() error {
	// Ensure config directory exists
	err := os.MkdirAll(defaultConfigDir, 0755)
	if err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Write settings.yaml - this should be customized by users
	settingsFile := filepath.Join(defaultConfigDir, "settings.yaml")
	if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
		err = os.WriteFile(settingsFile, []byte(defaultSettings), 0644)
		if err != nil {
			return fmt.Errorf("writing settings.yaml: %w", err)
		}
	}

	return nil
}

// addURLToConfig adds a URL to the YAML configuration file
func addURLToConfig(configPath, url string) error {
	// Validate URL format
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("invalid URL format: %s (must start with http:// or https://)", url)
	}

	// Check if config file exists
	var config *Config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create new config with empty items
		config = &Config{
			Items: []ArticleItem{},
		}
	} else {
		// Load existing config
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("reading config file: %w", err)
		}

		config = &Config{}
		err = yaml.Unmarshal(data, config)
		if err != nil {
			return fmt.Errorf("parsing config file: %w", err)
		}
	}

	// Check if URL already exists in config
	for _, item := range config.Items {
		if item.URL == url {
			return fmt.Errorf("URL already exists in configuration: %s", url)
		}
	}

	// Add new URL to config
	config.Items = append(config.Items, ArticleItem{URL: url})

	// Marshal config back to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// Ensure config directory exists
	dir := filepath.Dir(configPath)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Write config file
	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// cleanPlanTags removes any <plan>...</plan> sections from the content using regex
func cleanPlanTags(content string) string {
	// Remove <plan>...</plan> tags and their content (including nested content and newlines)
	planTagRegex := regexp.MustCompile(`(?s)<plan>.*?</plan>\s*`)
	return planTagRegex.ReplaceAllString(content, "")
}
