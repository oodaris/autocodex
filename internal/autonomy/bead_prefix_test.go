package autonomy

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseIssuePrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "autocodex\n", want: "autocodex"},
		{input: "issue_prefix: oodaris", want: "oodaris"},
		{input: "\"demo\"", want: "demo"},
		{input: "", want: ""},
	}
	for _, tt := range tests {
		if got := parseIssuePrefix(tt.input); got != tt.want {
			t.Fatalf("parseIssuePrefix(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveBeadPrefixFromBD(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub not supported on windows")
	}
	tempDir := t.TempDir()
	bdBin := filepath.Join(tempDir, "bd")
	if err := writeBDConfigStub(bdBin, "demo"); err != nil {
		t.Fatalf("write bd stub: %v", err)
	}
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", fmt.Sprintf("%s:%s", tempDir, origPath))

	if got := resolveBeadPrefix(); got != "demo" {
		t.Fatalf("resolveBeadPrefix() = %q, want %q", got, "demo")
	}
}

func TestResolveBeadPrefixFromConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("PATH", tempDir)

	if err := os.MkdirAll(filepath.Join(tempDir, ".beads"), 0o755); err != nil {
		t.Fatalf("mkdir .beads: %v", err)
	}
	configPath := filepath.Join(tempDir, ".beads", "config.yaml")
	if err := os.WriteFile(configPath, []byte("issue_prefix: repo\n"), 0o644); err != nil {
		t.Fatalf("write beads config: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	if got := resolveBeadPrefix(); got != "repo" {
		t.Fatalf("resolveBeadPrefix() = %q, want %q", got, "repo")
	}
}

func TestNormalizeTasksFilePrefix(t *testing.T) {
	tasksFile := TasksFile{
		Tasks: []Task{
			{ID: "autocodex-abc", Dependencies: []string{"autocodex-def"}},
		},
	}
	normalizeTasksFile(&tasksFile, "oodaris")
	if got := tasksFile.Tasks[0].ID; got != "oodaris-abc" {
		t.Fatalf("normalized id = %q, want %q", got, "oodaris-abc")
	}
	if got := tasksFile.Tasks[0].Dependencies[0]; got != "oodaris-def" {
		t.Fatalf("normalized dep = %q, want %q", got, "oodaris-def")
	}
}

func writeBDConfigStub(path, prefix string) error {
	script := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "config" ] && [ "$2" = "get" ] && [ "$3" = "issue_prefix" ]; then
  echo "issue_prefix: %s"
  exit 0
fi
exit 1
`, prefix)
	return os.WriteFile(path, []byte(script), 0o755)
}
