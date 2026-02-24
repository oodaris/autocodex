package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHarnessPreflightScriptWarnsOnMalformedHooksJSON(t *testing.T) {
	output, err := runPreflightScriptWithHooksOutput(t, "NOT_JSON_AT_ALL")
	if err != nil {
		t.Fatalf("run preflight script: %v\noutput:\n%s", err, output)
	}
	if !strings.Contains(output, "WARN: unable to parse bd hooks list --json output") {
		t.Fatalf("expected parse warning in output, got:\n%s", output)
	}
	if strings.Contains(output, "PASS: bd hooks are installed") {
		t.Fatalf("expected malformed hooks payload to not be treated as pass, got:\n%s", output)
	}
}

func TestHarnessPreflightScriptParsesCompactHooksJSON(t *testing.T) {
	payload := `{"hooks":[{"Name":"pre-commit","Installed":false},{"Name":"post-merge","Installed":true}]}`
	output, err := runPreflightScriptWithHooksOutput(t, payload)
	if err != nil {
		t.Fatalf("run preflight script: %v\noutput:\n%s", err, output)
	}
	if !strings.Contains(output, "WARN: bd hooks missing (pre-commit)") {
		t.Fatalf("expected missing hooks warning in output, got:\n%s", output)
	}
	if strings.Contains(output, "WARN: unable to parse bd hooks list --json output") {
		t.Fatalf("expected compact JSON to parse cleanly, got:\n%s", output)
	}
}

func runPreflightScriptWithHooksOutput(t *testing.T, hooksOutput string) (string, error) {
	t.Helper()

	repoRoot, err := repositoryRootFromPackageDir()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	fakeBin := t.TempDir()

	bdScript := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

if [[ "${1-}" == "info" && "${2-}" == "--json" ]]; then
  echo '{"repo":"ok"}'
  exit 0
fi
if [[ "${1-}" == "--version" ]]; then
  echo 'bd version 0.56.1'
  exit 0
fi
if [[ "${1-}" == "dolt" && "${2-}" == "test" ]]; then
  echo '{"connection_ok": true}'
  exit 0
fi
if [[ "${1-}" == "dolt" && "${2-}" == "show" ]]; then
  echo '{"backend":"dolt","connection_ok":true,"host":"127.0.0.1","port":3307}'
  exit 0
fi
if [[ "${1-}" == "hooks" && "${2-}" == "list" ]]; then
cat <<'EOF'
%s
EOF
  exit 0
fi

echo "unexpected bd args: $*" >&2
exit 1
`, hooksOutput)
	writeExecutable(t, filepath.Join(fakeBin, "bd"), bdScript)
	writeExecutable(t, filepath.Join(fakeBin, "codex"), "#!/usr/bin/env bash\nexit 0\n")
	writeExecutable(t, filepath.Join(fakeBin, "go"), "#!/usr/bin/env bash\nexit 0\n")

	cmd := exec.Command("bash", filepath.Join(repoRoot, "scripts", "dev", "harness-cli-preflight.sh"))
	cmd.Env = append(os.Environ(), "PATH="+fakeBin+":"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func writeExecutable(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}

func repositoryRootFromPackageDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root := filepath.Clean(filepath.Join(cwd, "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "scripts", "dev", "harness-cli-preflight.sh")); err != nil {
		return "", err
	}
	return root, nil
}
