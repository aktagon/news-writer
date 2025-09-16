package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"strings"

	"github.com/aktagon/llmkit/anthropic"
	"github.com/aktagon/llmkit/anthropic/agents"
	"github.com/aktagon/llmkit/anthropic/types"
)

// Target represents the target audience and tone for the article
type Target struct {
	Tone     string `json:"tone"`
	Audience string `json:"audience"`
}

// FrontmatterMetadata represents the metadata extracted by the planner agent
type FrontmatterMetadata struct {
	Title      string   `json:"title"`
	Categories []string `json:"categories"`
	Tags       []string `json:"tags"`
	Deck       string   `json:"deck"`
	Target     Target   `json:"target"`
}

// AgentManager handles AI agent creation and management
type AgentManager struct {
	writerAgent  *agents.ChatAgent
	plannerAgent *agents.ChatAgent
	config       *Config
	apiKey       string
}

// NewAgentManager creates a new AgentManager with writer and planner agents
func NewAgentManager(apiKey string, config *Config) (*AgentManager, error) {
	writerAgent, err := agents.New(apiKey)
	if err != nil {
		return nil, fmt.Errorf("creating writer agent: %w", err)
	}

	plannerAgent, err := agents.New(apiKey)
	if err != nil {
		return nil, fmt.Errorf("creating planner agent: %w", err)
	}

	return &AgentManager{
		writerAgent:  writerAgent,
		plannerAgent: plannerAgent,
		config:       config,
		apiKey:       apiKey,
	}, nil
}

// Write generates article content using the writer agent
func (am *AgentManager) Write(content *ContentResult, plan *FrontmatterMetadata) (string, error) {
	log.Printf("→ Writing...")
	systemPrompt := am.config.GetWriterSystemPrompt()
	userPromptTemplate := am.config.GetWriterUserPrompt()

	// Validate that template contains required variables
	if !strings.Contains(userPromptTemplate, "{{.Plan}}") {
		return "", fmt.Errorf("writer user prompt template must contain {{.Plan}} variable")
	}

	// Convert plan metadata to XML
	planXML, err := xml.MarshalIndent(plan, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal plan to XML: %w", err)
	}

	// Replace template variables
	userPrompt := strings.ReplaceAll(userPromptTemplate, "{{.Plan}}", string(planXML))

	// For text content, add it to the user prompt
	if content.Text != "" {
		userPrompt = fmt.Sprintf(`%s

Source content:
%s`, userPrompt, content.Text)
	}

	var files []types.File
	if content.FileID != "" {
		files = append(files, types.File{ID: content.FileID})
	}

	settings := types.RequestSettings{
		Model:       am.config.Settings.Agents.Writer.Model,
		MaxTokens:   am.config.Settings.Agents.Writer.MaxTokens,
		Temperature: am.config.Settings.Agents.Writer.Temperature,
		// TopK:        0,
		// TopP:        0.0,
	}
	response, err := anthropic.PromptWithSettings(systemPrompt, userPrompt, "", am.apiKey, settings, files...)
	if err != nil {
		return "", fmt.Errorf("writer agent failed: %w", err)
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	log.Printf("✓ Writing completed")
	return response.Content[0].Text, nil
}

// PlanMetadata generates frontmatter metadata using the planner agent with structured output
func (am *AgentManager) PlanMetadata(url string, content *ContentResult) (*FrontmatterMetadata, error) {
	log.Printf("→ Planning %s", url)
	// Limit source content to configured token limit
	limitedContent := am.limitContentTokens(content.Text, am.config.Settings.Agents.Planner.ContentMaxTokens)

	// Build categories list for the system prompt
	categoriesList := strings.Join(am.config.Settings.Categories, "\n- ")

	// Get prompts and validate template variables
	systemPromptTemplate := am.config.GetPlannerSystemPrompt()
	if !strings.Contains(systemPromptTemplate, "{{.categories}}") {
		return nil, fmt.Errorf("planner system prompt template must contain {{.categories}} variable")
	}
	systemPrompt := strings.ReplaceAll(systemPromptTemplate, "{{.categories}}", "- "+categoriesList)

	userPromptTemplate := am.config.GetPlannerUserPrompt()
	if !strings.Contains(userPromptTemplate, "{{.source_content}}") {
		return nil, fmt.Errorf("planner user prompt template must contain {{.source_content}} variable")
	}
	userPrompt := strings.ReplaceAll(userPromptTemplate, "{{.source_content}}", limitedContent)

	// Get schema for structured output
	schema := am.config.GetPlannerSchema()

	// Handle PDF files
	var files []types.File
	if content.FileID != "" {
		files = append(files, types.File{ID: content.FileID})
	}

	// Use structured output with schema
	settings := types.RequestSettings{
		Model:       am.config.Settings.Agents.Planner.Model,
		MaxTokens:   am.config.Settings.Agents.Planner.MaxTokens,
		Temperature: am.config.Settings.Agents.Planner.Temperature,
		TopK:        0,
		TopP:        0.0,
	}
	response, err := anthropic.PromptWithSettings(systemPrompt, userPrompt, schema, am.apiKey, settings, files...)
	if err != nil {
		return nil, fmt.Errorf("planner agent failed: %w", err)
	}

	if len(response.Content) == 0 {
		return nil, fmt.Errorf("no content in planner response")
	}

	// Parse structured JSON response
	var metadata FrontmatterMetadata
	if err := json.Unmarshal([]byte(response.Content[0].Text), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse planner structured response: %w", err)
	}

	log.Printf("✓ Planned: %s | Categories: %v | Tags: %v | Deck: %s", metadata.Title, metadata.Categories, metadata.Tags, metadata.Deck)
	return &metadata, nil
}

// limitContentTokens limits content to approximately N tokens (using 4 chars ≈ 1 token)
func (am *AgentManager) limitContentTokens(content string, maxTokens int) string {
	maxChars := maxTokens * 4 // Rough approximation: 4 chars ≈ 1 token
	if len(content) <= maxChars {
		return content
	}
	return content[:maxChars] + "..."
}

// GetModelInfo returns the model information for both agents
func (am *AgentManager) GetModelInfo() (plannerModel, writerModel string) {
	return am.config.Settings.Agents.Planner.Model, am.config.Settings.Agents.Writer.Model
}
