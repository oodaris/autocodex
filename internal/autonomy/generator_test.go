package autonomy

import (
	"path/filepath"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Add memory docs support": "add-memory-docs-support",
		"  Mixed CASE 123 ":       "mixed-case-123",
		"!!!":                     "task",
	}
	for input, expected := range cases {
		if got := slugify(input); got != expected {
			t.Fatalf("slugify(%q)=%q, want %q", input, got, expected)
		}
	}
}

func TestOutputPaths(t *testing.T) {
	specTemplate := filepath.Join("docs", "specs", "TEMPLATE.md")
	planTemplate := filepath.Join("docs", "plans", "TEMPLATE.md")
	slug := "autocodex-autonomy"

	if got := specOutputPath(specTemplate, slug); got != filepath.Join("docs", "specs", "autocodex-autonomy.md") {
		t.Fatalf("specOutputPath=%q", got)
	}
	if got := planOutputPath(planTemplate, slug); got != filepath.Join("docs", "plans", "autocodex-autonomy-plan.md") {
		t.Fatalf("planOutputPath=%q", got)
	}
}
