class NewsWriter < Formula
  desc "Command-line tool for distilling web articles into concise summaries using AI agents"
  homepage "https://github.com/aktagon/news-writer"
  # NOTE: The url, version, and sha256 are updated by the github action (.github/workflows/release.yml) automatically
  url "https://github.com/aktagon/news-writer/archive/refs/tags/v0.1.0.tar.gz"
  version "v0.1.0"
  sha256 "PLACEHOLDER_SHA256"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "-o", bin/"news-writer", "."
  end

  def caveats
    <<~EOS
      news-writer distills web articles into concise, insightful summaries.

      Set your Anthropic API key as an environment variable:
        export ANTHROPIC_API_KEY="your-anthropic-key"

      You can add this to your shell profile (~/.zshrc, ~/.bashrc, etc.)
      to make it permanent.

      Create articles directory and configuration:
        mkdir -p articles
        echo 'items:' > articles.yaml
        echo '  - url: "https://example.com/article"' >> articles.yaml

      Use 'news-writer --help' to see available commands and options.
    EOS
  end

  test do
    # Test that the binary was installed and can display help
    assert_match "Usage:", shell_output("#{bin}/news-writer --help 2>&1")
    
    # Test that it fails gracefully without API key
    assert_match "API key required", shell_output("#{bin}/news-writer -config /dev/null 2>&1", 1)
  end
end