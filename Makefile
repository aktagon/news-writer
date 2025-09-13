# Makefile
.PHONY: build run run-url clean deps install test

# Build the application
build: test
	go build -buildvcs=false -o news-writer .

# Run with default config
run: build
	./news-writer -config .news-writer/news-articles.yaml

# Install dependencies
deps:
	go mod tidy
	go mod download

# Install binary to PATH
install: build
	cp news-writer /usr/local/bin/news-writer
	chmod +x /usr/local/bin/news-writer

# Clean build artifacts
clean:
	rm -f news-writer
	rm -rf articles/

# Create articles directory
setup:
	mkdir -p articles

# Run with custom config
run-config: build
	./news-writer -config $(CONFIG)

# Run tests
test:
	go test ./...

# Development run (builds and runs)
dev: clean deps build setup run
