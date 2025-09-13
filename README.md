# News Writer

Minimal AI-powered article distillation system with flexible configuration and custom templates.

## Features

- **Simplified Architecture**: Clean, modular design focused on core functionality
- **Configuration-driven**: YAML-based settings with embedded defaults
- **Custom Templates**: Override prompts and templates without rebuilding
- **Rich Metadata**: Articles include categories, tags, deck summaries, and model information
- **Rewrite Mode**: Process individual URLs with optional overwriting
- **Debug Mode**: Detailed logging for troubleshooting
- **Graceful Error Handling**: Continues processing when individual articles fail
- **Duplicate Detection**: Skips articles that already exist (unless in rewrite mode)

## Architecture

```
Load Config (YAML) → Fetch Content → Plan Article → Write Article → Save with Frontmatter
```

The system uses two AI agents:

- **Planner Agent**: Analyzes content and creates structured metadata and outline
- **Writer Agent**: Generates the final distilled article following the plan

## Installation

### Option 1: Build from Source

```bash
git clone https://github.com/aktagon/news-writer.git
cd news-writer

# Install dependencies and build
make deps
make build

# Install to PATH (optional)
sudo make install

# Set up environment
export ANTHROPIC_API_KEY=your_api_key_here
```

### Option 2: Development Mode

```bash
# Full development setup (clean, deps, build, setup directories, run)
make dev
```

## Usage

### Basic Usage

```bash
# Process articles from default config (articles.yaml)
./news-writer

# Process articles from custom config file
./news-writer my-articles.yaml

# Process single URL in rewrite mode
./news-writer --rewrite https://example.com/article

# Enable debug logging
./news-writer --debug
```

### Configuration Files

Create an `articles.yaml` file in your project directory:

```yaml
sources:
  - url: "https://example.com/article1"
  - url: "https://example.com/article2"
  - url: "https://example.com/article3"
```

### Advanced Options

```bash
# Use custom writer prompt
./news-writer --writer-prompt custom-writer.md

# Use custom article template
./news-writer --template custom-template.md

# Specify API key directly
./news-writer --api-key your_key_here
```

## Configuration

The system creates a `.news-writer/` directory with default configuration:

### Settings (`settings.yaml`)

```yaml
output_directory: articles
template_path: .news-writer/news-article-template.md
agents:
  planner:
    model: claude-sonnet-4-20250514
    max_tokens: 1000
    temperature: 0.0
    content_max_tokens: 2000
  writer:
    model: claude-sonnet-4-20250514
    max_tokens: 6000
    temperature: 0.2
categories:
  - "Development/Programming"
  - "Technology/Innovation"
  - "Artificial Intelligence/Large Language Models"
```

### Customization

Override any embedded defaults by placing files in `.news-writer/`:

- `writer-system-prompt.md`: Custom writer system prompt
- `planner-system-prompt.md`: Custom planner system prompt
- `news-article-template.md`: Custom article template
- `settings.yaml`: Configuration overrides

## Output Format

Articles are saved as `articles/{date}-{slug}.md` with rich frontmatter:

```markdown
---
title: "React Performance: Essential Optimization Techniques"
source_url: "https://example.com/react-article"
source_domain: "example.com"
created_at: "2024-01-15T10:30:00Z"
draft: false
categories: ["Development/Programming"]
tags: ["React", "Performance", "JavaScript"]
planner_model: "claude-sonnet-4-20250514"
writer_model: "claude-sonnet-4-20250514"
deck: "Key techniques for optimizing React applications including memoization, code splitting, and profiling tools."
---

# React Performance: Essential Optimization Techniques

React apps slow down as they grow. Here are the techniques that make the biggest impact.

## Memoization Techniques

Use React.memo() for expensive components...

## Conclusion

- Start with React.memo() for expensive renders
- Implement route-based code splitting
- Use React DevTools profiler to measure impact
```

## Command Line Options

- `--api-key`: Anthropic API key (or use `ANTHROPIC_API_KEY` env var)
- `--rewrite`: Process single URL and overwrite existing files
- `--writer-prompt`: Path to custom writer prompt file
- `--template`: Path to custom article template file
- `--debug`: Enable detailed logging

## Development

```bash
# Install dependencies
make deps

# Run tests
make test

# Build binary
make build

# Development workflow
make dev

# Clean artifacts
make clean
```

## Dependencies

- Go 1.24+
- [llmkit](https://github.com/aktagon/llmkit) - AI agent framework
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown) - Content conversion

## Rules

1. **Citations**: All articles link to original source in frontmatter
2. **Length**: Optimized for readability (typically 500-1500 words)
3. **Filename**: `articles/{YYYY-MM-DD}-{slug}.md` format
4. **Skip Existing**: Won't overwrite existing files (unless `--rewrite` mode)
5. **Fail Gracefully**: Logs errors and continues processing remaining URLs
6. **Configuration First**: Embedded defaults with easy file-based overrides

