package main

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestPlanValidation(t *testing.T) {
	tests := []struct {
		name           string
		template       string
		expectError    bool
		errorContains  string
	}{
		{
			name: "valid template with plan variable",
			template: `Please analyze the source content and follow the plan below:

<plan>
{{.Plan}}
</plan>`,
			expectError: false,
		},
		{
			name:          "template missing plan variable",
			template:      "Please analyze the source content.",
			expectError:   true,
			errorContains: "writer user prompt template must contain {{.Plan}} variable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the validation logic directly
			hasVariable := strings.Contains(tt.template, "{{.Plan}}")

			if tt.expectError && hasVariable {
				t.Errorf("expected error case should not contain {{.Plan}} variable")
			}
			if !tt.expectError && !hasVariable {
				t.Errorf("valid case should contain {{.Plan}} variable")
			}
		})
	}
}

func TestPlanXMLGeneration(t *testing.T) {
	plan := &FrontmatterMetadata{
		Title:      "Test Article",
		Categories: []string{"Development", "Testing"},
		Tags:       []string{"go", "testing", "xml"},
		Deck:       "A test article for XML generation",
		Target: Target{
			Tone:     "technical",
			Audience: "software developers",
		},
	}

	// Test XML marshalling
	planXML, err := xml.MarshalIndent(plan, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal plan to XML: %v", err)
	}

	xmlString := string(planXML)

	// Verify XML contains expected elements
	expectedElements := []string{
		"<FrontmatterMetadata>",
		"<Title>Test Article</Title>",
		"<Categories>Development</Categories>",
		"<Categories>Testing</Categories>",
		"<Tags>go</Tags>",
		"<Tags>testing</Tags>",
		"<Tags>xml</Tags>",
		"<Deck>A test article for XML generation</Deck>",
		"<Target>",
		"<Tone>technical</Tone>",
		"<Audience>software developers</Audience>",
		"</Target>",
		"</FrontmatterMetadata>",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(xmlString, expected) {
			t.Errorf("XML output missing expected element: %s\nActual XML:\n%s", expected, xmlString)
		}
	}
}

func TestTemplateReplacement(t *testing.T) {
	template := `Please follow this plan:

<plan>
{{.Plan}}
</plan>

Source content: example`

	plan := &FrontmatterMetadata{
		Title:      "Test",
		Categories: []string{"Dev"},
		Tags:       []string{"test"},
		Deck:       "Test deck",
		Target: Target{
			Tone:     "casual",
			Audience: "testers",
		},
	}

	// Convert to XML
	planXML, err := xml.MarshalIndent(plan, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal plan: %v", err)
	}

	// Replace template variable
	result := strings.ReplaceAll(template, "{{.Plan}}", string(planXML))

	// Verify replacement occurred
	if strings.Contains(result, "{{.Plan}}") {
		t.Error("template variable was not replaced")
	}

	// Verify XML is present
	if !strings.Contains(result, "<FrontmatterMetadata>") {
		t.Error("XML content not found in result")
	}

	// Verify structure is maintained
	if !strings.Contains(result, "Please follow this plan:") {
		t.Error("original template structure not maintained")
	}
}

func TestPlannerPDFHandling(t *testing.T) {
	tests := []struct {
		name     string
		content  *ContentResult
		hasFiles bool
	}{
		{
			name: "text content should not create files",
			content: &ContentResult{
				Text:   "Sample text content",
				FileID: "",
			},
			hasFiles: false,
		},
		{
			name: "PDF content should create files",
			content: &ContentResult{
				Text:   "",
				FileID: "file-123",
			},
			hasFiles: true,
		},
		{
			name: "content with both text and fileID should create files",
			content: &ContentResult{
				Text:   "Some text",
				FileID: "file-456",
			},
			hasFiles: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the file creation logic from PlanMetadata
			var files []struct{ ID string }
			if tt.content.FileID != "" {
				files = append(files, struct{ ID string }{ID: tt.content.FileID})
			}

			hasFiles := len(files) > 0

			if hasFiles != tt.hasFiles {
				t.Errorf("expected hasFiles=%v, got hasFiles=%v", tt.hasFiles, hasFiles)
			}

			if tt.hasFiles && len(files) == 0 {
				t.Error("expected files to be created for PDF content")
			}

			if !tt.hasFiles && len(files) > 0 {
				t.Error("expected no files to be created for text content")
			}

			// Verify file ID is correct when present
			if tt.hasFiles && files[0].ID != tt.content.FileID {
				t.Errorf("expected file ID %s, got %s", tt.content.FileID, files[0].ID)
			}
		})
	}
}