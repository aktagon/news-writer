// main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	// Command line flags
	var (
		configPath        = flag.String("config", GetConfigPath("news-articles.yaml"), "Path to the articles configuration file")
		newsArticlesURL   = flag.String("news-articles-url", "", "URL to CSV file containing article URLs")
		apiKey            = flag.String("api-key", "", "Anthropic API key (or set ANTHROPIC_API_KEY env var)")
		overwrite         = flag.Bool("overwrite", false, "Overwrite existing article files")
		settingsFile      = flag.String("settings-file", "", "Path to custom settings.yaml file")
		plannerPromptFile = flag.String("planner-prompt-file", "", "Path to custom planner system prompt file")
		writerPromptFile  = flag.String("writer-prompt-file", "", "Path to custom writer system prompt file")
		plannerSchemaFile = flag.String("planner-schema-file", "", "Path to custom planner output schema file")
		templateFile      = flag.String("template-file", "", "Path to custom article template file")
	)
	flag.Parse()

	// Validate that only one config source is provided
	if *configPath != GetConfigPath("news-articles.yaml") && *newsArticlesURL != "" {
		log.Fatal("Cannot specify both -config and -news-articles-url flags")
	}

	var configSource string
	if *newsArticlesURL != "" {
		configSource = *newsArticlesURL
	} else {
		configSource = *configPath
	}

	// Get API key from environment if not provided
	if *apiKey == "" {
		*apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if *apiKey == "" {
		log.Fatal("API key required: use -api-key flag or ANTHROPIC_API_KEY environment variable")
	}

	// Create config overrides from command line flags
	var overrides *ConfigOverrides
	if *settingsFile != "" || *plannerPromptFile != "" || *writerPromptFile != "" || *plannerSchemaFile != "" || *templateFile != "" {
		overrides = &ConfigOverrides{}
		if *settingsFile != "" {
			overrides.SettingsPath = settingsFile
		}
		if *plannerPromptFile != "" {
			overrides.PlannerPromptPath = plannerPromptFile
		}
		if *writerPromptFile != "" {
			overrides.WriterPromptPath = writerPromptFile
		}
		if *plannerSchemaFile != "" {
			overrides.PlannerSchemaPath = plannerSchemaFile
		}
		if *templateFile != "" {
			overrides.TemplatePath = templateFile
		}
	}

	// Create processor
	processor, err := NewArticleProcessor(*apiKey, overrides)
	if err != nil {
		log.Fatalf("Failed to create processor: %v", err)
	}

	// Set overwrite flag
	processor.SetOverwrite(*overwrite)

	// Check if this is the first run (config file doesn't exist and is default)
	if configSource == GetConfigPath("news-articles.yaml") {
		if _, err := os.Stat(configSource); os.IsNotExist(err) {
			showFirstRunMessage()
			return
		}
	}

	// Process articles
	log.Printf("Starting article distillation from %s", configSource)
	results, err := processor.ProcessArticles(configSource)
	if err != nil {
		log.Fatalf("Failed to process articles: %v", err)
	}

	// Report results
	successful := 0
	failed := 0
	for _, result := range results {
		if result.Success {
			successful++
		} else {
			failed++
		}
	}

	log.Printf("Processing complete: %d successful, %d failed", successful, failed)

	if failed > 0 {
		os.Exit(1)
	}
}

// showFirstRunMessage displays instructions for first-time users
func showFirstRunMessage() {
	fmt.Printf("Welcome to news-writer! Configuration files have been created in %s\n", defaultConfigDir)
	fmt.Printf("\n")
	fmt.Printf("Configuration files:\n")
	fmt.Printf("  %s  - Article URLs to process\n", GetConfigPath("news-articles.yaml"))
	fmt.Printf("  %s         - Agent settings and output directory\n", GetConfigPath("settings.yaml"))
	fmt.Printf("  %s  - Article output template\n", GetConfigPath("news-article-template.md"))
	fmt.Printf("  %s  - AI agent prompts (customizable)\n", GetConfigPath("*-system-prompt.md"))
	fmt.Printf("\n")
	fmt.Printf("To get started, create the articles configuration file:\n")
	fmt.Printf("  %s\n", GetConfigPath("news-articles.yaml"))
	fmt.Printf("\n")
	fmt.Printf("Example configuration:\n")
	fmt.Printf(`  items:
    - url: "https://example.com/article1"
    - url: "https://example.com/article2"`)
	fmt.Printf("\n\n")
	fmt.Printf("Then run the command again to process articles.\n")
}
