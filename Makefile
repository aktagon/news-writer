# Makefile
.PHONY: build run clean install

# Build the application
build:
	go build -o bin/article-distiller .

# Run with example config
run: build
	./bin/article-distiller -config articles.yaml

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
	./bin/article-distiller -config $(CONFIG)

# Development run (builds and runs)
dev: clean build setup run
