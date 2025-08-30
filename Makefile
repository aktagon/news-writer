# Makefile
.PHONY: build run run-url clean install

# Build the application
build:
	go build -o bin/news-writer .

# Run with default config
run: build
	./bin/news-writer -config config/news-articles.yaml

# Install dependencies
install:
	go mod tidy
	go mod download

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf articles/

# Create articles directory
setup:
	mkdir -p articles

# Run with custom config
run-config: build
	./bin/news-writer -config $(CONFIG)

# Run with CSV URL
run-url: build
	./bin/news-writer -news-articles-url $(URL)

# Example: Run with Google Sheets CSV
example-sheets: build
	./bin/news-writer -news-articles-url "https://docs.google.com/spreadsheets/d/e/2PACX-1vTRHf3kQ8z8MqcodGRHoX00t56ewg0JTXF-BNz2E2gDSz7KCnzWcvupT-0OgAdJK-CBWpHjnIpzpmwo/pub?gid=0&single=true&output=csv"

# Development run (builds and runs)
dev: clean build setup run
