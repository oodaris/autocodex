package orchestrator

import "testing"

func TestSummarizeContentPrefersHeadingsAndBullets(t *testing.T) {
	content := `# Title

Intro line.
- First bullet
- Second bullet

More details here.`
	summary := summarizeContent(content, 2)
	if summary == "" {
		t.Fatalf("expected summary")
	}
	if summary == content {
		t.Fatalf("expected summary to be truncated")
	}
	if !(containsLine(summary, "# Title") && containsLine(summary, "- First bullet")) {
		t.Fatalf("expected heading and bullet in summary, got:\n%s", summary)
	}
}

func TestSummarizeContentFallbacksToFirstLines(t *testing.T) {
	content := "Alpha\n\nBeta\nGamma\n"
	summary := summarizeContent(content, 2)
	if !containsLine(summary, "Alpha") || !containsLine(summary, "Beta") {
		t.Fatalf("expected summary to include first non-empty lines, got:\n%s", summary)
	}
}

func containsLine(content, needle string) bool {
	for _, line := range splitLines(content) {
		if line == needle {
			return true
		}
	}
	return false
}

func splitLines(content string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			lines = append(lines, content[start:i])
			start = i + 1
		}
	}
	if start <= len(content) {
		lines = append(lines, content[start:])
	}
	return lines
}
