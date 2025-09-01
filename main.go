// main.go
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	apiKey            string
	overwrite         bool
	settingsFile      string
	plannerPromptFile string
	writerPromptFile  string
	plannerSchemaFile string
	templateFile      string
)

var rootCmd = &cobra.Command{
	Use:   "news-writer",
	Short: "Article distillation system using AI agents",
	Long: `A tool for distilling articles using AI agents.
Processes articles from YAML configuration files or CSV URLs.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var configSource string
		if len(args) > 0 {
			configSource = args[0]
		} else {
			configSource = GetConfigPath("news-articles.yaml")
		}

		newsArticlesURL, _ := cmd.Flags().GetString("url")
		if newsArticlesURL != "" {
			configSource = newsArticlesURL
		}

		runProcessor(configSource)
	},
}

var processCmd = &cobra.Command{
	Use:   "process [config-file]",
	Short: "Process articles from configuration file or URL",
	Long: `Process articles from a YAML configuration file or CSV URL.
If no config file is specified, uses the default news-articles.yaml.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var configSource string
		if len(args) > 0 {
			configSource = args[0]
		} else {
			configSource = GetConfigPath("news-articles.yaml")
		}

		newsArticlesURL, _ := cmd.Flags().GetString("url")
		if newsArticlesURL != "" {
			configSource = newsArticlesURL
		}

		runProcessor(configSource)
	},
}

var rewriteCmd = &cobra.Command{
	Use:   "rewrite <url>",
	Short: "Rewrite or create an article for a specific URL",
	Long: `Rewrite an existing article or create a new one for the specified URL.
If the URL already exists in the articles directory, it will be rewritten.
If it doesn't exist, it will be added to the YAML configuration and processed.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]
		configPath := GetConfigPath("news-articles.yaml")
		handleRewriteMode(url, configPath)
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "Anthropic API key (or set ANTHROPIC_API_KEY env var)")
	rootCmd.PersistentFlags().BoolVar(&overwrite, "overwrite", false, "Overwrite existing article files")
	rootCmd.PersistentFlags().StringVar(&settingsFile, "settings-file", "", "Path to custom settings.yaml file")
	rootCmd.PersistentFlags().StringVar(&plannerPromptFile, "planner-prompt-file", "", "Path to custom planner system prompt file")
	rootCmd.PersistentFlags().StringVar(&writerPromptFile, "writer-prompt-file", "", "Path to custom writer system prompt file")
	rootCmd.PersistentFlags().StringVar(&plannerSchemaFile, "planner-schema-file", "", "Path to custom planner output schema file")
	rootCmd.PersistentFlags().StringVar(&templateFile, "template-file", "", "Path to custom article template file")

	// Root command flags (for default process behavior)
	rootCmd.Flags().String("url", "", "URL to CSV file containing article URLs")

	// Process command flags
	processCmd.Flags().String("url", "", "URL to CSV file containing article URLs")

	// Add commands to root
	rootCmd.AddCommand(processCmd)
	rootCmd.AddCommand(rewriteCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runProcessor(configSource string) {
	// Get API key from environment if not provided
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		log.Fatal("API key required: use --api-key flag or ANTHROPIC_API_KEY environment variable")
	}

	// Create config overrides from command line flags
	var overrides *ConfigOverrides
	if settingsFile != "" || plannerPromptFile != "" || writerPromptFile != "" || plannerSchemaFile != "" || templateFile != "" {
		overrides = &ConfigOverrides{}
		if settingsFile != "" {
			overrides.SettingsPath = &settingsFile
		}
		if plannerPromptFile != "" {
			overrides.PlannerPromptPath = &plannerPromptFile
		}
		if writerPromptFile != "" {
			overrides.WriterPromptPath = &writerPromptFile
		}
		if plannerSchemaFile != "" {
			overrides.PlannerSchemaPath = &plannerSchemaFile
		}
		if templateFile != "" {
			overrides.TemplatePath = &templateFile
		}
	}

	// Create processor
	processor, err := NewArticleProcessor(apiKey, overrides)
	if err != nil {
		log.Fatalf("Failed to create processor: %v", err)
	}

	// Set overwrite flag
	processor.SetOverwrite(overwrite)

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

func handleRewriteMode(url, configPath string) {
	// Get API key from environment if not provided
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		log.Fatal("API key required: use --api-key flag or ANTHROPIC_API_KEY environment variable")
	}

	// Create config overrides from command line flags
	var overrides *ConfigOverrides
	if settingsFile != "" || plannerPromptFile != "" || writerPromptFile != "" || plannerSchemaFile != "" || templateFile != "" {
		overrides = &ConfigOverrides{}
		if settingsFile != "" {
			overrides.SettingsPath = &settingsFile
		}
		if plannerPromptFile != "" {
			overrides.PlannerPromptPath = &plannerPromptFile
		}
		if writerPromptFile != "" {
			overrides.WriterPromptPath = &writerPromptFile
		}
		if plannerSchemaFile != "" {
			overrides.PlannerSchemaPath = &plannerSchemaFile
		}
		if templateFile != "" {
			overrides.TemplatePath = &templateFile
		}
	}

	// Create processor
	processor, err := NewArticleProcessor(apiKey, overrides)
	if err != nil {
		log.Fatalf("Failed to create processor: %v", err)
	}

	// Force overwrite mode for rewrite command
	processor.SetOverwrite(true)

	// Check if URL already exists in articles
	existingFile := processor.FindExistingArticle(url)
	if existingFile != "" {
		log.Printf("Found existing article for %s: %s", url, existingFile)
		log.Printf("Rewriting article...")

		// Process the single item with existing filename to preserve it
		item := ArticleItem{URL: url}
		result := processor.ProcessItemWithFilename(item, existingFile)

		if result.Success {
			log.Printf("✓ Rewritten: %s", result.Filename)
		} else {
			log.Fatalf("✗ Failed to rewrite %s: %v", result.URL, result.Error)
		}
	} else {
		log.Printf("URL not found in existing articles. Checking configuration...")

		// Try to add URL to YAML configuration (will fail if already exists)
		err := addURLToConfig(configPath, url)
		if err != nil {
			// If URL already exists in config, that's fine - just process it
			if strings.Contains(err.Error(), "URL already exists in configuration") {
				log.Printf("URL already exists in configuration. Processing...")
			} else {
				log.Fatalf("Failed to add URL to configuration: %v", err)
			}
		} else {
			log.Printf("Added %s to %s", url, configPath)
		}

		// Process the single item
		item := ArticleItem{URL: url}
		result := processor.ProcessItem(item)

		if result.Success {
			log.Printf("✓ Created: %s", result.Filename)
		} else {
			log.Fatalf("✗ Failed to create %s: %v", result.URL, result.Error)
		}
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
