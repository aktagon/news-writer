package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestProcessItemWithFilename(t *testing.T) {
	// Create a temporary file to test existing file behavior
	tmpFile, err := os.CreateTemp("", "test-article-*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	item := ArticleItem{URL: "https://example.com/test"}

	ap := &ArticleProcessor{
		overwrite: false,
		settings: &Settings{
			OutputDirectory: "articles",
		},
	}

	result := ap.ProcessItemWithFilename(item, tmpFile.Name())

	// Should skip processing since file exists and overwrite is false
	if result.URL != item.URL {
		t.Errorf("Expected URL '%s', got '%s'", item.URL, result.URL)
	}
	if result.Filename != tmpFile.Name() {
		t.Errorf("Expected filename '%s', got '%s'", tmpFile.Name(), result.Filename)
	}
	if !result.Success {
		t.Error("Expected success when file exists and overwrite is false")
	}
	if result.Error != nil {
		t.Errorf("Expected no error, got %v", result.Error)
	}
}

func TestFileExists(t *testing.T) {
	ap := &ArticleProcessor{}

	// Test with existing file
	tmpFile, err := os.CreateTemp("", "test-exists-*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if !ap.fileExists(tmpFile.Name()) {
		t.Error("fileExists should return true for existing file")
	}

	// Test with non-existing file
	nonExistent := tmpFile.Name() + "-does-not-exist"
	if ap.fileExists(nonExistent) {
		t.Error("fileExists should return false for non-existing file")
	}
}

func TestFindExistingByHash(t *testing.T) {
	tmpDir := t.TempDir()

	ap := &ArticleProcessor{
		settings: &Settings{
			OutputDirectory: tmpDir,
		},
	}

	// Test URL
	testURL := "https://example.com/test-article"
	hash := generateURLHash(testURL)

	// Create nested directory structure
	yearDir := filepath.Join(tmpDir, "2025")
	monthDir := filepath.Join(yearDir, "09")
	err := os.MkdirAll(monthDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}

	// Create test file with hash suffix
	testFilename := filepath.Join(monthDir, "some-title-"+hash+".md")
	file, err := os.Create(testFilename)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	file.Close()

	// Test finding existing file by hash
	found := ap.findExistingByHash(testURL)
	if found != testFilename {
		t.Errorf("Expected to find %s, got %s", testFilename, found)
	}

	// Test with non-existing hash
	nonExistentURL := "https://example.com/non-existent"
	found = ap.findExistingByHash(nonExistentURL)
	if found != "" {
		t.Errorf("Expected empty string for non-existent hash, got %s", found)
	}
}

func TestHashBasedIdempotency(t *testing.T) {
	tmpDir := t.TempDir()

	ap := &ArticleProcessor{
		overwrite: false,
		settings: &Settings{
			OutputDirectory: tmpDir,
		},
	}

	// Test URL and hash
	testURL := "https://example.com/test-article"
	hash := generateURLHash(testURL)

	// Create nested directory structure and existing file
	yearDir := filepath.Join(tmpDir, "2025")
	monthDir := filepath.Join(yearDir, "09")
	err := os.MkdirAll(monthDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}

	existingFile := filepath.Join(monthDir, "existing-title-"+hash+".md")
	file, err := os.Create(existingFile)
	if err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}
	file.Close()

	// Test ProcessItem - should skip processing and return existing filename
	item := ArticleItem{URL: testURL}
	result := ap.ProcessItem(item)

	if !result.Success {
		t.Errorf("Expected success, got error: %v", result.Error)
	}
	if result.Filename != existingFile {
		t.Errorf("Expected filename %s, got %s", existingFile, result.Filename)
	}
	if result.URL != testURL {
		t.Errorf("Expected URL %s, got %s", testURL, result.URL)
	}

	// Verify the behavior is truly idempotent - no new files created
	matches, err := filepath.Glob(filepath.Join(tmpDir, "*/*", "*-"+hash+".md"))
	if err != nil {
		t.Fatalf("Error globbing files: %v", err)
	}
	if len(matches) != 1 {
		t.Errorf("Expected exactly 1 file with hash %s, found %d: %v", hash, len(matches), matches)
	}
}

func TestIdempotencyBug(t *testing.T) {
	// Test that demonstrates the idempotency bug:
	// initial filename check uses URL hash, but final filename uses title slug

	item := ArticleItem{URL: "https://example.com/test-article"}

	ap := &ArticleProcessor{
		overwrite: false,
		settings: &Settings{
			OutputDirectory: "articles",
		},
	}

	// Generate the initial filename (used for existence check)
	slug := generateSlug(item.URL)
	hash := generateURLHash(item.URL)
	year := "2025"
	month := "09"
	initialFilename := fmt.Sprintf("%s/%s/%s/%s-%s.md", ap.settings.OutputDirectory, year, month, slug, hash)

	// Simulate what happens after the article is generated - filename uses title slug
	mockArticle := &Article{
		Title:     "Survey: A Third of Senior Developers Say Over Half Their Code is AI-Generated",
		SourceURL: item.URL,
		CreatedAt: time.Now(),
	}
	finalFilename := ap.generateFilenameFromArticle(mockArticle)

	// These should be the same for idempotency to work, but they're not
	if initialFilename == finalFilename {
		t.Errorf("Bug not reproduced - filenames should be different but are the same: %s", initialFilename)
	}

	t.Logf("Initial filename (used for check): %s", initialFilename)
	t.Logf("Final filename (actually saved):    %s", finalFilename)
	t.Logf("This is why idempotency fails - the check uses one filename but saves to another!")
}
