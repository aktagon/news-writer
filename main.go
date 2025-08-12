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
		configPath = flag.String("config", "articles.yaml", "Path to the articles configuration file")
		apiKey     = flag.String("api-key", "", "Anthropic API key (or set ANTHROPIC_API_KEY env var)")
	)
	flag.Parse()

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

	// Process articles
	log.Printf("Starting article distillation from %s", *configPath)
	results, err := processor.ProcessArticles(*configPath)
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
