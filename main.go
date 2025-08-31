// main.go
package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	// Command line flags
	var (
		configPath      = flag.String("config", GetConfigPath("news-articles.yaml"), "Path to the articles configuration file")
		newsArticlesURL = flag.String("news-articles-url", "", "URL to CSV file containing article URLs")
		apiKey          = flag.String("api-key", "", "Anthropic API key (or set ANTHROPIC_API_KEY env var)")
		overwrite       = flag.Bool("overwrite", false, "Overwrite existing article files")
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

	// Create processor
	processor, err := NewArticleProcessor(*apiKey)
	if err != nil {
		log.Fatalf("Failed to create processor: %v", err)
	}

	// Set overwrite flag
	processor.SetOverwrite(*overwrite)

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
