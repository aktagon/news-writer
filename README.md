# Article Distillation System

A Go application that automatically distills web articles into concise, insightful summaries using AI agents.

## Features

- **Automated Processing**: Processes multiple articles from a YAML configuration
- **Dual Agent Architecture**: Separate planner and writer agents for optimal results
- **Web Content Fetching**: Automatically retrieves and cleans web content
- **Markdown Output**: Generates clean markdown files with proper citations
- **Graceful Error Handling**: Continues processing even if individual articles fail
- **Duplicate Detection**: Skips articles that have already been processed

## Architecture

```
Load YAML → Fetch Content → Plan Article → Write Article → Save Markdown
```

The system uses two AI agents:
- **Planner Agent**: Analyzes source content and creates a structured plan
- **Writer Agent**: Follows the plan to create the final distilled article

## Installation

```bash
# Install dependencies
make install

# Set up environment
export ANTHROPIC_API_KEY=your_api_key_here

# Create articles directory
make setup
```

## Usage

Create an `articles.yaml` file:

```yaml
items:
  - id: 1
    source_url: "https://example.com/article1"
    discussion_url: "https://news.ycombinator.com/item?id=123" # optional
  - id: 2
    source_url: "https://example.com/article2"
```

Run the distiller:

```bash
# Build and run with default config
make run

# Run with custom config
make run-config CONFIG=my-articles.yaml

# Or run directly
./bin/article-distiller -config articles.yaml -api-key your_key
```

## Output

Articles are saved as `articles/{id}-{slug}.md`:

```markdown
# React Performance: Essential Optimization Techniques
*Source: [react.dev](https://react.dev/learn/thinking-in-react)*

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

- `--config`: Path to YAML configuration file (default: `articles.yaml`)
- `--api-key`: Anthropic API key (or set `ANTHROPIC_API_KEY` env var)

## Development

```bash
# Development run (clean, build, setup, run)
make dev

# Build only
make build

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
3. **Filename**: `articles/{id}-{slug}.md`
4. **Skip Existing**: Won't overwrite existing files
5. **Fail Gracefully**: Logs errors and continues processing
