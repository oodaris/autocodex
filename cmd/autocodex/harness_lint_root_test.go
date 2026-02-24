package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oodaris/autocodex/internal/config"
)

func TestRunHarnessLintUsesConfigPathFromNestedDirectory(t *testing.T) {
	root := t.TempDir()
	writeHarnessFixture(t, root)

	nested := filepath.Join(root, "nested", "deeper")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("chdir nested: %v", err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	cfg := config.Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	result := runHarnessLint(cfg, filepath.Join("..", "..", "autocodex.yaml"))
	if result.Status != "ok" {
		t.Fatalf("expected harness lint to pass from nested directory, got %s (%s)", result.Status, result.Details)
	}
}

func writeHarnessFixture(t *testing.T, root string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: .git\n"), 0o644); err != nil {
		t.Fatalf("write .git marker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "autocodex.yaml"), []byte("version: v1\nmode: yolo\n"), 0o644); err != nil {
		t.Fatalf("write autocodex.yaml: %v", err)
	}

	var rolePack strings.Builder
	rolePack.WriteString("profile = \"max_capability\"\n\n[agents]\nmax_threads = 4\n")
	for _, role := range harnessLintExpectedRoles {
		rolePack.WriteString(fmt.Sprintf("\n[agents.%s]\nconfig_file = \"agents/%s.toml\"\n", role, role))
	}
	writeFile(t, filepath.Join(root, ".codex", "config.toml"), rolePack.String())

	roleConfig := strings.Join([]string{
		"developer_instructions = \"ok\"",
		"",
		"[features]",
		"multi_agent = true",
		"shell_tool = true",
		"unified_exec = true",
		"shell_snapshot = true",
		"runtime_metrics = true",
		"",
	}, "\n")
	for _, role := range harnessLintExpectedRoles {
		writeFile(t, filepath.Join(root, ".codex", "agents", role+".toml"), roleConfig)
	}

	writeFile(t, filepath.Join(root, "docs", "agents", "autocodex-harness-v2-operating-pack.md"), strings.Join([]string{
		"pattern a",
		"pattern e",
		"non-bypassable gate stack",
		"lifecycle/admission contract",
		"high-impact trigger criteria",
	}, "\n"))
	writeFile(t, filepath.Join(root, "docs", "agents", "harness-evals", "README.md"), "eval docs")
	writeFile(t, filepath.Join(root, "docs", "agents", "harness-evals", "golden-task-catalog.md"), "golden")
	writeFile(t, filepath.Join(root, "docs", "agents", "harness-evals", "failure-mode-catalog.md"), "failure")
	writeFile(t, filepath.Join(root, "scripts", "dev", "harness-cli-preflight.sh"), "#!/usr/bin/env bash\n")
	writeFile(t, filepath.Join(root, "docs", "runbooks", "harness-cli-preflight.md"), strings.Join([]string{
		"harness preflight",
		"scripts/dev/harness-cli-preflight.sh",
		"harness preflight passed",
	}, "\n"))
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
