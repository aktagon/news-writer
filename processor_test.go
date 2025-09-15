package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{"first heading", "# Title\nsome content", "Title"},
		{"with spaces", "  # Spaced Title  \n", "Spaced Title"},
		{"multiple headings", "# First\n## Second\n# Third", "First"},
		{"no heading", "just text\nno heading", ""},
		{"empty content", "", ""},
		{"heading with prefix", "text\n# Real Title\nmore", "Real Title"},
	}

	p := &ArticleProcessor{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.extractTitle(tt.content)
			if result != tt.expected {
				t.Errorf("extractTitle() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected string
	}{
		{"basic", "Hello World", "hello-world"},
		{"special chars", "Title: With & Special!", "title-with-special"},
		{"unicode", "Café & Naïve", "caf-na-ve"},
		{"numbers", "React 18.2 Guide", "react-18-2-guide"},
		{"empty", "", ""},
		{"long title", strings.Repeat("word ", 20), strings.Repeat("word-", 10)[:50]},
		{"hyphen trimming", "---start---", "start"},
	}

	p := &ArticleProcessor{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.generateSlug(tt.title)
			if result != tt.expected {
				t.Errorf("generateSlug() = %q, want %q", result, tt.expected)
			}
			if len(result) > 50 {
				t.Errorf("generateSlug() result too long: %d chars", len(result))
			}
		})
	}
}

func TestGenerateURLHash(t *testing.T) {
	p := &ArticleProcessor{}

	url1 := "https://example.com/article1"
	url2 := "https://example.com/article2"

	hash1 := p.generateURLHash(url1)
	hash2 := p.generateURLHash(url2)

	if len(hash1) != 8 {
		t.Errorf("hash length = %d, want 8", len(hash1))
	}

	if hash1 == hash2 {
		t.Error("different URLs produced same hash")
	}

	hash1Again := p.generateURLHash(url1)
	if hash1 != hash1Again {
		t.Error("same URL produced different hashes")
	}
}

func TestSaveArticle(t *testing.T) {
	p := &ArticleProcessor{}
	tempDir := t.TempDir()

	article := &Article{
		Title:     "Test Title",
		SourceURL: "https://example.com",
		Content:   "# Test\n\nContent here",
		CreatedAt: time.Now(),
	}

	filename := filepath.Join(tempDir, "test.md")
	err := p.saveArticle(filename, article)
	if err != nil {
		t.Fatalf("saveArticle() error = %v", err)
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Test Title") {
		t.Error("saved file missing title")
	}

	if !strings.Contains(contentStr, "https://example.com") {
		t.Error("saved file missing source URL")
	}
}

func TestGenerateFilename(t *testing.T) {
	// Create a processor with mock config
	config := &Config{
		Settings: &Settings{
			OutputDirectory: "articles",
		},
	}
	p := &ArticleProcessor{
		config: config,
	}
	tempDir := t.TempDir()

	// Change to temp directory for test
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	// Generate filename
	filename := p.generateFilename("https://example.com", "Test Title")

	// Check for year/month in path
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	expectedDir := filepath.Join("articles", year, month)

	if !strings.HasPrefix(filename, expectedDir) {
		t.Errorf("expected filename to be in %s, got %s", expectedDir, filename)
	}

	// Check for slug and hash
	slug := "test-title"
	hash := p.generateURLHash("https://example.com")
	expectedSuffix := slug + "-" + hash + ".md"

	if !strings.HasSuffix(filename, expectedSuffix) {
		t.Errorf("expected filename to have suffix %s, got %s", expectedSuffix, filename)
	}
}

func TestFindExistingFile(t *testing.T) {
	// Create a processor with mock config
	config := &Config{
		Settings: &Settings{
			OutputDirectory: "articles",
		},
	}
	p := &ArticleProcessor{
		config: config,
	}
	tempDir := t.TempDir()

	// Change to temp directory for test
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	// Create articles directory
	os.MkdirAll("articles", 0755)

	// Test non-existent file
	result := p.findExistingFile("https://nonexistent.com")
	if result != "" {
		t.Errorf("expected empty string for non-existent file, got %s", result)
	}

	// Create test file
	hash := p.generateURLHash("https://example.com")
	testFile := filepath.Join("articles", "test-"+hash+".md")
	os.WriteFile(testFile, []byte("test"), 0644)

	// Test existing file
	result = p.findExistingFile("https://example.com")
	if result != testFile {
		t.Errorf("expected %s, got %s", testFile, result)
	}
}

func TestFindExistingFileRecursive(t *testing.T) {
	// Create a processor with mock config
	config := &Config{
		Settings: &Settings{
			OutputDirectory: "articles",
		},
	}
	p := &ArticleProcessor{
		config: config,
	}
	tempDir := t.TempDir()

	// Change to temp directory for test
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	// Create test file in a nested directory
	hash := p.generateURLHash("https://example.com/nested")
	nestedDir := filepath.Join("articles", "2025", "09")
	os.MkdirAll(nestedDir, 0755)
	nestedFile := filepath.Join(nestedDir, "nested-test-"+hash+".md")
	os.WriteFile(nestedFile, []byte("nested test"), 0644)

	// Test existing file (recursive)
	result := p.findExistingFile("https://example.com/nested")
	if result != nestedFile {
		t.Errorf("expected %s, got %s", nestedFile, result)
	}
}

func TestLoadURLsFromFile(t *testing.T) {
	p := &ArticleProcessor{}
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			"basic urls",
			"sources:\n  - url: \"https://example.com\"\n  - url: \"https://test.com\"",
			[]string{"https://example.com", "https://test.com"},
		},
		{
			"with empty url",
			"sources:\n  - url: \"https://example.com\"\n  - url: \"\"\n  - url: \"https://test.com\"",
			[]string{"https://example.com", "https://test.com"},
		},
		{
			"invalid urls",
			"sources:\n  - url: \"https://example.com\"\n  - url: \"invalid-url\"\n  - url: \"ftp://test.com\"",
			[]string{"https://example.com"},
		},
		{
			"empty sources",
			"sources: []",
			[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := filepath.Join(tempDir, "test.yaml")
			os.WriteFile(filename, []byte(tt.content), 0644)

			result, err := p.loadURLsFromFile(filename)
			if err != nil {
				t.Fatalf("loadURLsFromFile() error = %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("got %d URLs, want %d", len(result), len(tt.expected))
			}

			for i, url := range result {
				if i >= len(tt.expected) || url != tt.expected[i] {
					t.Errorf("URL %d: got %q, want %q", i, url, tt.expected[i])
				}
			}
		})
	}
}

func TestNewArticleProcessor(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		overrides *ConfigOverrides
		wantErr   bool
	}{
		{"valid api key", "test-key", nil, false},
		{"empty api key", "", nil, true},
		{"with overrides", "test-key", &ConfigOverrides{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor, err := NewArticleProcessor(tt.apiKey, tt.overrides)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.wantErr && processor == nil {
				t.Error("expected processor, got nil")
			}
		})
	}
}
