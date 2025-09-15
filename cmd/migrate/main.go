package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: migrate <add-hashes|remove-duplicates> <articles-directory>")
	}

	command := os.Args[1]
	articlesDir := os.Args[2]

	switch command {
	case "add-hashes":
		if err := addHashes(articlesDir); err != nil {
			log.Fatal(err)
		}
	case "remove-duplicates":
		if err := removeDuplicates(articlesDir); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("Unknown command %q", command)
	}
}

func addHashes(articlesDir string) error {
	return filepath.WalkDir(articlesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Continue on errors
		}

		if !d.IsDir() && strings.HasSuffix(path, ".md") {
			if err := processFile(path); err != nil {
				log.Printf("Error processing %s: %v", path, err)
			}
		}

		return nil
	})
}

func processFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", filePath, err)
	}

	sourceURL := extractSourceURL(string(content))
	if sourceURL == "" {
		log.Printf("No source_url found in %s, skipping", filePath)
		return nil
	}

	hash := generateURLHash(sourceURL)

	fileName := filepath.Base(filePath)
	if hasHash(fileName) {
		log.Printf("File %s already has hash, skipping", fileName)
		return nil
	}

	dir := filepath.Dir(filePath)
	nameWithoutExt := strings.TrimSuffix(fileName, ".md")
	newFileName := fmt.Sprintf("%s-%s.md", nameWithoutExt, hash)
	newFilePath := filepath.Join(dir, newFileName)

	log.Printf("Renaming %s -> %s", fileName, newFileName)
	return os.Rename(filePath, newFilePath)
}

func extractSourceURL(content string) string {
	re := regexp.MustCompile(`source_url:\s*"([^"]*)"`)
	matches := re.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func generateURLHash(url string) string {
	h := sha256.Sum256([]byte(url))
	return fmt.Sprintf("%x", h)[:8]
}

func hasHash(fileName string) bool {
	re := regexp.MustCompile(`-[0-9a-f]{8}\.md$`)
	return re.MatchString(fileName)
}

func removeDuplicates(articlesDir string) error {
	hashToFiles := make(map[string][]string)
	reader := bufio.NewReader(os.Stdin)

	if err := filepath.WalkDir(articlesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Continue on errors
		}

		if !d.IsDir() && strings.HasSuffix(path, ".md") {
			if hash := extractHash(filepath.Base(path)); hash != "" {
				hashToFiles[hash] = append(hashToFiles[hash], path)
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("walking directory: %w", err)
	}

	totalRemoved := 0
	for hash, files := range hashToFiles {
		if len(files) <= 1 {
			continue
		}

		fmt.Printf("\nFound %d duplicates with hash %s:\n", len(files), hash)
		for i, file := range files {
			fileName := filepath.Base(file)
			if i == 0 {
				fmt.Printf("  KEEP: %s\n", fileName)
				continue
			}

			if confirmDelete(reader, file) {
				if err := os.Remove(file); err != nil {
					log.Printf("Error removing %s: %v", file, err)
				} else {
					totalRemoved++
					fmt.Printf("  REMOVED: %s\n", fileName)
				}
			} else {
				fmt.Printf("  SKIP: %s\n", fileName)
			}
		}
	}

	fmt.Printf("\nRemoved %d duplicate files\n", totalRemoved)
	return nil
}

func extractHash(fileName string) string {
	re := regexp.MustCompile(`-([0-9a-f]{8})\.md$`)
	matches := re.FindStringSubmatch(fileName)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func confirmDelete(reader *bufio.Reader, path string) bool {
	for {
		fmt.Printf("  DELETE %s? [y/N]: ", filepath.Base(path))
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading input: %v", err)
			return false
		}
		response := strings.ToLower(strings.TrimSpace(input))
		switch response {
		case "y", "yes":
			return true
		case "", "n", "no":
			return false
		default:
			fmt.Println("  Please enter y or n.")
		}
	}
}
