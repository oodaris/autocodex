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

func TestRunHarnessPreflightChecksUseConfigPathFromNestedDirectory(t *testing.T) {
	root := t.TempDir()
	writeHarnessFixture(t, root)
	if err := os.Remove(filepath.Join(root, "autocodex.yaml")); err != nil {
		t.Fatalf("remove autocodex.yaml: %v", err)
	}
	writeFile(t, filepath.Join(root, "config.example.yaml"), "version: v1\nmode: yolo\n")
	fakeBin := t.TempDir()
	writeExecutable(t, filepath.Join(fakeBin, "codex"), `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1-}" == "--version" ]]; then
  echo 'codex-cli 0.111.0'
  exit 0
fi
if [[ "${1-}" == "features" && "${2-}" == "list" ]]; then
cat <<'EOF'
shell_tool stable true
unified_exec stable true
shell_snapshot stable true
collaboration_modes stable true
multi_agent stable true
runtime_metrics stable true
memory_tool stable false
child_agents_md stable true
EOF
  exit 0
fi
echo "unexpected codex args: $*" >&2
exit 1
`)
	writeExecutable(t, filepath.Join(fakeBin, "bd"), `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1-}" == "--version" ]]; then
  echo 'bd version 0.56.1'
  exit 0
fi
if [[ "${1-}" == "dolt" && "${2-}" == "show" && "${3-}" == "--json" ]]; then
  echo '{"backend":"dolt","database":"beads","host":"127.0.0.1","port":3307,"connection_ok":true}'
  exit 0
fi
echo "unexpected bd args: $*" >&2
exit 1
`)
	t.Setenv("PATH", fakeBin+":"+os.Getenv("PATH"))

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

	resolvedConfig := resolveHarnessConfigPath(config.ResolveConfigPath(), false)
	cfg, err := config.Load(resolvedConfig)
	if err != nil {
		t.Fatalf("load fallback config: %v", err)
	}
	checks, hasFailure := runHarnessPreflightChecks(cfg, resolvedConfig, true)
	if hasFailure {
		t.Fatalf("expected nested preflight checks to pass, got failure set: %#v", checks)
	}
	if result := harnessCheckByName(t, checks, "doctor.git"); result.Status != "ok" {
		t.Fatalf("expected doctor.git ok from nested preflight, got %s (%s)", result.Status, result.Details)
	}
	if result := harnessCheckByName(t, checks, "harness.role-pack"); result.Status != "ok" {
		t.Fatalf("expected harness.role-pack ok from nested preflight, got %s (%s)", result.Status, result.Details)
	}
}

func harnessCheckByName(t *testing.T, checks []harnessCheck, name string) harnessCheck {
	t.Helper()
	for _, check := range checks {
		if check.Name == name {
			return check
		}
	}
	t.Fatalf("missing harness check %q", name)
	return harnessCheck{}
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
	rolePack.WriteString(strings.Join([]string{
		"profile = \"max_capability\"",
		"model = \"gpt-5.4\"",
		"review_model = \"gpt-5.4\"",
		"",
		"[agents]",
		"max_threads = 4",
		"",
		"[profiles.max_capability]",
		"model = \"gpt-5.4\"",
		"review_model = \"gpt-5.4\"",
		"",
		"[profiles.spark]",
		"model = \"gpt-5.3-codex-spark\"",
		"review_model = \"gpt-5.3-codex-spark\"",
		"model_reasoning_summary = \"none\"",
	}, "\n"))
	for _, role := range harnessLintExpectedRoles {
		rolePack.WriteString(fmt.Sprintf("\n[agents.%s]\nconfig_file = \"agents/%s.toml\"\n", role, role))
	}
	writeFile(t, filepath.Join(root, ".codex", "config.toml"), rolePack.String())

	roleConfig := strings.Join([]string{
		"model = \"gpt-5.4\"",
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
