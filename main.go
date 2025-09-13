package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var (
	rewriteMode      bool
	configFile       string
	apiKey           string
	writerPromptPath string
	templatePath     string
	debugMode        bool
)

var rootCmd = &cobra.Command{
	Use:   "news-writer [config-file]",
	Short: "Minimal article distillation system using AI",
	Long:  `A simplified tool for distilling web articles and PDFs using AI agents.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get config file path
		if len(args) > 0 {
			configFile = args[0]
		} else {
			configFile = "articles.yaml"
		}

		// Get API key
		if apiKey == "" {
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		}
		if apiKey == "" {
			log.Fatal("API key required: use --api-key flag or ANTHROPIC_API_KEY environment variable")
		}

		// Build config overrides
		overrides := &ConfigOverrides{}
		if writerPromptPath != "" {
			overrides.WriterPromptPath = &writerPromptPath
		}
		if templatePath != "" {
			overrides.TemplatePath = &templatePath
		}

		// Create processor with config overrides
		processor, err := NewArticleProcessor(apiKey, overrides)
		if err != nil {
			log.Fatalf("Failed to create processor: %v", err)
		}

		// Set debug mode globally
		if debugMode {
			SetDebugMode(true)
		}

		// Process URLs
		if rewriteMode {
			if len(args) == 0 {
				log.Fatal("URL required for rewrite mode")
			}
			_, err = processor.ProcessURL(args[0], true)
		} else {
			err = processor.ProcessURLsFromFile(configFile)
		}

		if err != nil {
			log.Fatalf("Processing failed: %v", err)
		}
	},
}

func init() {
	rootCmd.Flags().StringVar(&apiKey, "api-key", "", "Anthropic API key")
	rootCmd.Flags().BoolVar(&rewriteMode, "rewrite", false, "Rewrite a specific URL")
	rootCmd.Flags().StringVar(&writerPromptPath, "writer-prompt", "", "Path to custom writer prompt file")
	rootCmd.Flags().StringVar(&templatePath, "template", "", "Path to custom article template file")
	rootCmd.Flags().BoolVar(&debugMode, "debug", false, "Enable debug logging")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
