# News Writer

AI-powered article distillation system with structured category taxonomy and flexible tagging.

## Features

- **Two-Level Category System**: Fixed category groups with specific categories for consistent organization
- **Flexible Tagging**: Content-specific metadata for technologies, frameworks, concepts
- **Automated Processing**: Processes multiple articles from YAML files or CSV URLs
- **Dual Agent Architecture**: Separate planner and writer agents for optimal results
- **Web Content Fetching**: Automatically retrieves and cleans web content
- **Structured Output**: Articles with proper categorization and metadata
- **Graceful Error Handling**: Continues processing even if individual articles fail
- **Duplicate Detection**: Skips articles that have already been processed

## Category System

### Two-Level Structure

**Level 1: Category Groups** (8 groups for navigation)
- Development, Artificial Intelligence, Technology, Data & Analytics
- Business & Strategy, Security, Infrastructure, Crypto & Blockchain

**Level 2: Categories** (35 specific categories for content assignment)
- Each article assigned exactly one category: `programming`, `web-development`, `machine-learning`, etc.

**Flexible Tags**: Independent content metadata (React, Python, performance, security, etc.)

## Architecture

```
Load Config (YAML/CSV) → Fetch Content → Plan with Category → Write Article → Save with Metadata
```

The system uses two AI agents:

- **Planner Agent**: Analyzes content and selects appropriate category from fixed taxonomy
- **Writer Agent**: Creates distilled article structured for the assigned category

## Installation

### Option 1: Download Binary

Download the latest binary from [releases](https://github.com/aktagon/news-writer/releases):

```bash
# macOS (ARM)
curl -L https://github.com/aktagon/news-writer/releases/latest/download/news-writer-darwin-arm64 -o news-writer
chmod +x news-writer
sudo mv news-writer /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/aktagon/news-writer/releases/latest/download/news-writer-darwin-amd64 -o news-writer
chmod +x news-writer
sudo mv news-writer /usr/local/bin/

# Linux
curl -L https://github.com/aktagon/news-writer/releases/latest/download/news-writer-linux-amd64 -o news-writer
chmod +x news-writer
sudo mv news-writer /usr/local/bin/
```

### Option 2: Build from Source

```bash
# Clone and build
git clone https://github.com/aktagon/news-writer.git
cd news-writer

# Install dependencies
make deps

# Build binary
make build

# Install to PATH (requires sudo)
sudo make install

# Set up environment
export ANTHROPIC_API_KEY=your_api_key_here

# Create articles directory
mkdir -p articles
```

## Usage

### Option 1: YAML Configuration

Create an `articles.yaml` file:

```yaml
items:
  - url: "https://example.com/article1"
  - url: "https://example.com/article2"
```

### Option 2: CSV URL (Google Sheets)

Use a CSV URL with articles in the first column:

```csv
url
https://example.com/article1
https://example.com/article2
```

Run the distiller:

```bash
# Run with default YAML config
news-writer -config articles.yaml

# Run with custom YAML file
news-writer -config my-articles.yaml

# Run with CSV URL
news-writer -news-articles-url "https://docs.google.com/spreadsheets/d/e/2PACX-1vTRHf3kQ8z8MqcodGRHoX00t56ewg0JTXF-BNz2E2gDSz7KCnzWcvupT-0OgAdJK-CBWpHjnIpzpmwo/pub?gid=0&single=true&output=csv"

# Specify API key directly
news-writer -config articles.yaml -api-key your_key
```

## Output

Articles are saved as `articles/{date}-{slug}.md`:

```markdown
# React Performance: Essential Optimization Techniques

_Source: [react.dev](https://react.dev/learn/thinking-in-react)_

React apps slow down as they grow. Here are the techniques that make the biggest impact.

## Memoization Techniques

Use React.memo() for expensive components...

## Conclusion

- Start with React.memo() for expensive renders
- Implement route-based code splitting
- Use React DevTools profiler to measure impact
```

## Configuration

The system accepts various configuration options:

- `--config`: Path to YAML configuration file (default: `config/news-articles.yaml`)
- `--news-articles-url`: URL to CSV file containing article URLs (mutually exclusive with `--config`)
- `--api-key`: Anthropic API key (or set `ANTHROPIC_API_KEY` env var)

### CSV Format

When using `--news-articles-url`, the CSV should have URLs in the first column:

```csv
url
https://kmicinski.com/functional-programming/2025/08/01/loops/
https://news.ycombinator.com/item?id=44837949
```

The header row is optional and will be automatically detected and skipped.

## Development

```bash
# Install dependencies
make deps

# Development run (clean, install deps, build, setup, run)
make dev

# Build only
make build

# Install to PATH
make install

# Clean artifacts
make clean
```

## Dependencies

- Go 1.21+
- [llmkit](https://github.com/aktagon/llmkit) - AI agent framework
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML processing

## Rules

1. **Citations**: All articles link to original source
2. **Length**: 500-1500 words maximum
3. **Filename**: `articles/{date}-{slug}.md`
4. **Skip Existing**: Won't overwrite existing files
5. **Fail Gracefully**: Logs errors and continues processing
