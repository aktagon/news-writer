package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const minContentMaxTokens = 2000

// ConfigOverrides allows overriding embedded defaults with file paths
type ConfigOverrides struct {
	WriterPromptPath  *string
	PlannerPromptPath *string
	PlannerSchemaPath *string
	TemplatePath      *string
}

// Embedded configuration files
//
//go:embed .news-writer/writer-system-prompt.md
var defaultWriterSystemPrompt string

//go:embed .news-writer/writer-user-prompt.md
var defaultWriterUserPrompt string

//go:embed .news-writer/planner-system-prompt.md
var defaultPlannerSystemPrompt string

//go:embed .news-writer/planner-user-prompt.md
var defaultPlannerUserPrompt string

//go:embed .news-writer/planner-output-schema.json
var defaultPlannerSchema string

//go:embed .news-writer/news-article-template.md
var defaultTemplate string

// Settings represents the YAML configuration structure
type Settings struct {
	OutputDirectory string `yaml:"output_directory"`
	TemplatePath    string `yaml:"template_path"`
	Agents          struct {
		Planner struct {
			Model            string  `yaml:"model"`
			MaxTokens        int     `yaml:"max_tokens"`
			Temperature      float64 `yaml:"temperature"`
			ContentMaxTokens int     `yaml:"content_max_tokens"`
		} `yaml:"planner"`
		Writer struct {
			Model       string  `yaml:"model"`
			MaxTokens   int     `yaml:"max_tokens"`
			Temperature float64 `yaml:"temperature"`
		} `yaml:"writer"`
	} `yaml:"agents"`
	Categories []string `yaml:"categories"`
}

// Config holds configuration and overrides
type Config struct {
	Settings  *Settings
	Overrides *ConfigOverrides
}

// NewConfig creates a new Config with settings and overrides
func NewConfig(overrides *ConfigOverrides) (*Config, error) {
	settings, err := loadSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}

	return &Config{
		Settings:  settings,
		Overrides: overrides,
	}, nil
}

// GetWriterSystemPrompt returns the writer system prompt (from override file or embedded)
func (c *Config) GetWriterSystemPrompt() string {
	if c.Overrides != nil && c.Overrides.WriterPromptPath != nil {
		if content, err := os.ReadFile(*c.Overrides.WriterPromptPath); err == nil {
			return string(content)
		}
	}
	return defaultWriterSystemPrompt
}

// GetWriterUserPrompt returns the writer user prompt (embedded only for now)
func (c *Config) GetWriterUserPrompt() string {
	return defaultWriterUserPrompt
}

// GetPlannerSystemPrompt returns the planner system prompt (from override file or embedded)
func (c *Config) GetPlannerSystemPrompt() string {
	if c.Overrides != nil && c.Overrides.PlannerPromptPath != nil {
		if content, err := os.ReadFile(*c.Overrides.PlannerPromptPath); err == nil {
			return string(content)
		}
	}
	return defaultPlannerSystemPrompt
}

// GetPlannerUserPrompt returns the planner user prompt (embedded only for now)
func (c *Config) GetPlannerUserPrompt() string {
	return defaultPlannerUserPrompt
}

// GetPlannerSchema returns the planner schema (from override file or embedded)
func (c *Config) GetPlannerSchema() string {
	if c.Overrides != nil && c.Overrides.PlannerSchemaPath != nil {
		if content, err := os.ReadFile(*c.Overrides.PlannerSchemaPath); err == nil {
			return string(content)
		}
	}
	return defaultPlannerSchema
}

// GetTemplate returns the template (from override file or embedded)
func (c *Config) GetTemplate() string {
	if c.Overrides != nil && c.Overrides.TemplatePath != nil {
		if content, err := os.ReadFile(*c.Overrides.TemplatePath); err == nil {
			return string(content)
		}
	}
	return defaultTemplate
}

// loadSettings loads settings from the default location
func loadSettings() (*Settings, error) {
	settingsPath := getConfigPath("settings.yaml")

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read settings file %s: %w", settingsPath, err)
	}

	var settings Settings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings YAML: %w", err)
	}

	// Ensure ContentMaxTokens is at least the minimum
	if settings.Agents.Planner.ContentMaxTokens < minContentMaxTokens {
		log.Printf("Warning: planner.content_max_tokens is %d, defaulting to %d (minimum)", settings.Agents.Planner.ContentMaxTokens, minContentMaxTokens)
		settings.Agents.Planner.ContentMaxTokens = minContentMaxTokens
	}

	return &settings, nil
}

// getConfigPath returns the path to a config file in .news-writer directory
func getConfigPath(filename string) string {
	return filepath.Join(".news-writer", filename)
}

// ensureConfigExists creates the config directory and default files if they don't exist
func ensureConfigExists() error {
	configDir := ".news-writer"

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	// Write default settings if it doesn't exist
	settingsPath := getConfigPath("settings.yaml")
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		defaultSettings := `output_directory: articles
template_path: .news-writer/news-article-template.md
agents:
  planner:
    model: claude-sonnet-4-20250514
    max_tokens: 1000
    temperature: 0.0
    content_max_tokens: 2000
  writer:
    model: claude-sonnet-4-20250514
    max_tokens: 6000
    temperature: 0.2
categories:
  - "Development/Programming"
  - "Technology/Innovation"
  - "Artificial Intelligence/Large Language Models"
`
		if err := os.WriteFile(settingsPath, []byte(defaultSettings), 0644); err != nil {
			return fmt.Errorf("failed to write default settings: %w", err)
		}
	}

	return nil
}
