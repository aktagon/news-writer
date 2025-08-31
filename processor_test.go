package main

import (
	"encoding/json"
	"testing"
)

func TestPlanJSONUnmarshaling(t *testing.T) {
	validJSON := `{
		"title": "React Performance Guide",
		"deck": "Essential techniques for optimizing React applications",
		"key_points": ["Memoization reduces re-renders", "Code splitting improves load times"],
		"structure": ["Introduction", "Memoization techniques", "Bundle optimization"],
		"categories": ["Development/Web Development"],
		"tags": ["react", "performance"],
		"target": {
			"word_count": 1200,
			"tone": "practical"
		}
	}`

	var plan Plan
	err := json.Unmarshal([]byte(validJSON), &plan)
	if err != nil {
		t.Fatalf("Failed to unmarshal valid JSON: %v", err)
	}

	if plan.Title != "React Performance Guide" {
		t.Errorf("Expected title 'React Performance Guide', got '%s'", plan.Title)
	}
	if plan.Target.WordCount != 1200 {
		t.Errorf("Expected word count 1200, got %d", plan.Target.WordCount)
	}
	if len(plan.KeyPoints) != 2 {
		t.Errorf("Expected 2 key points, got %d", len(plan.KeyPoints))
	}
}

func TestPlanRequiredFields(t *testing.T) {
	plan := Plan{
		Title:      "Test Article",
		Deck:       "Test deck",
		KeyPoints:  []string{"Point 1"},
		Structure:  []string{"Section 1"},
		Categories: []string{"Technology/Programming"},
		Tags:       []string{"test"},
		Target: Target{
			WordCount: 800,
			Tone:      "informative",
		},
	}

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("Failed to marshal Plan: %v", err)
	}

	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	requiredFields := []string{"title", "deck", "key_points", "structure", "categories", "tags", "target"}
	for _, field := range requiredFields {
		if _, exists := jsonMap[field]; !exists {
			t.Errorf("Required field '%s' missing from JSON output", field)
		}
	}
}

func TestLoadSchema(t *testing.T) {
	ap := &ArticleProcessor{}

	schema, err := ap.loadSchema("planner")
	if err != nil {
		t.Fatalf("Failed to load planner schema: %v", err)
	}

	if schema == "" {
		t.Error("Schema should not be empty")
	}

	// Verify it's valid JSON
	var schemaObj map[string]interface{}
	err = json.Unmarshal([]byte(schema), &schemaObj)
	if err != nil {
		t.Errorf("Schema is not valid JSON: %v", err)
	}

	// Check for required schema properties
	if schemaObj["type"] != "object" {
		t.Error("Schema should have type 'object'")
	}

	if _, exists := schemaObj["properties"]; !exists {
		t.Error("Schema should have 'properties' field")
	}
}

func TestInvalidJSONHandling(t *testing.T) {
	tests := []string{
		`{"title": "Test"`, // incomplete JSON
		`{"title": }`,      // invalid syntax
		``,                 // empty string
	}

	for _, invalidJSON := range tests {
		var plan Plan
		err := json.Unmarshal([]byte(invalidJSON), &plan)
		if err == nil {
			t.Errorf("Expected error for invalid JSON: %s", invalidJSON)
		}
	}
}
