package codex

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunnerStreamsOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture not supported on windows")
	}

	tmp := t.TempDir()
	scriptPath := filepath.Join(tmp, "codex")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"exec\" ]; then\n" +
		"  cat >/dev/null\n" +
		"  echo \"out-1\"\n" +
		"  echo \"err-1\" 1>&2\n" +
		"fi\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	stdoutPath := filepath.Join(tmp, "stdout.txt")
	stderrPath := filepath.Join(tmp, "stderr.txt")
	stdoutFile, err := os.Create(stdoutPath)
	if err != nil {
		t.Fatalf("create stdout file: %v", err)
	}
	stderrFile, err := os.Create(stderrPath)
	if err != nil {
		t.Fatalf("create stderr file: %v", err)
	}

	ctx := WithOutputSinks(context.Background(), stdoutFile, stderrFile)
	runner := Runner{
		CLIPath:     scriptPath,
		PromptStdin: true,
		Timeout:     2 * time.Second,
	}

	if _, err := runner.Exec(ctx, "hello"); err != nil {
		t.Fatalf("exec: %v", err)
	}
	if err := stdoutFile.Sync(); err != nil {
		t.Fatalf("sync stdout: %v", err)
	}
	if err := stderrFile.Sync(); err != nil {
		t.Fatalf("sync stderr: %v", err)
	}
	if err := stdoutFile.Close(); err != nil {
		t.Fatalf("close stdout: %v", err)
	}
	if err := stderrFile.Close(); err != nil {
		t.Fatalf("close stderr: %v", err)
	}

	stdoutBytes, err := os.ReadFile(stdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	stderrBytes, err := os.ReadFile(stderrPath)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}

	if !strings.Contains(string(stdoutBytes), "out-1") {
		t.Fatalf("stdout missing expected output")
	}
	if !strings.Contains(string(stderrBytes), "err-1") {
		t.Fatalf("stderr missing expected output")
	}
}
